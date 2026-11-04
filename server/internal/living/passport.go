package living

import (
	"context"
	"fmt"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// PassportConfig is the root Org Config passport scheme (one shape per version).
type PassportConfig struct {
	Subject  PassportSubjectConfig  `json:"subject"`
	Identity PassportIdentityConfig `json:"identity"`
	Snapshot PassportSnapshotConfig `json:"snapshot"`
}

// PassportSubjectConfig binds the Lock Job instance id to a Domain entity kind.
type PassportSubjectConfig struct {
	Entity string `json:"entity"`
}

// PassportIdentityConfig maps Digital Link / product copy from Domain via exprs or literals.
type PassportIdentityConfig struct {
	GTIN        any `json:"gtin"`
	Lot         any `json:"lot"`
	Serial      any `json:"serial"`
	ProductName any `json:"productName,omitempty"`
	Description any `json:"description,omitempty"`
}

// PassportSnapshotConfig is the authored projection frozen on lock.
type PassportSnapshotConfig struct {
	Expr string `json:"expr"`
}

// PassportRecord is one issued Passport (org + subject), with frozen snapshot.
type PassportRecord struct {
	Org           string         `json:"org"`
	SubjectID     string         `json:"subjectId"`
	GTIN          string         `json:"gtin"`
	Lot           string         `json:"lot"`
	Serial        string         `json:"serial"`
	ProductName   string         `json:"productName,omitempty"`
	Description   string         `json:"description,omitempty"`
	Snapshot      map[string]any `json:"snapshot"`
	ConfigVersion string         `json:"configVersion,omitempty"`
	IssuedAt      string         `json:"issuedAt"`
	DigitalLink   string         `json:"digitalLink"`
}

// ErrPassportExists is returned when create-once rejects a second lock for the same subject.
var ErrPassportExists = fmt.Errorf("living: passport already exists")

// ValidatePassportConfig ensures root passport is well-formed and that any
// onSubmit.passport opt-in has a root block.
func ValidatePassportConfig(cfg OrgConfig) error {
	optIn := orgConfigHasPassportOptIn(cfg)
	if cfg.Passport == nil {
		if optIn {
			return fmt.Errorf("living: onSubmit.passport requires root passport")
		}
		return nil
	}
	if err := validatePassportBlock(cfg.Passport); err != nil {
		return err
	}
	return nil
}

func validatePassportBlock(p *PassportConfig) error {
	if p == nil {
		return fmt.Errorf("living: passport block required")
	}
	if strings.TrimSpace(p.Subject.Entity) == "" {
		return fmt.Errorf("living: passport.subject.entity required")
	}
	if p.Identity.GTIN == nil {
		return fmt.Errorf("living: passport.identity.gtin required")
	}
	if p.Identity.Lot == nil {
		return fmt.Errorf("living: passport.identity.lot required")
	}
	if p.Identity.Serial == nil {
		return fmt.Errorf("living: passport.identity.serial required")
	}
	if strings.TrimSpace(p.Snapshot.Expr) == "" {
		return fmt.Errorf("living: passport.snapshot.expr required")
	}
	return nil
}

func orgConfigHasPassportOptIn(cfg OrgConfig) bool {
	if cfg.Jobs.Values == nil {
		return false
	}
	for _, key := range cfg.Jobs.Keys {
		job, ok := AsJob(cfg.Jobs.Values[key])
		if !ok {
			continue
		}
		if tasksHavePassportOptIn(job.Tasks) {
			return true
		}
	}
	return false
}

func tasksHavePassportOptIn(tasks []TaskConfig) bool {
	for _, t := range tasks {
		if t.IsGroup() {
			if tasksHavePassportOptIn(t.ChildTasks()) {
				return true
			}
			continue
		}
		if taskOnSubmitPassport(map[string]any(t)) {
			return true
		}
	}
	return false
}

// taskOnSubmitPassport is true when the leaf declares a thin onSubmit.passport hook.
func taskOnSubmitPassport(task map[string]any) bool {
	onSubmit, _ := task["onSubmit"].(map[string]any)
	if onSubmit == nil {
		return false
	}
	raw, ok := onSubmit["passport"]
	if !ok || raw == nil {
		return false
	}
	switch v := raw.(type) {
	case bool:
		return v
	case map[string]any:
		return true
	default:
		return true
	}
}

type passportIssue struct {
	Record PassportRecord
	Links  []HttpLinkItem
}

// issuePassport evaluates root passport identity + snapshot, create-once persists,
// and returns public link items for LinkStore / TaskRecap (not HttpPort; Commit
// omits them from response Links so TaskForm does not flash the Passport URL).
func issuePassport(
	ctx context.Context,
	store PassportStore,
	cfg OrgConfig,
	domain Domain,
	subjectID string,
	catalogs map[string]any,
	bindings ExprBindings,
	issuedAt string,
) (passportIssue, error) {
	if store == nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: "passport store not configured"}
	}
	if err := ValidatePassportConfig(cfg); err != nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: err.Error()}
	}
	if cfg.Passport == nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: "onSubmit.passport requires root passport"}
	}
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return passportIssue{}, &CommitError{Kind: "prep", Message: "passport subject id required"}
	}

	entity := strings.TrimSpace(cfg.Passport.Subject.Entity)
	if _, ok := domain.Kind(entity)[subjectID]; !ok {
		return passportIssue{}, &CommitError{
			Kind:    "prep",
			Message: fmt.Sprintf("passport subject %s/%s not found in Domain", entity, subjectID),
		}
	}

	evalBindings := mergeBindings(domainBindings(domain, subjectID, catalogs), bindings)
	evalBindings = mergeBindings(evalBindings, ExprBindings{"subjectId": subjectID})

	identity, err := evalPassportIdentity(cfg.Passport.Identity, evalBindings)
	if err != nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: err.Error()}
	}
	snapshot, err := evalPassportSnapshot(cfg.Passport.Snapshot.Expr, evalBindings)
	if err != nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: err.Error()}
	}

	link := untp.URL(identity.gtin, identity.lot, identity.serial)
	record := PassportRecord{
		Org:           strings.TrimSpace(cfg.Org),
		SubjectID:     subjectID,
		GTIN:          identity.gtin,
		Lot:           identity.lot,
		Serial:        identity.serial,
		ProductName:   identity.productName,
		Description:   identity.description,
		Snapshot:      snapshot,
		ConfigVersion: strings.TrimSpace(cfg.Version),
		IssuedAt:      issuedAt,
		DigitalLink:   link,
	}
	if err := store.Create(ctx, record); err != nil {
		if err == ErrPassportExists {
			return passportIssue{}, &CommitError{Kind: "passport-exists", Message: "passport already issued for subject"}
		}
		return passportIssue{}, err
	}
	return passportIssue{
		Record: record,
		Links: []HttpLinkItem{{
			Label:  "Digital Product Passport",
			URL:    link,
			Action: HttpActionLink,
		}},
	}, nil
}

