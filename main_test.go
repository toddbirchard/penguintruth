package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/toddbirchard/penguintruth/home"
	"gopkg.in/natefinch/lumberjack.v2"
)

// restoreLogger snapshots the global logrus configuration and puts it back once
// the test finishes, since initLogger mutates process-wide state.
func restoreLogger(t *testing.T) {
	t.Helper()
	logger := log.StandardLogger()
	formatter, out, level := logger.Formatter, logger.Out, logger.Level
	t.Cleanup(func() {
		log.SetFormatter(formatter)
		log.SetOutput(out)
		log.SetLevel(level)
	})
}

func TestConstructAddress(t *testing.T) {
	t.Setenv("WEBSERVER_PORT", "8000")

	if got, want := constructAddress(), "127.0.0.1:8000"; got != want {
		t.Errorf("constructAddress() = %q, want %q", got, want)
	}
}

// constructAddress calls log.Fatal when the port is unset, so the exit has to be
// observed from a subprocess rather than in-process.
func TestConstructAddressExitsWhenPortUnset(t *testing.T) {
	if os.Getenv("TEST_SUBPROCESS_EXIT") == "1" {
		os.Unsetenv("WEBSERVER_PORT")
		constructAddress()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=^TestConstructAddressExitsWhenPortUnset$")
	cmd.Env = append(os.Environ(), "TEST_SUBPROCESS_EXIT=1")

	var exitErr *exec.ExitError
	if err := cmd.Run(); !errors.As(err, &exitErr) {
		t.Fatalf("expected the process to exit when WEBSERVER_PORT is unset, got err=%v", err)
	}
	if got := exitErr.ExitCode(); got != 1 {
		t.Errorf("exit code = %d, want 1", got)
	}
}

func TestRouterServesHomepage(t *testing.T) {
	recorder := httptest.NewRecorder()
	Router().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", recorder.Code, http.StatusOK)
	}

	// Every field of MetaData should reach the rendered template.
	body := recorder.Body.String()
	for _, want := range []string{
		"Penguin Truth",
		"Exposing the facts about penguins and their flightless origins.",
		"https://penguintruth.org/",
		"/static/dist/img/penguin-share@2x.jpg",
		"/static/dist/img/antipenguin@2x.png",
		"/static/dist/img/favicon.png",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("homepage is missing %q", want)
		}
	}

	// An unresolved action means a field was renamed out from under the template.
	if strings.Contains(body, "<no value>") {
		t.Error("homepage contains an unrendered template field")
	}
}

// captureProductionLogs points logrus at a buffer using the same formatter
// production runs with, so assertions see the field names Datadog will ingest.
func captureProductionLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	restoreLogger(t)

	var logs bytes.Buffer
	log.SetOutput(&logs)
	log.SetFormatter(&log.JSONFormatter{
		FieldMap:        datadogFieldMap,
		TimestampFormat: time.RFC3339Nano,
	})
	log.SetLevel(log.InfoLevel)
	return &logs
}

func TestHomepageRequestEmitsInfoLog(t *testing.T) {
	logs := captureProductionLogs(t)

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Forwarded-For", "203.0.113.7, 198.51.100.4")
	request.Header.Set("User-Agent", "PenguinTruthTest/1.0")
	request.Header.Set("Referer", "https://example.com/roost")

	recorder := httptest.NewRecorder()
	Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET / = %d, want %d", recorder.Code, http.StatusOK)
	}

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v (got %q)", err, logs.String())
	}

	for key, want := range map[string]any{
		"message":               "Homepage rendered.",
		"status":                "info",
		"http.method":           http.MethodGet,
		"http.url_details.path": "/",
		"http.status_code":      float64(http.StatusOK),
		"http.useragent":        "PenguinTruthTest/1.0",
		"http.referer":          "https://example.com/roost",
		// Only the left-most forwarded entry is the visitor.
		"network.client.ip": "203.0.113.7",
	} {
		if got := entry[key]; got != want {
			t.Errorf("entry[%q] = %v, want %v", key, got, want)
		}
	}

	if _, ok := entry["duration"].(float64); !ok {
		t.Errorf(`entry["duration"] = %v, want a number of nanoseconds`, entry["duration"])
	}
}

