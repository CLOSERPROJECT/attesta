package design

import . "goa.design/goa/v3/dsl"

var _ = API("attesta", func() {
	Title("Attesta")
	Description("Design-first contract for Attesta workflow, process, backoffice, and DPP endpoints.")
	Server("attesta", func() {
		Host("development", func() {
			URI("http://localhost:3000")
		})
	})
})

var _ = Service("workflow", func() {
	Description("Workflow-scoped process, backoffice, impersonation, and event endpoints.")

	Method("workflowHome", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}")
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
			POST("/w/{workflow_key}/process/start")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
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
			GET("/w/{workflow_key}/process/{process_id}")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("readTimeline", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Required("workflow_key", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/process/{process_id}/timeline")
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
			GET("/w/{workflow_key}/process/{process_id}/downloads")
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
			GET("/w/{workflow_key}/process/{process_id}/files.zip")
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
			GET("/w/{workflow_key}/process/{process_id}/notarized.json")
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
			GET("/w/{workflow_key}/process/{process_id}/merkle.json")
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
			POST("/w/{workflow_key}/process/{process_id}/substep/{substep_id}/complete")
			Response(StatusOK)
			Response(StatusConflict)
			Response(StatusForbidden)
			Response(StatusNotFound)
		})
	})

	Method("downloadSubstepFile", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "process_id", String)
			Field(3, "substep_id", String)
			Required("workflow_key", "process_id", "substep_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/process/{process_id}/substep/{substep_id}/file")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("backofficeLanding", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/backoffice")
			Response(StatusOK)
		})
	})

	Method("backofficeRole", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "role", String)
			Required("workflow_key", "role")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/backoffice/{role}")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("backofficeRolePartial", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "role", String)
			Required("workflow_key", "role")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/backoffice/{role}/partial")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("backofficeRoleProcess", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Field(2, "role", String)
			Field(3, "process_id", String)
			Required("workflow_key", "role", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/backoffice/{role}/process/{process_id}")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("impersonate", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			POST("/w/{workflow_key}/impersonate")
			Response(StatusSeeOther)
			Response(StatusBadRequest)
		})
	})

	Method("events", func() {
		Payload(func() {
			Field(1, "workflow_key", String)
			Required("workflow_key")
		})
		Result(Empty)
		HTTP(func() {
			GET("/w/{workflow_key}/events")
			Response(StatusOK)
			Response(StatusBadRequest)
		})
	})
})

var _ = Service("legacy", func() {
	Description("Legacy compatibility endpoints.")

	Method("legacyStart", func() {
		Result(Empty)
		HTTP(func() {
			POST("/process/start")
			Response(StatusBadRequest)
		})
	})

	Method("legacyProcess", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Required("process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}")
			Response(StatusSeeOther)
			Response(StatusNotFound)
		})
	})

	Method("legacyTimeline", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Required("process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}/timeline")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyDownloads", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Required("process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}/downloads")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyFilesZip", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Required("process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}/files.zip")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyNotarized", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Required("process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}/notarized.json")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyMerkle", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Required("process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}/merkle.json")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyCompleteSubstep", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Field(2, "substep_id", String)
			Required("process_id", "substep_id")
		})
		Result(Empty)
		HTTP(func() {
			POST("/process/{process_id}/substep/{substep_id}/complete")
			Response(StatusBadRequest)
		})
	})

	Method("legacySubstepFile", func() {
		Payload(func() {
			Field(1, "process_id", String)
			Field(2, "substep_id", String)
			Required("process_id", "substep_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/process/{process_id}/substep/{substep_id}/file")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyBackoffice", func() {
		Result(Empty)
		HTTP(func() {
			GET("/backoffice")
			Response(StatusOK)
		})
	})

	Method("legacyBackofficeRole", func() {
		Payload(func() {
			Field(1, "role", String)
			Required("role")
		})
		Result(Empty)
		HTTP(func() {
			GET("/backoffice/{role}")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyBackofficeRolePartial", func() {
		Payload(func() {
			Field(1, "role", String)
			Required("role")
		})
		Result(Empty)
		HTTP(func() {
			GET("/backoffice/{role}/partial")
			Response(StatusOK)
			Response(StatusNotFound)
		})
	})

	Method("legacyBackofficeRoleProcess", func() {
		Payload(func() {
			Field(1, "role", String)
			Field(2, "process_id", String)
			Required("role", "process_id")
		})
		Result(Empty)
		HTTP(func() {
			GET("/backoffice/{role}/process/{process_id}")
			Response(StatusSeeOther)
			Response(StatusNotFound)
		})
	})

	Method("legacyImpersonate", func() {
		Result(Empty)
		HTTP(func() {
			POST("/impersonate")
			Response(StatusBadRequest)
		})
	})

	Method("legacyEvents", func() {
		Result(Empty)
		HTTP(func() {
			GET("/events")
			Response(StatusOK)
			Response(StatusBadRequest)
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
})
