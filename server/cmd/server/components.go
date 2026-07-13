package main

// PageHeaderView is the view model for templates/components/page_header.html.
type PageHeaderView struct {
	BackHref    string
	BackLabel   string
	Title       string
	Subtitle    string
	Description string
	Meta        string
}
