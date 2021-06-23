package main

type Bookmark struct {
	Id            int
	Url           string
	Title         string
	Description   string
	Website_title string
	Tag_names     []string
}

type Bookmarks struct {
	Count    int
	Next     string
	Previous string
	Results  []Bookmark
}
