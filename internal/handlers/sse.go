// Package handlers — Server-Sent Events patching for Datastar.
//
// The vendored Datastar v1.0.4 client only understands SSE responses whose
// events are named `datastar-patch-elements` / `datastar-patch-signals`.
// The older `Datastar-Selector`/`Datastar-Mode` HTTP headers are ignored by
// this version, so fragments must be streamed as events instead.
package handlers

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/a-h/templ"
)

// sseBegin switches the response to an SSE stream.
func sseBegin(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
}

// sseFlush flushes buffered events so Datastar can apply them immediately.
func sseFlush(w http.ResponseWriter) {
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

// sseSignals emits a `datastar-patch-signals` event with a JSON signal patch,
// e.g. `{"modalOpen": false}`.
func sseSignals(w http.ResponseWriter, signals string) {
	fmt.Fprint(w, "event: datastar-patch-signals\n")
	fmt.Fprintf(w, "data: signals %s\n\n", signals)
}

// sseElements emits a `datastar-patch-elements` event that swaps the inner
// (or other mode) content of selector with the rendered component.
func sseElements(w http.ResponseWriter, r *http.Request, selector, mode string, c templ.Component) {
	var buf bytes.Buffer
	if err := c.Render(r.Context(), &buf); err != nil {
		log.Printf("render sse fragment: %v", err)
		return
	}
	fmt.Fprint(w, "event: datastar-patch-elements\n")
	if selector != "" {
		fmt.Fprintf(w, "data: selector %s\n", selector)
	}
	if mode != "" {
		fmt.Fprintf(w, "data: mode %s\n", mode)
	}
	for _, line := range strings.Split(buf.String(), "\n") {
		fmt.Fprintf(w, "data: elements %s\n", line)
	}
	fmt.Fprint(w, "\n")
}

// respondFragment sends a single component swap over SSE. It is the
// Datastar counterpart of writing a plain HTML fragment.
func respondFragment(w http.ResponseWriter, r *http.Request, selector string, c templ.Component) {
	sseBegin(w)
	sseElements(w, r, selector, "inner", c)
	sseFlush(w)
}
