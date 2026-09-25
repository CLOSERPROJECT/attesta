package main

import "strings"

func rolePillRowFromOrgAdminOptions(options []OrgAdminRoleOption, size string) RolePillRowView {
	pills := make([]RolePillView, 0, len(options))
	for _, opt := range options {
		label := strings.TrimSpace(opt.Name)
		if label == "" {
			label = strings.TrimSpace(opt.Slug)
		}
		if label == "" {
			continue
		}
		pills = append(pills, RolePillView{
			Label:   label,
			Palette: strings.TrimSpace(opt.Palette),
		})
	}
	return RolePillRowView{Pills: pills, Size: strings.TrimSpace(size)}
}

func rolePillRowSelectedFromOrgAdminOptions(options []OrgAdminRoleOption, size string) RolePillRowView {
	selected := make([]OrgAdminRoleOption, 0, len(options))
	for _, opt := range options {
		if opt.Selected {
			selected = append(selected, opt)
		}
	}
	return rolePillRowFromOrgAdminOptions(selected, size)
}

func rolePillRowFromSubstepBody(body SubstepBodyView) RolePillRowView {
	pills := make([]RolePillView, 0, len(body.RoleBadges))
	for _, badge := range body.RoleBadges {
		label := strings.TrimSpace(badge.Label)
		if label == "" {
			label = strings.TrimSpace(badge.ID)
		}
		if label == "" {
			continue
		}
		pills = append(pills, RolePillView{
			Label:   label,
			Palette: strings.TrimSpace(badge.Palette),
		})
	}
	label := "Required roles:"
	if body.Status == "done" {
		label = "Completed by role:"
	} else if len(pills) == 1 {
		label = "Required role:"
	}
	return RolePillRowView{
		Pills: pills,
		Label: label,
		Size:  "sm",
		Class: "u-m-0",
	}
}
