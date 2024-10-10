package home

import (
	"html/template"
	"net/http"
)

// MetaData Dynamic template values
type MetaData struct {
	Title      string
	TagLine    string
	SiteUrl    string
	ShareImage string
	MainImage  string
	Icon       string
}

// IndexHandler Render homepage template
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("static/templates/index.html"))
	data := MetaData{
		Title:      "Penguin Truth",
		TagLine:    "Exposing the facts about penguins and their flightless origins.",
		SiteUrl:    "https://penguintruth.org/",
		ShareImage: "/dist/img/penguin-share@2x.jpg",
		MainImage:  "/dist/img/antipenguin@2x.png",
		Icon:       "/dist/img/favicon.png",
	}
	_ = tmpl.Execute(w, data)
}