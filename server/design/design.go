package design

import . "goa.design/goa/v3/dsl"

var _ = API("attesta", func() {
	Title("Attesta")
	Description("Design-first contract for Attesta workflow, admin, auth, catalog, Formata Builder, and DPP endpoints.")
	Server("attesta", func() {
		Host("development", func() {
			URI("http://localhost:3000")
		})
	})
})

var _ = Service("workflow", func() {
	Description("Workflow-scoped process, workflow management, and event endpoints under /my/streams.")

	Method("home", func() {
		Description("Public homepage (unauthenticated landing page).")
		Result(Empty)
		HTTP(func() {
			GET("/")
			Response(StatusOK)
		})
	})

	Method("appHome", func() {
		Description("Authenticated app home (stream picker).")
		Result(Empty)
		HTTP(func() {
			GET("/my")
			Response(StatusOK)
		})
	})

	Method("workflowHome", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}")
			Response(StatusOK)
		})
	})

	Method("startProcess", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			POST("/my/streams/{workflow_key}/instance/start")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusInternalServerError)
		})
	})

	Method("deleteWorkflow", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			POST("/my/streams/{workflow_key}/delete")
			Response(StatusSeeOther)
			Response(StatusForbidden)
			Response(StatusNotFound)
			Response(StatusInternalServerError)
		})
	})

	Method("readProcess", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("readProcessContent", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Field(3, "substep", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}/content")
			Param("substep")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("readDownloads", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}/downloads")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("downloadAllFiles", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}/files.zip")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("notarizedJSON", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}/notarized.json")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("merkleJSON", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}/merkle.json")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("completeSubstep", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Field(3, "substep_id", String)
			Required("workflow_key", "process_id", "substep_id")
		})
		Result(Empty)
		HTTP(func() {
			POST("/my/streams/{workflow_key}/instance/{process_id}/substep/{substep_id}/complete")
			Response(StatusOK)
			Response(StatusConflict)
			Response(StatusForbidden)
			Response(StatusNotFound)
		})
	})

	Method("downloadAttachmentFile", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Field(3, "attachment_id", String)
			Field(4, "inline", String)
			Required("workflow_key", "process_id", "attachment_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/instance/{process_id}/attachment/{attachment_id}/file")
			Param("inline")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("events", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/streams/{workflow_key}/events")
			Response(StatusOK)
			Response(StatusBadRequest)
		})
	})
})

var CatalogResponse = Type("CatalogResponse", func() {
	Field(1, "organizations", ArrayOf(CatalogOrganization))
	Field(2, "roles", ArrayOf(CatalogRole))
	Required("organizations", "roles")
})

var CatalogOrganization = Type("CatalogOrganization", func() {
	Field(1, "name", String)
	Field(2, "slug", String)
	Required("name", "slug")
})

var CatalogRole = Type("CatalogRole", func() {
	Field(1, "orgSlug", String)
	Field(2, "name", String)
	Field(3, "slug", String)
	Field(4, "color", String)
	Field(5, "border", String)
	Required("orgSlug", "name", "slug", "color", "border")
})

var _ = Service("catalog", func() {
	Description("Authenticated API endpoints used by the Formata Builder and other admin clients.")

	Method("publicCatalog", func() {
		Result(CatalogResponse)
		HTTP(func() {
			GET("/api/catalog")
			Response(StatusOK)
			Response(StatusForbidden)
			Response(StatusUnauthorized)
			Response(StatusInternalServerError)
			Response(StatusBadGateway)
		})
	})
})

