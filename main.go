package main

import (
	"github.com/gorilla/mux"
	"html/template"
	"log"
	"net/http"
	"time"
)

// Dynamic template values
type HomeMetaData struct {
	Title      string
	TagLine    string
	SiteUrl    string
	ShareImage string
	Icon       string
}

// Render homepage
func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	data := HomeMetaData{
		Title:      "Penguin Truth",
		TagLine:    "Exposing the facts about penguins and their flightless origins.",
		SiteUrl:    "https://penguintruth.org/",
		ShareImage: "/static/img/penguin-share@2x.jpg",
		Icon:       "/static/img/favicon.png",
	}
	tmpl.Execute(w, data)
}

// Route declaration
func Router() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/", indexHandler)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return r
}

// Initiate web server
func main() {
	router := Router()

	srv := &http.Server{
		Handler:      router,
		Addr:         "127.0.0.1:9200",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	log.Fatal(srv.ListenAndServe())
}