// The webserver only listens on loopback, so an unproxied RemoteAddr would
// record every visitor as 127.0.0.1.
func TestHomepageLogPrefersForwardedClientIP(t *testing.T) {
	for name, tc := range map[string]struct {
		headers map[string]string
		want    string
	}{
		"forwarded chain": {map[string]string{"X-Forwarded-For": "203.0.113.7, 198.51.100.4"}, "203.0.113.7"},
		"padded entry":    {map[string]string{"X-Forwarded-For": "  203.0.113.7  "}, "203.0.113.7"},
		"real ip":         {map[string]string{"X-Real-IP": "203.0.113.9"}, "203.0.113.9"},
		"forwarded wins":  {map[string]string{"X-Forwarded-For": "203.0.113.7", "X-Real-IP": "203.0.113.9"}, "203.0.113.7"},
		// httptest gives synthetic requests a RemoteAddr of 192.0.2.1:1234.
		"unproxied":    {nil, "192.0.2.1"},
		"empty header": {map[string]string{"X-Forwarded-For": "   "}, "192.0.2.1"},
	} {
		t.Run(name, func(t *testing.T) {
			logs := captureProductionLogs(t)

			request := httptest.NewRequest(http.MethodGet, "/", nil)
			for header, value := range tc.headers {
				request.Header.Set(header, value)
			}
			Router().ServeHTTP(httptest.NewRecorder(), request)

			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("log line is not valid JSON: %v (got %q)", err, logs.String())
			}
			if got := entry["network.client.ip"]; got != tc.want {
				t.Errorf("network.client.ip = %v, want %v", got, tc.want)
			}
		})
	}
}

// Static assets are served straight off disk; logging each one would bury the
// pageview lines the homepage log exists to produce.
func TestStaticAssetRequestsAreNotLogged(t *testing.T) {
	logs := captureProductionLogs(t)

	request := httptest.NewRequest(http.MethodGet, "/static/dist/css/style.css", nil)
	recorder := httptest.NewRecorder()
	Router().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("GET /static/dist/css/style.css = %d, want %d", recorder.Code, http.StatusOK)
	}
	if logs.Len() != 0 {
		t.Errorf("static asset request logged %q", logs.String())
	}
}

func TestRouterServesStaticAssets(t *testing.T) {
	for _, path := range []string{
		"/static/dist/css/style.css",
		"/static/dist/css/fonts.css",
		"/static/dist/img/favicon.png",
		"/static/dist/fonts/Inter-Regular.woff2",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			Router().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			if recorder.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d", path, recorder.Code, http.StatusOK)
			}
			if recorder.Body.Len() == 0 {
				t.Errorf("GET %s returned an empty body", path)
			}
		})
	}
}

func TestRouterReturnsNotFoundForUnknownPaths(t *testing.T) {
	for _, path := range []string{"/penguins", "/static/dist/css/nonexistent.css"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			Router().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			if recorder.Code != http.StatusNotFound {
				t.Errorf("GET %s = %d, want %d", path, recorder.Code, http.StatusNotFound)
			}
		})
	}
}

// The file server must stay rooted at ./static. The traversal cases below are
// redirected by mux's path cleaning before they reach the handler, so it is the
// non-traversing paths that actually detect a misrooted http.Dir.
func TestRouterServesOnlyFilesUnderStaticDir(t *testing.T) {
	for _, path := range []string{
		"/static/main.go",
		"/static/go.mod",
		"/static/.env",
		"/static/../main.go",
		"/static/dist/../../main.go",
		"/static/..%2fmain.go",
	} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			Router().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))

			if recorder.Code == http.StatusOK {
				t.Errorf("GET %s = 200; the static handler is serving outside ./static/", path)
			}
			if strings.Contains(recorder.Body.String(), "package main") {
				t.Errorf("GET %s leaked Go source", path)
			}
		})
	}
}

func TestCompileStylesheetsProducesCSS(t *testing.T) {
	// CompileStylesheets writes into the working tree, so restore the committed
	// stylesheet afterwards rather than leaving the repo dirty.
	const compiled = "./static/dist/css/style.css"
	original, err := os.ReadFile(compiled)
	if err != nil {
		t.Fatalf("reading %s: %v", compiled, err)
	}
	t.Cleanup(func() {
		if err := os.WriteFile(compiled, original, 0o644); err != nil {
			t.Errorf("restoring %s: %v", compiled, err)
		}
	})

	if err := os.WriteFile(compiled, []byte("/* overwritten by test */"), 0o644); err != nil {
		t.Fatalf("clearing %s: %v", compiled, err)
	}

	home.CompileStylesheets()

	got, err := os.ReadFile(compiled)
	if err != nil {
		t.Fatalf("reading %s after compiling: %v", compiled, err)
	}
	if len(got) == 0 {
		t.Fatal("compiled stylesheet is empty")
	}
	if bytes.Contains(got, []byte("overwritten by test")) {
		t.Fatal("CompileStylesheets did not rewrite the stylesheet")
	}
	// The .less sources declare variables that must not survive into the output.
	if bytes.Contains(got, []byte("@import")) {
		t.Error("compiled stylesheet still contains an unresolved @import")
	}
}

func TestInitLoggerWritesReadableTextToStdoutOutsideProduction(t *testing.T) {
	restoreLogger(t)
	t.Setenv("ENVIRONMENT", "development")
	logPath := filepath.Join(t.TempDir(), "info.json")

	initLogger(logPath)

	if _, ok := log.StandardLogger().Formatter.(*log.TextFormatter); !ok {
		t.Errorf("formatter = %T, want *logrus.TextFormatter outside production", log.StandardLogger().Formatter)
	}
	if log.StandardLogger().Out != os.Stdout {
		t.Error("expected logs to be written to stdout outside production")
	}
	if log.GetLevel() != log.InfoLevel {
		t.Errorf("level = %s, want info", log.GetLevel())
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("expected no log file outside production, but %s exists", logPath)
	}
}

