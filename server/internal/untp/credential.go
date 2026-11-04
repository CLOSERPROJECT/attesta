package untp

import (
	"net/url"
	"strings"
	"time"
)

// ProductIdentity is GS1 item identity plus display copy for UNTP Product subjects.
type ProductIdentity struct {
	Name        string
	Description string
	OwnerName   string
	BatchNumber string
	ItemNumber  string
	GTIN        string
	ValidFrom   time.Time
}

// AttachmentRef is a public attachment referenced from a ModifyEvent.
type AttachmentRef struct {
	ID          string
	Filename    string
	ContentType string
}

// EventInput is one completed lifecycle checkpoint for a DTE ModifyEvent.
type EventInput struct {
	ID          string
	Title       string
	EventDate   time.Time
	ActorID     string
	ActorRole   string
	Attachments []AttachmentRef
}

// DPPInput feeds BuildDPPCredential without Stream or living engine types.
type DPPInput struct {
	BaseURL         string
	Link            string
	Product         ProductIdentity
	Characteristics map[string]interface{}
}

// DTEInput feeds BuildDTECredential without Stream or living engine types.
type DTEInput struct {
	BaseURL string
	Link    string
	Product ProductIdentity
	Scheme  string
	Events  []EventInput
}

func FormatRFC3339UTC(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func issuer(baseURL, ownerName string) CredentialIssuer {
	return CredentialIssuer{
		Type: []string{"CredentialIssuer"},
		ID:   baseURL,
		Name: strings.TrimSpace(ownerName),
	}
}

func BuildDPPCredential(in DPPInput) Credential {
	productID := AbsoluteURL(in.BaseURL, in.Link)
	name := strings.TrimSpace(in.Product.Name)
	characteristics := make(map[string]interface{}, len(in.Characteristics)+1)
	for key, value := range in.Characteristics {
		characteristics[key] = value
	}
	if element := GS1ElementString(in.Product.GTIN, in.Product.BatchNumber, in.Product.ItemNumber); element != "" {
		characteristics["gs1Element"] = element
	}
	subject := Product{
		Type:          []string{"Product"},
		ID:            productID,
		Name:          name,
		Description:   strings.TrimSpace(in.Product.Description),
		IDScheme:      IdentifierScheme{Type: []string{"IdentifierScheme"}, ID: GS1DigitalLinkScheme, Name: "GS1 Digital Link"},
		BatchNumber:   strings.TrimSpace(in.Product.BatchNumber),
		ItemNumber:    strings.TrimSpace(in.Product.ItemNumber),
		IDGranularity: "item",
		RelatedDocument: []Link{{
			LinkURL:   productID + "/events",
			LinkName:  "Digital Traceability Events",
			MediaType: "application/json",
			LinkType:  LinkTypeDTE,
		}},
		Characteristics: characteristics,
	}
	credential := Credential{
		Type:              []string{"DigitalProductPassport", "VerifiableCredential"},
		Context:           []string{VCCredentialsContext, VocabularyContext},
		ID:                productID,
		Issuer:            issuer(in.BaseURL, in.Product.OwnerName),
		Name:              "Digital Product Passport — " + name,
		CredentialSubject: subject,
		ValidFrom:         FormatRFC3339UTC(in.Product.ValidFrom),
	}
	return credential
}

func BuildModifyEvents(baseURL, link string, product ProductIdentity, schemeName string, events []EventInput) []ModifyEvent {
	productID := AbsoluteURL(baseURL, link)
	productRef := ProductRef{
		Type:          []string{"Product"},
		ID:            productID,
		Name:          strings.TrimSpace(product.Name),
		BatchNumber:   strings.TrimSpace(product.BatchNumber),
		ItemNumber:    strings.TrimSpace(product.ItemNumber),
		IDGranularity: "item",
	}
	scheme := strings.TrimSpace(schemeName)
	out := make([]ModifyEvent, 0, len(events))
	for _, in := range events {
		id := strings.TrimSpace(in.ID)
		title := strings.TrimSpace(in.Title)
		event := ModifyEvent{
			Type: []string{"ModifyEvent", "LifecycleEvent"},
			ID:   productID + "/events#" + url.QueryEscape(id),
			Name: title,
			ActivityType: Classification{
				Code:       id,
				Name:       title,
				SchemeID:   baseURL,
				SchemeName: scheme,
			},
			ModifiedProduct: []EventProduct{{
				Product:     productRef,
				Disposition: "active",
			}},
			EventDate: FormatRFC3339UTC(in.EventDate),
		}
		if actorID := strings.TrimSpace(in.ActorID); actorID != "" {
			event.RelatedParty = []PartyRole{{
				Role: strings.TrimSpace(in.ActorRole),
				Party: Party{
					Type: []string{"Party"},
					ID:   baseURL + "/#actor=" + url.QueryEscape(actorID),
					Name: actorID,
				},
			}}
		}
		for _, meta := range in.Attachments {
			if strings.TrimSpace(meta.ID) == "" {
				continue
			}
			event.RelatedDocument = append(event.RelatedDocument, Link{
				LinkURL:   productID + "/attachment/" + url.PathEscape(meta.ID) + "/file",
				LinkName:  meta.Filename,
				MediaType: meta.ContentType,
			})
		}
		out = append(out, event)
	}
	return out
}

func BuildDTECredential(in DTEInput) Credential {
	productID := AbsoluteURL(in.BaseURL, in.Link)
	name := strings.TrimSpace(in.Product.Name)
	return Credential{
		Type:              []string{"DigitalTraceabilityEvent", "VerifiableCredential"},
		Context:           []string{VCCredentialsContext, VocabularyContext},
		ID:                productID + "/events",
		Issuer:            issuer(in.BaseURL, in.Product.OwnerName),
		Name:              "Digital Traceability Events — " + name,
		CredentialSubject: BuildModifyEvents(in.BaseURL, in.Link, in.Product, in.Scheme, in.Events),
		ValidFrom:         FormatRFC3339UTC(in.Product.ValidFrom),
	}
}
