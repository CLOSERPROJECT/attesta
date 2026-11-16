package org

import (
	"context"
	"fmt"
	"strings"

	"github.com/CLOSERPROJECT/attesta/server/internal/untp"
)

// DPPConfig is the root Org Config passport scheme (one shape per version).
type DPPConfig struct {
	Subject  DPPSubjectConfig  `json:"subject"`
	Identity DPPIdentityConfig `json:"identity"`
	Snapshot DPPSnapshotConfig `json:"snapshot"`
}

// DPPSubjectConfig binds the Lock Job instance id to a Domain entity kind.
type DPPSubjectConfig struct {
	Entity string `json:"entity"`
}

// DPPIdentityConfig maps Digital Link / product copy from Domain via exprs or literals.
type DPPIdentityConfig struct {
	GTIN        any `json:"gtin"`
	Lot         any `json:"lot"`
	Serial      any `json:"serial"`
	ProductName any `json:"productName,omitempty"`
	Description any `json:"description,omitempty"`
}

// DPPSnapshotConfig is the authored projection frozen on lock.
type DPPSnapshotConfig struct {
	Expr string `json:"expr"`
}

// DPPRecord is one issued Passport (org + subject), with frozen snapshot.
type DPPRecord struct {
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

// ErrDPPExists is returned when create-once rejects a second lock for the same subject.
var ErrDPPExists = fmt.Errorf("org: passport already exists")

// ValidateDPPConfig ensures root passport is well-formed and that any
// onSubmit.passport opt-in has a root block.
func ValidateDPPConfig(cfg OrgConfig) error {
	optIn := orgConfigHasDPPOptIn(cfg)
	if cfg.DPP == nil {
		if optIn {
			return fmt.Errorf("org: onSubmit.passport requires root passport")
		}
		return nil
	}
	if err := validateDPPBlock(cfg.DPP); err != nil {
		return err
	}
	return nil
}

func validateDPPBlock(p *DPPConfig) error {
	if p == nil {
		return fmt.Errorf("org: passport block required")
	}
	if strings.TrimSpace(p.Subject.Entity) == "" {
		return fmt.Errorf("org: passport.subject.entity required")
	}
	if p.Identity.GTIN == nil {
		return fmt.Errorf("org: passport.identity.gtin required")
	}
	if p.Identity.Lot == nil {
		return fmt.Errorf("org: passport.identity.lot required")
	}
	if p.Identity.Serial == nil {
		return fmt.Errorf("org: passport.identity.serial required")
	}
	if strings.TrimSpace(p.Snapshot.Expr) == "" {
		return fmt.Errorf("org: passport.snapshot.expr required")
	}
	return nil
}

func orgConfigHasDPPOptIn(cfg OrgConfig) bool {
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
		if taskOnSubmitDPP(map[string]any(t)) {
			return true
		}
	}
	return false
}

// taskOnSubmitDPP is true when the leaf declares a thin onSubmit.passport hook.
func taskOnSubmitDPP(task map[string]any) bool {
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
	Record DPPRecord
	Links  []HttpLinkItem
}

// issueDPP evaluates root passport identity + snapshot, create-once persists,
// and returns Commit links (not HttpPort).
func issueDPP(
	ctx context.Context,
	store DPPStore,
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
	if err := ValidateDPPConfig(cfg); err != nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: err.Error()}
	}
	if cfg.DPP == nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: "onSubmit.passport requires root passport"}
	}
	subjectID = strings.TrimSpace(subjectID)
	if subjectID == "" {
		return passportIssue{}, &CommitError{Kind: "prep", Message: "passport subject id required"}
	}

	entity := strings.TrimSpace(cfg.DPP.Subject.Entity)
	if _, ok := domain.Kind(entity)[subjectID]; !ok {
		return passportIssue{}, &CommitError{
			Kind:    "prep",
			Message: fmt.Sprintf("passport subject %s/%s not found in Domain", entity, subjectID),
		}
	}

	evalBindings := mergeBindings(domainBindings(domain, subjectID, catalogs), bindings)
	evalBindings = mergeBindings(evalBindings, ExprBindings{"subjectId": subjectID})

	identity, err := evalDPPIdentity(cfg.DPP.Identity, evalBindings)
	if err != nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: err.Error()}
	}
	snapshot, err := evalDPPSnapshot(cfg.DPP.Snapshot.Expr, evalBindings)
	if err != nil {
		return passportIssue{}, &CommitError{Kind: "prep", Message: err.Error()}
	}

	link := untp.URL(identity.gtin, identity.lot, identity.serial)
	record := DPPRecord{
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
		if err == ErrDPPExists {
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

func evalDPPIdentity(id DPPIdentityConfig, bindings ExprBindings) (passportIdentityValues, error) {
	gtinRaw, err := evalDPPString(id.GTIN, "passport.identity.gtin", bindings)
	if err != nil {
		return passportIdentityValues{}, err
	}
	gtin, err := untp.NormalizeGTIN(gtinRaw)
	if err != nil {
		return passportIdentityValues{}, fmt.Errorf("passport.identity.gtin: %w", err)
	}
	lot, err := evalDPPString(id.Lot, "passport.identity.lot", bindings)
	if err != nil {
		return passportIdentityValues{}, err
	}
	serial, err := evalDPPString(id.Serial, "passport.identity.serial", bindings)
	if err != nil {
		return passportIdentityValues{}, err
	}
	out := passportIdentityValues{gtin: gtin, lot: lot, serial: serial}
	if id.ProductName != nil {
		name, err := evalDPPString(id.ProductName, "passport.identity.productName", bindings)
		if err != nil {
			return passportIdentityValues{}, err
		}
		out.productName = name
	}
	if id.Description != nil {
		desc, err := evalDPPString(id.Description, "passport.identity.description", bindings)
		if err != nil {
			return passportIdentityValues{}, err
		}
		out.description = desc
	}
	return out, nil
}

func evalDPPString(raw any, label string, bindings ExprBindings) (string, error) {
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

// DPPSnapshotChromeID is the dataReport chrome id injected on Lock Load
// when onSubmit.passport is set — same passport.snapshot.expr as Commit freeze.
const DPPSnapshotChromeID = "passportSnapshot"

// EvalDPPSnapshot evaluates passport.snapshot.expr against Domain bindings.
// Exported for Lock Load recap (same projection as freeze).
func EvalDPPSnapshot(expr string, domain Domain, subjectID string, catalogs map[string]any, extra ExprBindings) (map[string]any, error) {
	bindings := mergeBindings(domainBindings(domain, subjectID, catalogs), extra)
	bindings = mergeBindings(bindings, ExprBindings{"subjectId": subjectID})
	return evalDPPSnapshot(expr, bindings)
}

func evalDPPSnapshot(expr string, bindings ExprBindings) (map[string]any, error) {
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

func clonePassportRecord(in DPPRecord) DPPRecord {
	out := in
	out.Snapshot = cloneAnyMap(in.Snapshot)
	return out
}
