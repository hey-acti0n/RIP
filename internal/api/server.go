package api

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"rip/internal/app/handler"
	"rip/internal/app/repository"
)

// mustParseTemplates searches for the templates directory and parses base templates
func mustParseTemplates() *template.Template {
	var base string
	possiblePaths := []string{
		"templates",
		"RIP/templates",
		filepath.Join("..", "templates"),
	}
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			base = p
			break
		}
	}
	if base == "" {
		log.Fatal("Не удалось найти папку templates. Проверьте, что вы запускаете программу из правильной директории.")
	}
	t := template.Must(template.ParseFiles(
		filepath.Join(base, "catalog.html"),
		filepath.Join(base, "calc.html"),
		filepath.Join(base, "detail.html"),
	))
	return t
}

// Start configures and starts HTTP server
func Start() {
	tmpl := mustParseTemplates()
	minioHost := repository.Getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
	assetsBase := fmt.Sprintf("%s/images/icons", minioHost)

	srv := handler.NewServer(assetsBase, tmpl)

	var staticDir string
	possibleStaticPaths := []string{
		"static",
		"RIP/static",
		filepath.Join("..", "static"),
	}
	for _, p := range possibleStaticPaths {
		if _, err := os.Stat(p); err == nil {
			staticDir = p
			break
		}
	}
	if staticDir == "" {
		staticDir = "static"
	}
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// pages
	http.HandleFunc("/", srv.HandleCatalog)
	http.HandleFunc("/calc", srv.HandleCalc)
	http.HandleFunc("/detail/", srv.HandleDetail)

	// actions
	http.HandleFunc("/add", srv.HandleAdd)
	http.HandleFunc("/clear", srv.HandleClear)

	// API
	http.HandleFunc("/api/services", srv.ApiServices)
	http.HandleFunc("/api/add", srv.ApiAddToCart)
	http.HandleFunc("/api/cart", srv.ApiCart)
	http.HandleFunc("/api/clear", srv.ApiClearCart)
	http.HandleFunc("/api/calc", srv.ApiCalc)

	// DB
	db := repository.InitDB()
	srv.AttachDB(db)
	repository.MountORMRoutes(db)

	addr := repository.Getenv("ADDR", ":8080")
	log.Printf("UltraRezina server listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