var _ = Service("auth", func() {
	Description("Account, session, invite, and password recovery pages.")

	Method("loginPage", func() {
		Result(Empty)
		HTTP(func() {
			GET("/login")
			Response(StatusOK)
			Response(StatusSeeOther)
		})
	})

	Method("login", func() {
		Result(Empty)
		HTTP(func() {
			POST("/login")
			Response(StatusSeeOther)
			Response(StatusUnauthorized)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("signupPage", func() {
		Result(Empty)
		HTTP(func() {
			GET("/signup")
			Response(StatusOK)
			Response(StatusSeeOther)
			Response(StatusNotFound)
		})
	})

	Method("signup", func() {
		Result(Empty)
		HTTP(func() {
			POST("/signup")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusNotFound)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("logout", func() {
		Result(Empty)
		HTTP(func() {
			POST("/logout")
			Response(StatusSeeOther)
			Response(StatusMethodNotAllowed)
		})
	})

	Method("acceptInvite", func() {
		Payload(func() {
			Field(1, "teamId", String)
			Field(2, "membershipId", String)
			Field(3, "userId", String)
			Field(4, "secret", String)
			Required("teamId", "membershipId", "userId", "secret")
		})
		Result(Empty)
		HTTP(func() {
			GET("/invite/accept")
			Param("teamId")
			Param("membershipId")
			Param("userId")
			Param("secret")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusNotFound)
			Response(StatusInternalServerError)
		})
	})

	Method("invitePasswordPage", func() {
		Result(Empty)
		HTTP(func() {
			GET("/invite/password")
			Response(StatusOK)
			Response(StatusNotFound)
			Response(StatusUnauthorized)
		})
	})

	Method("setInvitePassword", func() {
		Result(Empty)
		HTTP(func() {
			POST("/invite/password")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusNotFound)
			Response(StatusUnauthorized)
			Response(StatusInternalServerError)
		})
	})

	Method("resetPage", func() {
		Result(Empty)
		HTTP(func() {
			GET("/reset")
			Response(StatusOK)
		})
	})

	Method("requestReset", func() {
		Result(Empty)
		HTTP(func() {
			POST("/reset")
			Response(StatusOK)
			Response(StatusBadRequest)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("resetConfirmPage", func() {
		Payload(func() {
			Field(1, "userId", String)
			Field(2, "secret", String)
			Required("userId", "secret")
		})
		Result(Empty)
		HTTP(func() {
			GET("/reset/confirm")
			Param("userId")
			Param("secret")
			Response(StatusOK)
			Response(StatusBadRequest)
		})
	})

	Method("confirmReset", func() {
		Payload(func() {
			Field(1, "userId", String)
			Field(2, "secret", String)
			Required("userId", "secret")
		})
		Result(Empty)
		HTTP(func() {
			POST("/reset/confirm")
			Param("userId")
			Param("secret")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusInternalServerError)
		})
	})
})

var _ = Service("admin", func() {
	Description("Platform and organization administration endpoints.")

	Method("platformOrgs", func() {
		Payload(func() {
			Field(1, "q", String)
			Field(2, "page", Int)
		})
		Result(Empty)
		HTTP(func() {
			GET("/admin/orgs")
			Param("q")
			Param("page")
			Response(StatusOK)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusInternalServerError)
		})
	})

	Method("platformOrgAction", func() {
		Result(Empty)
		HTTP(func() {
			POST("/admin/orgs")
			Response(StatusOK)
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusInternalServerError)
		})
	})

	Method("platformOrgLogo", func() {
		Payload(func() {
			Field(1, "logo_id", String)
			Required("logo_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/admin/orgs/logo/{logo_id}")
			Response(StatusOK)
			Response(StatusNotFound)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
		})
	})

	Method("orgAdminProfile", func() {
		Result(Empty)
		HTTP(func() {
			GET("/my/organization/profile")
			Response(StatusOK)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("orgAdminMembers", func() {
		Result(Empty)
		HTTP(func() {
			GET("/my/organization/members")
			Response(StatusOK)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("orgAdminUsers", func() {
		Result(Empty)
		HTTP(func() {
			// Legacy entry: GET redirects to /my/organization/profile.
			GET("/my/organization/users")
			Response(StatusSeeOther)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("orgAdminUserAction", func() {
		Result(Empty)
		HTTP(func() {
			POST("/my/organization/users")
			Response(StatusOK)
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("orgAdminRoles", func() {
		Result(Empty)
		HTTP(func() {
			GET("/my/organization/roles")
			Response(StatusOK)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusInternalServerError)
		})
	})

	Method("orgAdminRoleAction", func() {
		Result(Empty)
		HTTP(func() {
			POST("/my/organization/roles")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
			Response(StatusServiceUnavailable)
			Response(StatusInternalServerError)
		})
	})

	Method("orgAdminLogo", func() {
		Payload(func() {
			Field(1, "logo_id", String)
			Required("logo_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/organization/logo/{logo_id}")
			Response(StatusOK)
			Response(StatusNotFound)
			Response(StatusUnauthorized)
			Response(StatusForbidden)
		})
	})

	Method("organizationLogo", func() {
		Payload(func() {
			Field(1, "org_slug", String)
			Required("org_slug")
		})
		Result(Empty)
		HTTP(func() {
			GET("/organization/logo/{org_slug}")
			Response(StatusOK)
			Response(StatusNotFound)
			Response(StatusUnauthorized)
		})
	})
})

var _ = Service("formata_builder", func() {
	Description("Embedded Formata Builder UI and stream persistence endpoints.")

	Method("builderApp", func() {
		Result(Empty)
		HTTP(func() {
			GET("/my/organization/formata-builder")
			Response(StatusOK)
			Response(StatusForbidden)
			Response(StatusUnauthorized)
			Response(StatusBadGateway)
			Response(StatusNotFound)
			Response(StatusInternalServerError)
		})
	})

	Method("builderAsset", func() {
		Payload(func() {
			Field(1, "asset_path", String)
			Required("asset_path")
		})
		Result(Empty)
		HTTP(func() {
			GET("/my/organization/formata-builder/{asset_path}")
			Response(StatusOK)
			Response(StatusForbidden)
			Response(StatusUnauthorized)
			Response(StatusBadGateway)
			Response(StatusNotFound)
			Response(StatusInternalServerError)
		})
	})

	Method("loadStream", func() {
		Payload(func() {
			Field(1, "stream_id", String)
			Required("stream_id")
		})
		Result(Any)
		HTTP(func() {
			GET("/my/organization/formata-builder/stream/{stream_id}")
			Response(StatusOK)
			Response(StatusForbidden)
			Response(StatusUnauthorized)
			Response(StatusBadGateway)
			Response(StatusNotFound)
			Response(StatusInternalServerError)
		})
	})

	Method("saveStream", func() {
		Payload(func() {
			Field(1, "body", String)
			Field(2, "new", Boolean)
			Field(3, "stream", String)
			Required("body")
		})
		Result(Empty)
		HTTP(func() {
			POST("/my/organization/formata-builder")
			Param("stream")
			Param("new")
			Body("body")
			Response(StatusNoContent)
			Response(StatusBadRequest)
			Response(StatusForbidden)
			Response(StatusUnauthorized)
			Response(StatusConflict)
			Response(StatusRequestEntityTooLarge)
			Response(StatusBadGateway)
			Response(StatusInternalServerError)
		})
	})
})

var _ = Service("dpp", func() {
	Description("GS1 Digital Link endpoints for DPP.")

	Method("digitalLink", func() {
		Payload(func() {
			Field(1, "gtin", String)
			Field(2, "lot", String)
			Field(3, "serial", String)
			Field(4, "format", String)
			Required("gtin", "lot", "serial")
		})
		Result(Empty)
		HTTP(func() {
			GET("/01/{gtin}/10/{lot}/21/{serial}")
			Param("format")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("downloadAttachmentFile", func() {
		Description("Public attachment download for a DPP Digital Link.")
		Payload(func() {
			Field(1, "gtin", String)
			Field(2, "lot", String)
			Field(3, "serial", String)
			Field(4, "attachment_id", String)
			Field(5, "inline", String)
			Required("gtin", "lot", "serial", "attachment_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/01/{gtin}/10/{lot}/21/{serial}/attachment/{attachment_id}/file")
			Param("inline")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})
})
