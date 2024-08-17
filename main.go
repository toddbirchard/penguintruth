package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/bep/godartsass/v2"
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

type LogEvent struct {
	// Type is the type of log event.
	Type LogEventType

	// Message on the form url:line:col message.
	Message string
}



type Options struct {
	// The path to the Dart Sass wrapper binary, an absolute filename
	// if not in $PATH.
	// If this is not set, we will try 'dart-sass'
	// (or 'dart-sass.bat' on Windows) in the OS $PATH.
	// There may be several ways to install this, one would be to
	// download it from here: https://github.com/sass/dart-sass/releases
	DartSassEmbeddedFilename string

	// Timeout is the duration allowed for dart sass to transpile.
	// This was added for the beta6 version of Dart Sass Protocol,
	// as running this code against the beta5 binary would hang
	// on Execute.
	Timeout time.Duration

	// LogEventHandler will, if set, receive log events from Dart Sass,
	// e.g. @debug and @warn log statements.
	LogEventHandler func(LogEvent)
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
	home.Start()
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
