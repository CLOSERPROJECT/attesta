package main

import "strings"

func orgPendingRowFromJoinRequest(row OrgAdminJoinRequestRow) OrgPendingRowView {
	return OrgPendingRowView{
		Email:     strings.TrimSpace(row.RequesterEmail),
		Roles:     rolePillRowFromOrgAdminOptions(row.Roles, "sm"),
		DateLabel: "Requested on",
		Date:      strings.TrimSpace(row.CreatedAt),
	}
}

func orgPendingRowFromInvite(row OrgAdminInviteRow) OrgPendingRowView {
	return OrgPendingRowView{
		Email:     strings.TrimSpace(row.Email),
		Roles:     rolePillRowFromOrgAdminOptions(row.Roles, "sm"),
		DateLabel: "Invited on",
		Date:      strings.TrimSpace(row.CreatedAtLabel),
	}
}
