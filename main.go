package main

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"

	"osint-ru/internal/handlers"
)

//go:embed static
var staticFiles embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()

	// API маршруты
	mux.HandleFunc("/api/search", corsMiddleware(handlers.SearchHandler))
	mux.HandleFunc("/api/health", corsMiddleware(handlers.HealthHandler))

	// Статические файлы
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatal(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Printf("OSINT-RU запущен на порту %s", port)
	log.Printf("Открой: http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next(w, r)
	}
}
