package sources

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"osint-ru/internal/models"
)

// SudrfSource — поиск по судебным делам через ГАС Правосудие
// Публичный доступ к судебным решениям согласно ФЗ №262-ФЗ "Об обеспечении доступа к информации о деятельности судов"
type SudrfSource struct {
	client *http.Client
}

func NewSudrfSource() *SudrfSource {
	return &SudrfSource{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
	}
}

func (s *SudrfSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "ГАС Правосудие — судебные дела",
		SourceURL: "https://sudrf.ru",
		Icon:      "scale",
		Status:    "pending",
	}

	if q.LastName == "" {
		result.Status = "error"
		result.Error = "Необходима фамилия для поиска"
		return result
	}

	fio := q.LastName
	if q.FirstName != "" {
		fio += " " + q.FirstName
	}
	if q.MiddleName != "" {
		fio += " " + q.MiddleName
	}

	// Формируем URL для поиска на портале Правосудие
	searchURL := fmt.Sprintf(
		"https://bsr.sudrf.ru/bigs/portal.html#%s",
		url.QueryEscape(fmt.Sprintf(`{"query":"%s","type":"OVERALL"}`, fio)),
	)
	result.SearchedURL = searchURL

	// Пытаемся выполнить поиск через API ГАС Правосудие
	apiURL := "https://bsr.sudrf.ru/bigs/search.action"
	formData := url.Values{
		"request": {fmt.Sprintf(`{"query":"%s","operator":"AND","type":"OVERALL","page":0,"perPage":25}`, fio)},
	}

	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OSINT-RU/1.0)")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Origin", "https://bsr.sudrf.ru")
	req.Header.Set("Referer", "https://bsr.sudrf.ru/bigs/portal.html")

	resp, err := s.client.Do(req)
	if err != nil {
		// ГАС может быть недоступен — возвращаем ссылки для ручного поиска
		result.Status = "manual"
		result.Error = "Портал ГАС Правосудие недоступен автоматически"
		result.SearchedURL = fmt.Sprintf("https://bsr.sudrf.ru/bigs/portal.html#%s",
			url.QueryEscape(fmt.Sprintf(`{"query":"%s","type":"OVERALL"}`, fio)))
		result.Records = []models.Record{
			{
				Title: "Ручной поиск на ГАС Правосудие",
				Fields: []models.Field{
					{Label: "Ссылка для поиска", Value: result.SearchedURL, Kind: "link"},
					{Label: "Поиск по делам", Value: fmt.Sprintf("https://sudrf.ru/index.php?id=300&searchtype=sp"), Kind: "link"},
					{Label: "Подсказка", Value: fmt.Sprintf("Введите ФИО: %s", fio), Kind: "text"},
				},
				Tags: []string{"ручной поиск"},
			},
		}
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Парсим простой вариант - ищем данные о результатах
	if strings.Contains(bodyStr, `"total":0`) || strings.Contains(bodyStr, `"count":0`) {
		result.Status = "not_found"
		return result
	}

	if len(bodyStr) > 100 {
		result.Status = "found"
		result.Records = []models.Record{
			{
				Title: fmt.Sprintf("Судебные дела по запросу: %s", fio),
				Fields: []models.Field{
					{Label: "Подробности", Value: "Найдены записи в базе ГАС Правосудие", Kind: "text"},
					{Label: "Ссылка для просмотра", Value: result.SearchedURL, Kind: "link"},
				},
				Tags: []string{"судебные дела"},
			},
		}
	} else {
		result.Status = "manual"
		result.SearchedURL = fmt.Sprintf("https://bsr.sudrf.ru/bigs/portal.html#%s",
			url.QueryEscape(fmt.Sprintf(`{"query":"%s","type":"OVERALL"}`, fio)))
		result.Records = []models.Record{
			{
				Title: "Открыть поиск на ГАС Правосудие",
				Fields: []models.Field{
					{Label: "Поиск по ФИО", Value: result.SearchedURL, Kind: "link"},
				},
				Tags: []string{"ручной поиск"},
			},
		}
	}

	return result
}
