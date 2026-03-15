package handlers

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync"

	"osint-ru/internal/models"
	"osint-ru/internal/sources"
)

type Searcher interface {
	Search(q models.SearchQuery) models.SourceResult
}

// SearchHandler — основной обработчик поиска
// Принимает как JSON, так и multipart/form-data (для загрузки фото)
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	var q models.SearchQuery

	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		// Парсим форму с файлом
		if err := r.ParseMultipartForm(10 << 20); err != nil { // 10 MB max
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}
		q.LastName   = r.FormValue("last_name")
		q.FirstName  = r.FormValue("first_name")
		q.MiddleName = r.FormValue("middle_name")
		q.BirthDate  = r.FormValue("birth_date")
		q.INN        = r.FormValue("inn")
		q.Region     = r.FormValue("region")
		q.PhotoURL   = r.FormValue("photo_url")

		// Читаем загруженный файл фото
		file, _, err := r.FormFile("photo")
		if err == nil && file != nil {
			defer file.Close()
			data, err := io.ReadAll(file)
			if err == nil {
				q.PhotoBase64 = base64.StdEncoding.EncodeToString(data)
			}
		}
	} else {
		// JSON-запрос
		if err := json.NewDecoder(r.Body).Decode(&q); err != nil {
			http.Error(w, "Bad Request: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	allSources := []Searcher{
		sources.NewFSSPSource(),
		sources.NewFNSSource(),
		sources.NewINNSource(),
		sources.NewFedresursSource(),
		sources.NewRosreestrSource(),
		sources.NewGovLinksSource(),
		sources.NewSocialsSource(),
		sources.NewPhotoSource(),
	}

	results := make([]models.SourceResult, len(allSources))
	var wg sync.WaitGroup

	for i, s := range allSources {
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

	// Не отдаём base64 обратно клиенту (экономим трафик)
	resp.Query.PhotoBase64 = ""

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	json.NewEncoder(w).Encode(resp)
}

// HealthHandler — простая проверка состояния
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
