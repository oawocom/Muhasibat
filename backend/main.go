package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"oawo-muhasibat/internal/api"
	"oawo-muhasibat/internal/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	db, err := store.Connect()
	if err != nil {
		log.Fatalf("Verilənlər bazasına qoşulmaq mümkün olmadı: %v", err)
	}

	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// CORS for standalone frontends / API clients.
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	r.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok", "product": "oawo-muhasibat"}) })

	server := api.New(db)
	server.RegisterRoutes(r)

	// Serve the embedded SPA.
	sub, _ := fs.Sub(webFS, "web")
	serveSPA(r, sub)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("🚀 OAWO Mühasibat http://0.0.0.0:%s ünvanında işə düşdü", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}

// serveSPA serves static assets and falls back to index.html for app routes.
func serveSPA(r *gin.Engine, static fs.FS) {
	fileServer := http.FileServer(http.FS(static))
	index, _ := fs.ReadFile(static, "index.html")

	r.NoRoute(func(c *gin.Context) {
		p := c.Request.URL.Path
		if strings.HasPrefix(p, "/api") {
			c.JSON(404, gin.H{"detail": "Endpoint tapılmadı"})
			return
		}
		// Try to serve a real file (js, css, assets).
		if p != "/" {
			if _, err := fs.Stat(static, strings.TrimPrefix(p, "/")); err == nil {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.Data(200, "text/html; charset=utf-8", index)
	})
}
