package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"osint-ru/internal/models"
)

// FedresursSource — поиск в Федресурсе (bankrotstvo.fedresurs.ru)
// Публичный реестр: банкротства, ликвидации, сведения о юрлицах
type FedresursSource struct {
	client *http.Client
}

func NewFedresursSource() *FedresursSource {
	return &FedresursSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type fedresursResp struct {
	Data struct {
		Messages []fedresursMsg `json:"Messages"`
		Total    int            `json:"Total"`
	} `json:"Data"`
}

type fedresursMsg struct {
	ID          int    `json:"ID"`
	Title       string `json:"Title"`
	DatePublish string `json:"DatePublish"`
	Type        string `json:"Type"`
	Url         string `json:"Url"`
	Debtor      struct {
		Name string `json:"FullName"`
		INN  string `json:"INN"`
		Type string `json:"Type"` // "2" — физлицо
	} `json:"Debtor"`
	ArbitrManager struct {
		Name string `json:"FullName"`
		INN  string `json:"INN"`
	} `json:"ArbitrManager"`
}

func (s *FedresursSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "Федресурс — банкротства и ликвидации",
		SourceURL: "https://fedresurs.ru",
		Icon:      "alert-triangle",
		Status:    "pending",
	}

	fio := strings.TrimSpace(q.LastName + " " + q.FirstName + " " + q.MiddleName)
	searchTerm := fio
	if q.INN != "" {
		searchTerm = q.INN
	}

	if searchTerm == "" {
		result.Status = "error"
		result.Error = "Недостаточно данных"
		return result
	}

	// API Федресурса
	apiURL := fmt.Sprintf("https://fedresurs.ru/backend/bankrupcy/search?searchString=%s&offset=0&limit=20&onlyActual=false",
		url.QueryEscape(searchTerm))
	result.SearchedURL = apiURL

	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OSINT-RU/1.0)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://fedresurs.ru/search/persons")

	resp, err := s.client.Do(req)
	if err != nil {
		result.Status = "manual"
		result.Error = "Федресурс недоступен автоматически"
		result.SearchedURL = fmt.Sprintf("https://fedresurs.ru/search/persons?searchString=%s", url.QueryEscape(searchTerm))
		result.Records = buildManualRecord("Федресурс — Банкротства физлиц", result.SearchedURL,
			fmt.Sprintf("Поиск: %s", searchTerm))
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data fedresursResp
	if err := json.Unmarshal(body, &data); err != nil {
		result.Status = "manual"
		result.Error = "Не удалось разобрать ответ Федресурса"
		result.SearchedURL = fmt.Sprintf("https://fedresurs.ru/search/persons?searchString=%s", url.QueryEscape(searchTerm))
		result.Records = buildManualRecord("Открыть Федресурс", result.SearchedURL, "Ручной поиск")
		return result
	}

	if data.Data.Total == 0 {
		result.Status = "not_found"
		return result
	}

	result.Status = "found"
	for _, msg := range data.Data.Messages {
		debtorName := msg.Debtor.Name
		if debtorName == "" {
			debtorName = msg.Title
		}
		fields := []models.Field{
			{Label: "Должник", Value: debtorName, Kind: "text"},
			{Label: "Тип сообщения", Value: msg.Type, Kind: "badge"},
			{Label: "Дата публикации", Value: msg.DatePublish, Kind: "date"},
			{Label: "ИНН должника", Value: msg.Debtor.INN, Kind: "text"},
		}
		if msg.ArbitrManager.Name != "" {
			fields = append(fields, models.Field{Label: "Арбитражный управляющий", Value: msg.ArbitrManager.Name, Kind: "text"})
		}
		msgURL := msg.Url
		if msgURL == "" {
			msgURL = fmt.Sprintf("https://fedresurs.ru/message/%d", msg.ID)
		}
		result.Records = append(result.Records, models.Record{
			Title:      msg.Title,
			Fields:     fields,
			Tags:       []string{"банкротство", "Федресурс"},
			SourceLink: msgURL,
		})
	}

	return result
}

func buildManualRecord(title, link, hint string) []models.Record {
	return []models.Record{
		{
			Title: title,
			Fields: []models.Field{
				{Label: "Ссылка", Value: link, Kind: "link"},
				{Label: "Подсказка", Value: hint, Kind: "text"},
			},
			Tags: []string{"ручной поиск"},
		},
	}
}
