package handlers

import (
	_ "embed"
	"net/http"

	"github.com/go-chi/chi/v5"
)

//go:embed static/datastar.js
var datastarJS []byte

//go:embed static/goinvoice.css
var goinvoiceCSS []byte

//go:embed static/goinvoice.js
var goinvoiceJS []byte

//go:embed static/pattika/css/tokens.css
var pattikaTokens []byte

//go:embed static/pattika/css/base.css
var pattikaBase []byte

//go:embed static/pattika/css/motion.css
var pattikaMotion []byte

//go:embed static/pattika/css/layout.css
var pattikaLayout []byte

//go:embed static/pattika/css/sidebar.css
var pattikaSidebar []byte

//go:embed static/pattika/css/components.css
var pattikaComponents []byte

//go:embed static/pattika/css/utilities.css
var pattikaUtilities []byte

//go:embed static/pattika/js/theme.js
var pattikaThemeJS []byte

// pattikaFiles maps URL paths to bundled file contents. The design system
// is copied verbatim from the pattika project (design tokens, app shell,
// sidebar, components, motion); goinvoice.css holds app-specific additions.
var pattikaFiles = map[string]struct {
	content     []byte
	contentType string
}{
	"/static/pattika/css/tokens.css":     {pattikaTokens, "text/css; charset=utf-8"},
	"/static/pattika/css/base.css":       {pattikaBase, "text/css; charset=utf-8"},
	"/static/pattika/css/motion.css":     {pattikaMotion, "text/css; charset=utf-8"},
	"/static/pattika/css/layout.css":     {pattikaLayout, "text/css; charset=utf-8"},
	"/static/pattika/css/sidebar.css":    {pattikaSidebar, "text/css; charset=utf-8"},
	"/static/pattika/css/components.css": {pattikaComponents, "text/css; charset=utf-8"},
	"/static/pattika/css/utilities.css":  {pattikaUtilities, "text/css; charset=utf-8"},
	"/static/pattika/js/theme.js":        {pattikaThemeJS, "text/javascript; charset=utf-8"},
	"/static/goinvoice.css":              {goinvoiceCSS, "text/css; charset=utf-8"},
	"/static/goinvoice.js":               {goinvoiceJS, "text/javascript; charset=utf-8"},
}

// Static serves the bundled assets so the binary stays self-contained.
func Static(r chi.Router) {
	r.Get("/static/datastar.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(datastarJS)
	})
	for route, f := range pattikaFiles {
		route, f := route, f
		r.Get(route, func(w http.ResponseWriter, req *http.Request) {
			// Long cache is safe: URLs are versioned with ?v=N in Layout.
			w.Header().Set("Content-Type", f.contentType)
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(f.content)
		})
	}
}
