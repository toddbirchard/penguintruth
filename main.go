package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/kib357/less-go"
	"html/template"
	"log"
	"net/http"
	"time"
)

// CompileStylesheets Compile and minify .LESS files
func CompileStylesheets() {
	staticFolder := "./static/styles/%s"
	err := less.RenderFile(fmt.Sprintf(staticFolder, "style.less"), fmt.Sprintf(staticFolder, "style.css"), map[string]interface{}{"compress": true})
	if err != nil {
		log.Fatal(err)
	}
}

// HomeMetaData Dynamic template values
type HomeMetaData struct {
	Title      string
	TagLine    string
	SiteUrl    string
	ShareImage string
	MainImage  string
	Icon       string
}

// IndexHandler Render homepage template
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/index.html"))
	data := HomeMetaData{
		Title:      "Penguin Truth",
		TagLine:    "Exposing the facts about penguins and their flightless origins.",
		SiteUrl:    "https://penguintruth.org/",
		ShareImage: "/static/img/penguin-share@2x.jpg",
		MainImage:  "/static/img/antipenguin@2x.png",
		Icon:       "/static/img/favicon.png",
	}
	_ = tmpl.Execute(w, data)
}

// Router Route declaration
func Router() *mux.Router {
	staticDir := "/static/"
	// Page routes
	r := mux.NewRouter()
	r.HandleFunc("/", IndexHandler)
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("."+staticDir))))
	return r
}

// Initiate web server
func main() {
	CompileStylesheets()
	router := Router()
	client := &http.Server{
		Handler:      router,
		Addr:         "127.0.0.1:9200",
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Fatal(client.ListenAndServe())
}
