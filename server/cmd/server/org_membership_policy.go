package main

// Org membership policy: pure allow/deny decisions with one UI-facing reason each.
// Shared by org-admin handlers, view builders, and Affiliation.LeaveOrganization.

type PolicyDecision struct {
	Allowed bool
	Reason  string // empty when Allowed
}

const (
	reasonSelfDelete         = "You can't delete your own account from here. Use Leave."
	reasonSoleOrgAdminDemote = "This is the only Org admin. Add another before removing this one."
	reasonSoleOrgAdminLeave  = "You're the only Org admin. Add another before leaving."
	reasonLastCatalogRole    = "Organizations must keep at least one role."
	reasonRoleInUse          = "Role in use"
)

// CountOrgAdmins returns how many non-platform-admin users hold Org admin standing.
func CountOrgAdmins(users []IdentityUser) int {
	n := 0
	for _, user := range users {
		if isPlatformAdminIdentityUser(user) {
			continue
		}
		if user.IsOrgAdmin {
			n++
		}
	}
	return n
}

func CanDeleteMember(actorKey, targetKey string) PolicyDecision {
	if actorKey == targetKey {
		return PolicyDecision{Reason: reasonSelfDelete}
	}
	return PolicyDecision{Allowed: true}
}

func CanChangeOrgAdmin(targetIsAdmin, wantAdmin bool, otherAdminCount int) PolicyDecision {
	if targetIsAdmin && !wantAdmin && otherAdminCount == 0 {
		return PolicyDecision{Reason: reasonSoleOrgAdminDemote}
	}
	return PolicyDecision{Allowed: true}
}

func CanLeaveOrganization(isOrgAdmin bool, adminCount int) PolicyDecision {
	if !isOrgAdmin {
		return PolicyDecision{Allowed: true}
	}
	if adminCount < 2 {
		return PolicyDecision{Reason: reasonSoleOrgAdminLeave}
	}
	return PolicyDecision{Allowed: true}
}

func CanDeleteCatalogRole(catalogLen int, inUse bool) PolicyDecision {
	if inUse {
		return PolicyDecision{Reason: reasonRoleInUse}
	}
	if catalogLen <= 1 {
		return PolicyDecision{Reason: reasonLastCatalogRole}
	}
	return PolicyDecision{Allowed: true}
}

func CanEditCatalogRole(inUse bool) PolicyDecision {
	if inUse {
		return PolicyDecision{Reason: reasonRoleInUse}
	}
	return PolicyDecision{Allowed: true}
}