func TestInitLoggerWritesDatadogJSONToFileOnProduction(t *testing.T) {
	restoreLogger(t)
	t.Setenv("ENVIRONMENT", "production")
	logPath := filepath.Join(t.TempDir(), "info.json")

	initLogger(logPath)
	log.WithField("port", 8000).Info("PenguinTruth now live")

	contents, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading %s: %v", logPath, err)
	}

	var entry map[string]any
	if err := json.Unmarshal(contents, &entry); err != nil {
		t.Fatalf("log line is not valid JSON: %v (got %q)", err, contents)
	}

	// Datadog's standard attributes, plus any custom fields.
	for key, want := range map[string]any{
		"message": "PenguinTruth now live",
		"status":  "info",
		"port":    float64(8000),
	} {
		if got := entry[key]; got != want {
			t.Errorf("entry[%q] = %v, want %v", key, got, want)
		}
	}

	// Logrus' own key names would be ingested as unmapped attributes.
	for _, key := range []string{"msg", "level", "time"} {
		if _, ok := entry[key]; ok {
			t.Errorf("entry uses logrus' default key %q; Datadog expects it remapped", key)
		}
	}

	timestamp, ok := entry["date"].(string)
	if !ok {
		t.Fatalf(`entry["date"] = %v, want an RFC3339Nano string`, entry["date"])
	}
	if _, err := time.Parse(time.RFC3339Nano, timestamp); err != nil {
		t.Errorf("date %q is not RFC3339Nano, so Datadog cannot parse it: %v", timestamp, err)
	}
}

func TestInitLoggerRotatesProductionLogs(t *testing.T) {
	restoreLogger(t)
	t.Setenv("ENVIRONMENT", "production")
	logPath := filepath.Join(t.TempDir(), "info.json")

	initLogger(logPath)

	// Asserted rather than exercised: triggering a real rotation would mean
	// writing MaxSize (100mb) to disk on every test run.
	rotator, ok := log.StandardLogger().Out.(*lumberjack.Logger)
	if !ok {
		t.Fatalf("production output = %T, want *lumberjack.Logger so logs are rotated", log.StandardLogger().Out)
	}
	if rotator.Filename != logPath {
		t.Errorf("rotator.Filename = %q, want %q", rotator.Filename, logPath)
	}
	if rotator.MaxSize != 100 {
		t.Errorf("rotator.MaxSize = %d, want 100", rotator.MaxSize)
	}
	if rotator.MaxBackups != 7 {
		t.Errorf("rotator.MaxBackups = %d, want 7", rotator.MaxBackups)
	}
	if rotator.MaxAge != 30 {
		t.Errorf("rotator.MaxAge = %d, want 30", rotator.MaxAge)
	}
	if !rotator.Compress {
		t.Error("rotator.Compress = false, want true")
	}
}

func TestInitLoggerPreflightLeavesLogFileReadableByAgent(t *testing.T) {
	restoreLogger(t)
	t.Setenv("ENVIRONMENT", "production")
	logPath := filepath.Join(t.TempDir(), "info.json")

	initLogger(logPath)

	info, err := os.Stat(logPath)
	if err != nil {
		t.Fatalf("expected the preflight to create %s: %v", logPath, err)
	}

	// lumberjack creates rotated files at 0600 unless it can copy the mode off
	// an existing file, so the mode the preflight leaves behind is what keeps
	// the Datadog agent's user able to read the log after a rotation.
	// Assumes the process umask does not strip group and other read bits.
	if perm := info.Mode().Perm(); perm&0o044 == 0 {
		t.Errorf("log file mode is %v, which the Datadog agent's user cannot read", perm)
	}
}

func TestInitLoggerFallsBackToStderrWhenLogFileCannotBeOpened(t *testing.T) {
	restoreLogger(t)
	t.Setenv("ENVIRONMENT", "production")

	// Point the output somewhere inert and capture it: the fallback path must
	// leave the existing output alone rather than swallowing the failure.
	var stderr bytes.Buffer
	log.SetOutput(&stderr)

	// The parent directory does not exist, so the preflight open fails.
	logPath := filepath.Join(t.TempDir(), "missing", "info.json")

	initLogger(logPath)

	if log.StandardLogger().Out != &stderr {
		t.Errorf("output = %T, want the original writer to be left in place", log.StandardLogger().Out)
	}
	if _, err := os.Stat(logPath); !os.IsNotExist(err) {
		t.Errorf("expected no log file at %s", logPath)
	}
	if !strings.Contains(stderr.String(), "falling back to stderr") {
		t.Errorf("expected a diagnostic about the unopenable log file, got %q", stderr.String())
	}
}
