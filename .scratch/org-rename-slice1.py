#!/usr/bin/env python3
"""One-shot renames for org slice 1 (Go engine + host)."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]

GO_DIRS = [
    ROOT / "server/internal/org",
]
GO_FILES = list(GO_DIRS[0].rglob("*.go"))
GO_FILES += [
    ROOT / "server/cmd/server/org.go",
    ROOT / "server/cmd/server/org_dpp.go",
    ROOT / "server/cmd/server/org_test.go",
    ROOT / "server/cmd/server/org_dpp_test.go",
    ROOT / "server/cmd/server/org_mongo_integration_test.go",
    ROOT / "server/cmd/server/seed.go",
    ROOT / "server/cmd/server/seed_fixture.go",
    ROOT / "server/cmd/server/seed_cmd.go",
    ROOT / "server/cmd/server/seed_session.go",
    ROOT / "server/cmd/server/seed_test.go",
    ROOT / "server/cmd/server/main.go",
    ROOT / "server/cmd/server/prepare_demo_cmd.go",
    ROOT / "server/cmd/server/prepare_demo_test.go",
    ROOT / "server/cmd/server/test_helpers_test.go",
    ROOT / "server/cmd/seed-loaded-extracted/main.go",
]

TEMPLATE_FILES = [
    ROOT / "server/templates/pages/org_dpp.html",
    ROOT / "server/templates/layout.html",
]

YAML_FIXTURE = ROOT / "server/internal/org/fixtures/feragame-config.yaml"

# Longest-first token renames (word boundaries).
TOKEN_RENAMES = [
    ("NewMongoPassportStore", "NewMongoDPPStore"),
    ("NewMemoryPassportStore", "NewMemoryDPPStore"),
    ("MongoPassportStore", "MongoDPPStore"),
    ("MemoryPassportStore", "MemoryDPPStore"),
    ("createErrPassportStore", "createErrDPPStore"),
    ("PassportSnapshotChromeID", "DPPSnapshotChromeID"),
    ("ValidatePassportConfig", "ValidateDPPConfig"),
    ("EvalPassportSnapshot", "EvalDPPSnapshot"),
    ("EvalPassportIdentity", "EvalDPPIdentity"),
    ("PassportSnapshotConfig", "DPPSnapshotConfig"),
    ("PassportIdentityConfig", "DPPIdentityConfig"),
    ("PassportSubjectConfig", "DPPSubjectConfig"),
    ("PassportConfig", "DPPConfig"),
    ("PassportRecord", "DPPRecord"),
    ("PassportStore", "DPPStore"),
    ("ErrPassportExists", "ErrDPPExists"),
    ("issuePassport", "issueDPP"),
    ("evalPassportIdentity", "evalDPPIdentity"),
    ("evalPassportString", "evalDPPString"),
    ("validatePassportBlock", "validateDPPBlock"),
    ("orgConfigHasPassportOptIn", "orgConfigHasDPPOptIn"),
    ("taskOnSubmitPassport", "taskOnSubmitDPP"),
    ("syntheticPassportConfig", "syntheticDPPConfig"),
    ("passportFromMongoDoc", "dppFromMongoDoc"),
    ("mongoPassportDoc", "mongoDPPDoc"),
    ("LivingAppBaseURL", "OrgAppBaseURL"),
    ("livingResourceKind", "taskResourceKind"),
    ("collectionLivingSubmissionLinks", "collectionOrgSubmissionLinks"),
    ("collectionLivingActiveConfig", "collectionOrgActiveConfig"),
    ("collectionLivingOrgConfigs", "collectionOrgOrgConfigs"),
    ("collectionLivingDomains", "collectionOrgDomains"),
    ("collectionLivingPassports", "collectionOrgDPPs"),
    ("LivingPassports", "OrgDPPs"),
    ("livingPassports", "orgDPPs"),
    ("livingProductArtifacts", "orgProductArtifacts"),
    ("livingProductHttpPort", "orgProductHttpPort"),
    ("livingProductAuthorizer", "orgProductAuthorizer"),
    ("livingDomainHandler", "orgDomainHandler"),
    ("livingVerifiedGate", "orgVerifiedGate"),
    ("newLivingPassportStore", "newOrgDPPStore"),
    ("newLivingLinkStore", "newOrgLinkStore"),
    ("newLivingConfigStore", "newOrgConfigStore"),
    ("newLivingDomainStore", "newOrgDomainStore"),
    ("livingMongoDB", "orgMongoDB"),
    ("ensureLivingStores", "ensureOrgStores"),
    ("livingHandler", "orgAPI"),
    ("livingAuthorizer", "orgAuthorizer"),
    ("livingHttpPort", "orgHttpPort"),
    ("livingLinks", "orgLinks"),
    ("livingConfigs", "orgConfigs"),
    ("livingDomains", "orgDomains"),
    ("demoLivingLogisticUserEmail", "demoLogisticUserEmail"),
    ("demoLivingAdminUserEmail", "demoAdministrationUserEmail"),
    ("demoLivingLineUserEmail", "demoLineUserEmail"),
    ("demoLivingUserEmail", "demoOrgAdminUserEmail"),
    ("demoLivingOrgSlug", "demoOrgSlug"),
    ("testCommitter", "testTaskCommitter"),
    ("TaskCommitter", "TASK_COMMITTER_PLACEHOLDER"),
    ("Committer", "TaskCommitter"),
    ("TASK_COMMITTER_PLACEHOLDER", "TaskCommitter"),
    ("DomainDomainLoader", "DomainLoader"),  # guard double rename
    ("Loader", "DomainLoader"),
    ("Handler", "API"),
]


def apply_tokens(text: str) -> str:
    text = text.replace("http.Handler", "§HTTP_HANDLER§")
    text = text.replace("logRequests(next http.Handler)", "logRequests(next http.Handler)")
    for old, new in TOKEN_RENAMES:
        if old == "Handler":
            continue  # done below with protection
        text = re.sub(r"\b" + re.escape(old) + r"\b", new, text)
    text = re.sub(r"\bHandler\b", "API", text)
    text = text.replace("§HTTP_HANDLER§", "http.Handler")
    # Fix accidental DomainDomainLoader if Loader was inside DomainLoader already
    text = text.replace("DomainDomainLoader", "DomainLoader")
    return text


def apply_common(text: str) -> str:
    text = text.replace('package living', 'package org')
    text = text.replace(
        "github.com/CLOSERPROJECT/attesta/server/internal/living",
        "github.com/CLOSERPROJECT/attesta/server/internal/org",
    )
    text = text.replace("/api/living", "/api/org")
    text = re.sub(r"\bliving:", "org:", text)
    text = text.replace("administration@attesta.com", "administration@feragame.com")
    text = text.replace("logistic@attesta.com", "logistic@feragame.com")
    text = text.replace("line@attesta.com", "line@feragame.com")
    text = text.replace("living@attesta.com", "admin@feragame.com")
    text = text.replace("LIVING_HTTP_", "ORG_HTTP_")
    text = text.replace("http://living-http:", "http://org-http:")
    text = text.replace("living_passports", "org_dpps")
    text = text.replace("living_domains", "org_domains")
    text = text.replace("living_org_configs", "org_configs")
    text = text.replace("living_active_config", "org_active_config")
    text = text.replace("living_submission_links", "org_submission_links")
    text = text.replace("orgAPI := s.orgAPI()", "orgHTTP := s.orgAPI()")
    text = text.replace("mux.Handle(\"/api/org/domain\", orgAPI)", "mux.Handle(\"/api/org/domain\", orgHTTP)")
    text = text.replace("mux.Handle(\"/api/org/task-form\", orgAPI)", "mux.Handle(\"/api/org/task-form\", orgHTTP)")
    text = text.replace("mux.Handle(\"/api/org/task-recap\", orgAPI)", "mux.Handle(\"/api/org/task-recap\", orgHTTP)")
    text = text.replace("mux.Handle(\"/api/org/commit\", orgAPI)", "mux.Handle(\"/api/org/commit\", orgHTTP)")
    text = text.replace("mux.Handle(\"/api/org/close\", orgAPI)", "mux.Handle(\"/api/org/close\", orgHTTP)")
    text = text.replace("mux.Handle(\"/api/org/qr-resolve\", orgAPI)", "mux.Handle(\"/api/org/qr-resolve\", orgHTTP)")
    text = text.replace("mux.Handle(\"/api/org/artifacts/\", orgAPI)", "mux.Handle(\"/api/org/artifacts/\", orgHTTP)")
    text = text.replace('taskResourceKind = "living"', 'taskResourceKind = "task"')
    text = text.replace('collectionOrgOrgConfigs', 'collectionOrgConfigs')
    text = text.replace('const collectionOrgOrgConfigs', 'const collectionOrgConfigs')
    # OrgConfig field
    text = text.replace("Passports ", "DPPs ")
    text = text.replace("Passports:", "DPPs:")
    text = text.replace(".Passports", ".DPPs")
    text = text.replace("opts.Passports", "opts.DPPs")
    text = re.sub(r"\bPassportStore\b", "DPPStore", text)
    text = re.sub(r"\bcfg\.Passport\b", "cfg.DPP", text)
    text = re.sub(r"\bwire\.Passport\b", "wire.DPP", text)
    text = re.sub(r"\bp\.Passport\b", "p.DPP", text)
    text = re.sub(r"\bnoRoot\.Passport\b", "noRoot.DPP", text)
    text = re.sub(r"\bbad\.Passport\b", "bad.DPP", text)
    text = re.sub(r"\bbadIdent\.Passport\b", "badIdent.DPP", text)
    text = re.sub(r"\bbadSnap\.Passport\b", "badSnap.DPP", text)
    text = re.sub(r"\bboomSnap\.Passport\b", "boomSnap.DPP", text)
    text = text.replace("*PassportConfig", "*DPPConfig")
    text = text.replace("Passport *PassportConfig", "DPP *DPPConfig")
    text = text.replace(
        'Passport *PassportConfig `json:"passport,omitempty"`',
        'DPP *DPPConfig `json:"dpp,omitempty"`',
    )
    text = text.replace('json:"passport,omitempty"', 'json:"dpp,omitempty"')
    text = text.replace("living_passport", "org_dpp")
    text = text.replace("living load not configured", "org load not configured")
    text = text.replace("living mongo", "org mongo")
    text = text.replace("living: onSubmit.passport", "org: onSubmit.passport")
    text = text.replace("living: passport", "org: dpp")
    text = text.replace('org: dpp already exists', 'org: dpp already exists')
    return text


def process_file(path: Path) -> None:
    if not path.exists():
        return
    original = path.read_text()
    text = apply_tokens(original)
    text = apply_common(text)
    if text != original:
        path.write_text(text)


def process_yaml() -> None:
    if not YAML_FIXTURE.exists():
        return
    text = YAML_FIXTURE.read_text()
    text = re.sub(r"^passport:", "dpp:", text, flags=re.M)
    text = text.replace("Living Passport", "Org DPP")
    YAML_FIXTURE.write_text(text)


def process_templates() -> None:
    for path in TEMPLATE_FILES:
        if not path.exists():
            continue
        text = path.read_text()
        new = text.replace("living_passport", "org_dpp")
        if new != text:
            path.write_text(new)


def main() -> None:
    for path in GO_FILES:
        process_file(path)
    process_yaml()
    process_templates()


if __name__ == "__main__":
    main()