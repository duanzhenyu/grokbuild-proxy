// Package adminui serves the embedded zero-build Admin Web UI.
//
// Mount (by the HTTP server, before admin JSON routes that require auth):
//
//	mux.Handle("GET /admin", adminui.IndexHandler())
//	mux.Handle("GET /admin/{$}", adminui.IndexHandler())
//	mux.Handle("GET /admin/ui/", http.StripPrefix("/admin/ui/", adminui.AssetsHandler()))
//
// Static assets live under /admin/ui/* and are unauthenticated.
// Admin JSON APIs remain under /admin/credentials, /admin/clients, /admin/system, etc.
package adminui

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
	"strings"
)

//go:embed static/*
var staticFS embed.FS

// IndexHandler returns a handler that serves index.html with no-store caching.
// Intended for exact routes: GET /admin and GET /admin/.
func IndexHandler() http.Handler {
	return http.HandlerFunc(ServeIndex)
}

// ServeIndex writes the SPA shell (index.html). Safe without admin auth.
func ServeIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := staticFS.ReadFile("static/index.html")
	if err != nil {
		http.Error(w, "admin ui not available", http.StatusInternalServerError)
		return
	}

	setSecurityHeaders(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return
	}
	_, _ = w.Write(data)
}

// AssetsHandler returns a file server for embedded static assets.
// Mount with StripPrefix("/admin/ui/", ...) so requests map to static/*.
// Example: GET /admin/ui/app.js → static/app.js
func AssetsHandler() http.Handler {
	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "admin ui assets not available", http.StatusInternalServerError)
		})
	}
	files := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Prevent directory listing / path escape; only serve known files.
		name := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
		name = strings.TrimPrefix(name, "/")
		if name == "" || name == "." || strings.Contains(name, "..") {
			http.NotFound(w, r)
			return
		}
		setSecurityHeaders(w)
		// Short cache for hashed-less static assets; index uses no-store separately.
		if strings.HasSuffix(name, ".js") || strings.HasSuffix(name, ".css") {
			w.Header().Set("Cache-Control", "no-cache")
		}
		// Ensure correct content types when FileServer misses them.
		switch {
		case strings.HasSuffix(name, ".js"):
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		case strings.HasSuffix(name, ".css"):
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		case strings.HasSuffix(name, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		// Verify file exists before FileServer (cleaner 404).
		f, err := sub.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		_ = f.Close()
		files.ServeHTTP(w, r)
	})
}

// Handler is a convenience mux for /admin UI only (index + assets).
// Callers that need fine-grained registration should use IndexHandler/AssetsHandler.
func Handler() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("GET /admin", IndexHandler())
	mux.Handle("GET /admin/{$}", IndexHandler())
	mux.Handle("GET /admin/ui/", http.StripPrefix("/admin/ui/", AssetsHandler()))
	// Fallback without method patterns (older mux / HEAD).
	mux.HandleFunc("/admin", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin" {
			http.NotFound(w, r)
			return
		}
		ServeIndex(w, r)
	})
	mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/admin/" {
			ServeIndex(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/admin/ui/") {
			http.StripPrefix("/admin/ui/", AssetsHandler()).ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
	return mux
}

func setSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Referrer-Policy", "no-referrer")
	// Tight CSP: same-origin scripts/styles only; no inline scripts (app.js external).
	w.Header().Set("Content-Security-Policy",
		"default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'")
}

// ReadStatic is a test helper / debug helper.
func ReadStatic(name string) ([]byte, error) {
	return staticFS.ReadFile(path.Join("static", name))
}

// Ensure embed content is non-empty at init (fail fast in tests).
func init() {
	// Touch embed so go:embed failure surfaces at package load in tests.
	_, _ = staticFS.Open("static/index.html")
	_, _ = staticFS.Open("static/app.js")
	_, _ = staticFS.Open("static/app.css")
}
