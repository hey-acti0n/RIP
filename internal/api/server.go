package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"RIP/internal/app"
	"RIP/internal/app/handler"
)

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func findStaticDir() string {
	possible := []string{"static", "RIP/static", filepath.Join("..", "static")}
	for _, p := range possible {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "static"
}

func NewHTTPServer() *http.ServeMux {
	minioHost := getenv("MINIO_PUBLIC_ENDPOINT", "http://localhost:9000")
	assetsBase := fmt.Sprintf("%s/%s", minioHost, "images")

	srv := &handler.Server{
		Store:      app.NewStore(),
		AssetsBase: assetsBase,
		Templates:  handler.MustParseTemplates(),
	}

	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir(findStaticDir()))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	mux.HandleFunc("/", srv.HandleCatalog)
	mux.HandleFunc("/calc", srv.HandleCalc)
	mux.HandleFunc("/detail/", srv.HandleDetail)

	mux.HandleFunc("/api/services", srv.ApiServices)
	mux.HandleFunc("/api/add", srv.ApiAddToCart)
	mux.HandleFunc("/api/cart", srv.ApiCart)
	mux.HandleFunc("/api/clear", srv.ApiClearCart)
	mux.HandleFunc("/api/calc", srv.ApiCalc)

	// SSR actions
	mux.HandleFunc("/add", srv.HandleAdd)
	mux.HandleFunc("/clear", srv.HandleClear)
	return mux
}

func Run() error {
	addr := getenv("ADDR", ":8080")
	log.Printf("UltraRezina server listening on %s", addr)
	return http.ListenAndServe(addr, NewHTTPServer())
}
