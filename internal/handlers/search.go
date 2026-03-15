package handlers

import (
	"encoding/json"
	"net/http"
	"sync"

	"osint-ru/internal/models"
	"osint-ru/internal/sources"
)

type Searcher interface {
	Search(q models.SearchQuery) models.SourceResult
}

// SearchHandler — основной обработчик поиска
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var q models.SearchQuery
	if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
		http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	searchers := []Searcher{
		sources.NewFSSPSource(),
		sources.NewFNSSource(),
		sources.NewINNSource(),
		sources.NewFedresursSource(),
		sources.NewRosreestrSource(),
		sources.NewGovLinksSource(),
	}

	results := make([]models.SourceResult, len(searchers))
	var wg sync.WaitGroup

	for i, s := range searchers {
		wg.Add(1)
		go func(idx int, src Searcher) {
			defer wg.Done()
			results[idx] = src.Search(q)
		}(i, s)
	}

	wg.Wait()

	resp := models.SearchResponse{
		Query:   q,
		Results: results,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(resp)
}

// HealthHandler — простая проверка состояния
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
