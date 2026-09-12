package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/toddbirchard/penguintruth/home"

	log "github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Production logs are written here for collection by the Datadog agent.
// The file and its parent directory are provisioned by the deploy.
const logFilePath = "/var/log/penguintruth/info.json"

// datadogFieldMap renames logrus' default keys to Datadog's standard attributes.
var datadogFieldMap = log.FieldMap{
	log.FieldKeyTime:  "date",
	log.FieldKeyLevel: "status",
	log.FieldKeyMsg:   "message",
}

// Writes Datadog-friendly JSON to a file on production,
// and human-readable text to stdout in every other environment.
// `logPath` is a parameter so tests can target a temporary file.
func initLogger(logPath string) {
	log.SetLevel(log.InfoLevel)

	if os.Getenv("ENVIRONMENT") != "production" {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
		log.SetOutput(os.Stdout)
		return
	}

	log.SetFormatter(&log.JSONFormatter{
		FieldMap:        datadogFieldMap,
		TimestampFormat: time.RFC3339Nano,
	})

	// lumberjack opens the file lazily, so preflight it here to surface a bad
	// path or permissions at boot rather than on the first write.
	logFile, err := os.OpenFile(logPath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		// Retain logrus' default stderr output rather than losing logs entirely.
		log.WithError(err).Errorf("Failed to open %s; falling back to stderr.", logPath)
		return
	}
	_ = logFile.Close()

	log.SetOutput(&lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100,  // Rotate once the file reaches 100mb.
		MaxBackups: 7,    // Retain 7 rotated files,
		MaxAge:     30,   // ...none older than 30 days.
		Compress:   true, // Gzip rotated files.
	})
}

// Construct Host IP Address
func constructAddress() string {
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
	// Load environment variables from `.env` file.
	envErr := godotenv.Load()

	// Initialize logger.
	initLogger(logFilePath)

	// Warn if .env file is missing, but continue using process environment variables.
	if envErr != nil {
		log.Warn("No .env file found; falling back to process environment.")
	}

	home.CompileStylesheets()
	webserverAddress := constructAddress()
	router := Router()
	client := &http.Server{
		Handler:      router,
		Addr:         webserverAddress,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	log.Infof("PenguinTruth now live and listening at %s...", webserverAddress)
	log.Fatal(client.ListenAndServe())
}
