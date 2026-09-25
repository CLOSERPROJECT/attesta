package main

// Attesta's EPCIS user-extension vocabulary, served at /vocabulary/untp/.
//
// UNTP DTE 0.8.0 carries GS1 EPCIS 2.0 events, and EPCIS has no slot for the
// arbitrary values a workflow substep captures. Those ride in an `attesta:`
// extension whose namespace this deployment owns, so the namespace has to
// resolve: JSON-LD term IRIs that 404 are a dead end for integrators, and UNTP
// IDR-14 puts the burden of a meaningful URI space on the resolver operator.
//
// One term table drives three consumers: the inline `@context` emitted on
// every DTE, the JSON-LD context document served here, and the HTML reference
// page. They cannot drift.

import (
	"encoding/json"
	"net/http"
	"strings"
)

// untpVocabularyTerm documents one extension term. TypeMapping is the JSON-LD
// `@type` for the term, set where the plain IRI mapping would lose data.
type untpVocabularyTerm struct {
	Name        string
	Description string
	Example     string
	TypeMapping string
}

// untpSubstepTerms are the properties of the `attesta:substep` object. They
// are scoped under that term, so they only bind inside it.
var untpSubstepTerms = []untpVocabularyTerm{
	{Name: "substepId", Description: "Workflow substep identifier, unique within the stream.", Example: "1.1"},
	{Name: "title", Description: "Substep title as configured in the workflow.", Example: "Intake weighing"},
	{Name: "stepId", Description: "Identifier of the parent workflow step.", Example: "1"},
	{Name: "stepTitle", Description: "Title of the parent workflow step.", Example: "Intake"},
	{Name: "organization", Description: "Slug of the organization that owns the parent step.", Example: "refinery-nl"},
	{Name: "role", Description: "Workflow role slug that completed the substep.", Example: "operator"},
	{Name: "untpRole", Description: "The UNTP PartyRole the workflow role maps to, when one is configured.", Example: "operator"},
	{Name: "completedBy", Description: "Identifier of the account that completed the substep.", Example: "68f2a79b8e7f7d8f3c7c99aa"},
	{
		Name:        "input",
		Description: "The values entered when completing the substep, with their original JSON types. Typed as a JSON literal (`@json`), so arbitrary form payloads survive JSON-LD expansion and canonicalization intact. Attachment metadata is excluded; uploaded files appear in the event's bizTransactionList.",
		Example:     `{"netWeightKg": 18.4}`,
		TypeMapping: "@json",
	},
	{Name: "digest", Description: "SHA-256 over the stored substep payload, prefixed with the algorithm. Matches the substep notarization and the merkle leaf published on the passport page.", Example: "sha256:8968aad7…"},
}

// untpRootTerms are the terms bound directly in the `attesta:` namespace.
var untpRootTerms = []untpVocabularyTerm{
	{Name: "substep", Description: "The workflow substep whose completion produced an EPCIS event. Emitted as `attesta:substep` on every event in a Digital Traceability Event credential."},
	{Name: "attachment", Description: "EPCIS `bizTransaction` type for a file uploaded as evidence on a substep. The transaction value is the file's public download URL."},
	{Name: "categories", Description: "Classification scheme for products whose passport derives `productCategory` from the Attesta stream taxonomy instead of a configured UN CPC code. The classification code is the stream's sub-category slug."},
}

// attestaContextDefinition is the inline @context entry appended to every DTE.
// It declares the prefix and scopes the substep properties, so the extension
// survives JSON-LD expansion without a network fetch of this vocabulary.
func attestaContextDefinition(baseURL string) map[string]interface{} {
	scoped := make(map[string]interface{}, len(untpSubstepTerms))
	for _, term := range untpSubstepTerms {
		iri := attestaContextPrefix + ":" + term.Name
		if term.TypeMapping == "" {
			scoped[term.Name] = iri
			continue
		}
		scoped[term.Name] = map[string]interface{}{"@id": iri, "@type": term.TypeMapping}
	}
	return map[string]interface{}{
		"@version":           1.1,
		attestaContextPrefix: attestaVocabularyIRI(baseURL),
		attestaContextPrefix + ":substep": map[string]interface{}{
			"@id":      attestaContextPrefix + ":substep",
			"@context": scoped,
		},
	}
}

