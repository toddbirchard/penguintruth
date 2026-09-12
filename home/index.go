package home

import (
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
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

// The webserver binds to loopback and is fronted by a reverse proxy, so
// RemoteAddr is the proxy rather than the visitor. Prefer the headers the proxy
// sets, falling back to the connection itself for direct requests.
func clientIP(r *http.Request) string {
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// The client is the left-most entry; the rest are intermediate proxies.
		client, _, _ := strings.Cut(forwarded, ",")
		if client = strings.TrimSpace(client); client != "" {
			return client
		}
	}
	if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
		return realIP
	}
	// RemoteAddr carries a port, except in the synthetic requests tests build.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

// IndexHandler Render homepage template
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	tmpl := template.Must(template.ParseFiles("static/dist/html/index.html"))
	data := MetaData{
		Title:      "Penguin Truth",
		TagLine:    "Exposing the facts about penguins and their flightless origins.",
		SiteUrl:    "https://penguintruth.org/",
		ShareImage: "/static/dist/img/penguin-share@2x.jpg",
		MainImage:  "/static/dist/img/antipenguin@2x.png",
		Icon:       "/static/dist/img/favicon.png",
	}

	// Field names follow Datadog's standard HTTP attributes so these lines land
	// on its built-in facets instead of arriving as unmapped custom attributes.
	entry := log.WithFields(log.Fields{
		"http.method":           r.Method,
		"http.url_details.path": r.URL.Path,
		"http.useragent":        r.UserAgent(),
		"http.referer":          r.Referer(),
		"network.client.ip":     clientIP(r),
	})

	if err := tmpl.Execute(w, data); err != nil {
		// A 200 and part of the page are already on the wire by this point, so
		// the status cannot be corrected; record why the response is truncated.
		entry.WithError(err).Error("Failed to render homepage.")
		return
	}

	entry.WithFields(log.Fields{
		"http.status_code": http.StatusOK,
		// Datadog reads its `duration` standard attribute as nanoseconds.
		"duration": time.Since(start).Nanoseconds(),
	}).Info("Homepage rendered.")
}
