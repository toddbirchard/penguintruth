package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/toddbirchard/penguintruth/home"
)

// Construct Host IP Address
func constructAddress() string {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load environment variables from .env file.")
	}

	addressPort := os.Getenv("WEBSERVER_PORT")
	if addressPort == "" {
		log.Fatal("Webserver port not set via `WEBSERVER_PORT` env var.")
	}
	return fmt.Sprintf("127.0.0.1:%s", addressPort)
}

// Router declaration
func Router() *mux.Router {
	staticDir := "/static/"
	// Page routes
	r := mux.NewRouter()
	r.HandleFunc("/", home.IndexHandler)
	r.PathPrefix(staticDir).Handler(http.StripPrefix(staticDir, http.FileServer(http.Dir("."+staticDir))))
	return r
}

// Initiate web server
func main() {
	home.CompileStylesheets()
	webserverAddress := constructAddress()
	router := Router()
	client := &http.Server{
		Handler:      router,
		Addr:         webserverAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Printf("PenguinTruth now live and listening at %s...", webserverAddress)
	log.Fatal(client.ListenAndServe())
}