type passportIdentityValues struct {
	gtin        string
	lot         string
	serial      string
	productName string
	description string
}

func evalPassportIdentity(id PassportIdentityConfig, bindings ExprBindings) (passportIdentityValues, error) {
	gtinRaw, err := evalPassportString(id.GTIN, "passport.identity.gtin", bindings)
	if err != nil {
		return passportIdentityValues{}, err
	}
	gtin, err := untp.NormalizeGTIN(gtinRaw)
	if err != nil {
		return passportIdentityValues{}, fmt.Errorf("passport.identity.gtin: %w", err)
	}
	lot, err := evalPassportString(id.Lot, "passport.identity.lot", bindings)
	if err != nil {
		return passportIdentityValues{}, err
	}
	serial, err := evalPassportString(id.Serial, "passport.identity.serial", bindings)
	if err != nil {
		return passportIdentityValues{}, err
	}
	out := passportIdentityValues{gtin: gtin, lot: lot, serial: serial}
	if id.ProductName != nil {
		name, err := evalPassportString(id.ProductName, "passport.identity.productName", bindings)
		if err != nil {
			return passportIdentityValues{}, err
		}
		out.productName = name
	}
	if id.Description != nil {
		desc, err := evalPassportString(id.Description, "passport.identity.description", bindings)
		if err != nil {
			return passportIdentityValues{}, err
		}
		out.description = desc
	}
	return out, nil
}

func evalPassportString(raw any, label string, bindings ExprBindings) (string, error) {
	got, err := evalScalar(raw, map[string]any{}, bindings)
	if err != nil {
		return "", fmt.Errorf("%s: %w", label, err)
	}
	if !got.Defined {
		return "", fmt.Errorf("%s must evaluate to a non-empty string", label)
	}
	s, ok := got.Value.(string)
	if !ok || strings.TrimSpace(s) == "" {
		return "", fmt.Errorf("%s must evaluate to a non-empty string", label)
	}
	return strings.TrimSpace(s), nil
}

// PassportSnapshotChromeID is the dataReport chrome id injected on Lock Load
// when onSubmit.passport is set — same passport.snapshot.expr as Commit freeze.
const PassportSnapshotChromeID = "passportSnapshot"

// EvalPassportSnapshot evaluates passport.snapshot.expr against Domain bindings.
// Exported for Lock Load recap (same projection as freeze).
func EvalPassportSnapshot(expr string, domain Domain, subjectID string, catalogs map[string]any, extra ExprBindings) (map[string]any, error) {
	bindings := mergeBindings(domainBindings(domain, subjectID, catalogs), extra)
	bindings = mergeBindings(bindings, ExprBindings{"subjectId": subjectID})
	return evalPassportSnapshot(expr, bindings)
}

func evalPassportSnapshot(expr string, bindings ExprBindings) (map[string]any, error) {
	got, err := evaluateExpr(expr, map[string]any{}, bindings)
	if err != nil {
		return nil, fmt.Errorf("passport.snapshot.expr: %w", err)
	}
	m, ok := got.(map[string]any)
	if !ok || m == nil {
		return nil, fmt.Errorf("passport.snapshot.expr must evaluate to an object")
	}
	return cloneAnyMap(m), nil
}

func cloneAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func clonePassportRecord(in PassportRecord) PassportRecord {
	out := in
	out.Snapshot = cloneAnyMap(in.Snapshot)
	return out
}
