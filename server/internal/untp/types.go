package untp

const (
	VCCredentialsContext = "https://www.w3.org/ns/credentials/v2"
	VocabularyContext    = "https://vocabulary.uncefact.org/untp/"
	LinkTypeDTE          = "https://test.uncefact.org/vocabulary/linkTypes/dte"
	GS1DigitalLinkScheme = "https://id.gs1.org"
)

// Credential is the W3C VC 2.0 envelope shared by DPP and DTE credentials.
type Credential struct {
	Type              []string         `json:"type"`
	Context           []string         `json:"@context"`
	ID                string           `json:"id"`
	Issuer            CredentialIssuer `json:"issuer"`
	Name              string           `json:"name,omitempty"`
	ValidFrom         string           `json:"validFrom,omitempty"`
	CredentialSubject interface{}      `json:"credentialSubject"`
}

type CredentialIssuer struct {
	Type []string `json:"type"`
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
}

// Product is the credential subject of a Digital Product Passport.
type Product struct {
	Type            []string               `json:"type"`
	ID              string                 `json:"id"`
	Name            string                 `json:"name,omitempty"`
	Description     string                 `json:"description,omitempty"`
	IDScheme        IdentifierScheme       `json:"idScheme,omitzero"`
	BatchNumber     string                 `json:"batchNumber,omitempty"`
	ItemNumber      string                 `json:"itemNumber,omitempty"`
	IDGranularity   string                 `json:"idGranularity,omitempty"`
	RelatedDocument []Link                 `json:"relatedDocument,omitempty"`
	Characteristics map[string]interface{} `json:"characteristics,omitempty"`
}

type IdentifierScheme struct {
	Type []string `json:"type"`
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
}

// Link is a UNTP core vocabulary Link (relatedDocument entries).
type Link struct {
	LinkURL   string `json:"linkURL"`
	LinkName  string `json:"linkName,omitempty"`
	MediaType string `json:"mediaType,omitempty"`
	LinkType  string `json:"linkType,omitempty"`
}

// ModifyEvent is a UNTP ModifyEvent lifecycle event.
type ModifyEvent struct {
	Type            []string       `json:"type"`
	ID              string         `json:"id"`
	Name            string         `json:"name,omitempty"`
	EventDate       string         `json:"eventDate,omitempty"`
	ActivityType    Classification `json:"activityType,omitzero"`
	ModifiedProduct []EventProduct `json:"modifiedProduct,omitempty"`
	RelatedParty    []PartyRole    `json:"relatedParty,omitempty"`
	RelatedDocument []Link         `json:"relatedDocument,omitempty"`
}

type Classification struct {
	Code       string `json:"code,omitempty"`
	Name       string `json:"name,omitempty"`
	SchemeID   string `json:"schemeId,omitempty"`
	SchemeName string `json:"schemeName,omitempty"`
}

type EventProduct struct {
	Product     ProductRef `json:"product,omitzero"`
	Disposition string     `json:"disposition,omitempty"`
}

// ProductRef is the product identification carried inside event products.
type ProductRef struct {
	Type          []string `json:"type"`
	ID            string   `json:"id"`
	Name          string   `json:"name,omitempty"`
	BatchNumber   string   `json:"batchNumber,omitempty"`
	ItemNumber    string   `json:"itemNumber,omitempty"`
	IDGranularity string   `json:"idGranularity,omitempty"`
}

type PartyRole struct {
	Role  string `json:"role"`
	Party Party  `json:"party,omitzero"`
}

type Party struct {
	Type []string `json:"type"`
	ID   string   `json:"id"`
	Name string   `json:"name,omitempty"`
}

// Linkset is an RFC 9264 linkset as returned by the identity resolver.
type Linkset struct {
	Linkset []LinksetContext `json:"linkset"`
}

type LinksetContext struct {
	Anchor string          `json:"anchor"`
	DPP    []LinksetTarget `json:"dpp,omitempty"`
	DTE    []LinksetTarget `json:"dte,omitempty"`
	PIP    []LinksetTarget `json:"pip,omitempty"`
}

type LinksetTarget struct {
	Href     string   `json:"href"`
	Type     string   `json:"type"`
	Title    string   `json:"title,omitempty"`
	Hreflang []string `json:"hreflang,omitempty"`
}