// buildUNTPVocabularyContext is the standalone JSON-LD context document served
// at /vocabulary/untp/. It is the same definition a DTE carries inline, so a
// consumer can reference it instead of relying on the embedded copy.
func buildUNTPVocabularyContext(baseURL string) map[string]interface{} {
	return map[string]interface{}{"@context": attestaContextDefinition(baseURL)}
}

// UNTPVocabularyTermView renders one term row.
type UNTPVocabularyTermView struct {
	Term        string
	IRI         string
	Description string
	Example     string
}

// UNTPVocabularyPageView backs the human-readable vocabulary reference.
type UNTPVocabularyPageView struct {
	PageBase
	NamespaceIRI string
	ContextURL   string
	RootTerms    []UNTPVocabularyTermView
	SubstepTerms []UNTPVocabularyTermView
}

func untpVocabularyTermViews(namespace string, terms []untpVocabularyTerm) []UNTPVocabularyTermView {
	views := make([]UNTPVocabularyTermView, 0, len(terms))
	for _, term := range terms {
		views = append(views, UNTPVocabularyTermView{
			Term:        term.Name,
			IRI:         namespace + term.Name,
			Description: term.Description,
			Example:     term.Example,
		})
	}
	return views
}

func (s *Server) buildUNTPVocabularyView(baseURL string) UNTPVocabularyPageView {
	namespace := attestaVocabularyIRI(baseURL)
	return UNTPVocabularyPageView{
		PageBase:     s.pageBase("untp_vocabulary_body", "", ""),
		NamespaceIRI: namespace,
		ContextURL:   namespace,
		RootTerms:    untpVocabularyTermViews(namespace, untpRootTerms),
		SubstepTerms: untpVocabularyTermViews(namespace, untpSubstepTerms),
	}
}

// untpVocabularyTermExists reports whether a path segment under the namespace
// names a term. Unknown terms 404 rather than silently resolving.
func untpVocabularyTermExists(name string) bool {
	for _, term := range untpRootTerms {
		if term.Name == name {
			return true
		}
	}
	for _, term := range untpSubstepTerms {
		if term.Name == name {
			return true
		}
	}
	return false
}

// prefersJSONLDResponse reports whether the client asked for the machine
// representation of the vocabulary.
func prefersJSONLDResponse(r *http.Request) bool {
	accept := strings.ToLower(strings.TrimSpace(r.Header.Get("Accept")))
	if strings.Contains(accept, "application/ld+json") {
		return true
	}
	return prefersJSONResponse(r)
}

// handleUNTPVocabulary serves the Attesta EPCIS extension namespace: the
// JSON-LD context under content negotiation, the HTML reference otherwise.
// Term IRIs (`…/vocabulary/untp/substep`) resolve to the same documents.
func (s *Server) handleUNTPVocabulary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	switch path := strings.TrimPrefix(r.URL.Path, "/vocabulary/untp"); path {
	case "":
		http.Redirect(w, r, "/vocabulary/untp/", http.StatusMovedPermanently)
		return
	case "/":
	default:
		if !untpVocabularyTermExists(strings.TrimPrefix(path, "/")) {
			http.NotFound(w, r)
			return
		}
	}

	baseURL := requestBaseURL(r)
	if prefersJSONLDResponse(r) {
		w.Header().Set("Content-Type", "application/ld+json")
		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(buildUNTPVocabularyContext(baseURL))
		return
	}
	if err := s.tmpl.ExecuteTemplate(w, "untp_vocabulary.html", s.buildUNTPVocabularyView(baseURL)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
