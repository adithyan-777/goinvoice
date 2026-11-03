// Command server runs the goinvoice HTTP server built with Gin.
//
// Endpoints:
//
//	POST   /invoices[?dry_run=true|html=true]   create (calculate + validate + store)
//	GET    /invoices                            list summaries
//	GET    /invoices/:id                        full envelope as JSON (source of truth)
//	GET    /invoices/:id/html[?locale|watermark|sandbox]  printable HTML (→ PDF)
//	DELETE /invoices/:id                        delete (no legal cancellation semantics)
//	GET    /healthz                             liveness probe
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"

	"github.com/neo/goinvoice/internal/handlers"
	"github.com/neo/goinvoice/internal/store"
)

func main() {
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	entries := store.New()
	handlers.New(entries).Register(r)

	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Dump a full envelope to disk for inspection.
	r.GET("/invoices/:id/envelope.json", func(c *gin.Context) {
		env, ok := entries.Get(c.Param("id"))
		if !ok {
			c.JSON(http.StatusNotFound, gin.H{"error": "invoice not found"})
			return
		}
		c.Header("Content-Disposition", "attachment; filename="+c.Param("id")+".json")
		c.JSON(http.StatusOK, env)
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("goinvoice listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
