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

// FNSSource — поиск ИП по ФИО через ФНС ЕГРИП (открытые данные)
// API: https://egrul.nalog.ru/ (публичный реестр, открытый по ФЗ №129-ФЗ)
type FNSSource struct {
	client *http.Client
}

func NewFNSSource() *FNSSource {
	return &FNSSource{
		client: &http.Client{
			Timeout: 30 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return nil
			},
		},
	}
}

type fnsSearchResp struct {
	T  string       `json:"t"` // token
	Rows []fnsRow   `json:"rows"`
	Total int        `json:"total"`
}

type fnsRow struct {
	N    string `json:"n"`    // наименование/ФИО
	C    string `json:"c"`    // статус
	R    string `json:"r"`    // регион
	Inno string `json:"inno"` // ИНН
	K    string `json:"k"`    // ОГРН/ОГРНИП
	O    string `json:"o"`    // ОКВЭД
	P    string `json:"p"`    // дата регистрации
	T    string `json:"t"`    // тип: "Ю" юрлицо, "Ф" ИП
	A    string `json:"a"`    // адрес
}

func (s *FNSSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "ФНС — ЕГРИП (Индивидуальные предприниматели)",
		SourceURL: "https://egrul.nalog.ru/",
		Icon:      "building",
		Status:    "pending",
	}

	// Формируем запрос на поиск по ФИО или ИНН
	searchQuery := ""
	if q.INN != "" {
		searchQuery = q.INN
	} else if q.LastName != "" {
		parts := []string{q.LastName}
		if q.FirstName != "" {
			parts = append(parts, q.FirstName)
		}
		if q.MiddleName != "" {
			parts = append(parts, q.MiddleName)
		}
		searchQuery = strings.Join(parts, " ")
	}

	if searchQuery == "" {
		result.Status = "error"
		result.Error = "Недостаточно данных для поиска (нужны ФИО или ИНН)"
		return result
	}

	// Шаг 1: получаем токен поиска
	tokenURL := "https://egrul.nalog.ru/search-result?query=" + url.QueryEscape(searchQuery) + "&region=&_=&page=1"
	result.SearchedURL = tokenURL

	req, _ := http.NewRequest("GET", tokenURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OSINT-RU/1.0)")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Referer", "https://egrul.nalog.ru/")

	resp, err := s.client.Do(req)
	if err != nil {
		result.Status = "manual"
		result.Error = "Сервис ФНС недоступен, выполните поиск вручную"
		result.SearchedURL = fmt.Sprintf("https://egrul.nalog.ru/#%s", url.QueryEscape(searchQuery))
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "error"
		result.Error = "Ошибка чтения ответа ФНС"
		return result
	}

	var data fnsSearchResp
	if err := json.Unmarshal(body, &data); err != nil {
		result.Status = "manual"
		result.Error = "Ответ ФНС не в ожидаемом формате"
		result.SearchedURL = fmt.Sprintf("https://egrul.nalog.ru/#%s", url.QueryEscape(searchQuery))
		return result
	}

	// Фильтруем только ИП (тип "Ф")
	ipRecords := []fnsRow{}
	for _, row := range data.Rows {
		if row.T == "Ф" || strings.Contains(strings.ToUpper(row.N), strings.ToUpper(q.LastName)) {
			ipRecords = append(ipRecords, row)
		}
	}

	if len(ipRecords) == 0 && len(data.Rows) == 0 {
		result.Status = "not_found"
		return result
	}

	result.Status = "found"
	rowsToProcess := data.Rows
	if len(ipRecords) > 0 {
		rowsToProcess = ipRecords
	}

	for _, row := range rowsToProcess {
		status := row.C
		tags := []string{"ЕГРИП/ЕГРЮЛ"}
		if row.T == "Ф" {
			tags = append(tags, "ИП")
		} else {
			tags = append(tags, "Юр. лицо")
		}

		fields := []models.Field{
			{Label: "Наименование / ФИО", Value: row.N, Kind: "text"},
			{Label: "ИНН", Value: row.Inno, Kind: "text"},
			{Label: "ОГРН/ОГРНИП", Value: row.K, Kind: "text"},
			{Label: "Статус", Value: status, Kind: "badge"},
			{Label: "Регион", Value: row.R, Kind: "text"},
			{Label: "Основной ОКВЭД", Value: row.O, Kind: "text"},
			{Label: "Дата регистрации", Value: row.P, Kind: "date"},
		}
		if row.A != "" {
			fields = append(fields, models.Field{Label: "Адрес", Value: row.A, Kind: "text"})
		}

		result.Records = append(result.Records, models.Record{
			Title:  row.N,
			Fields: fields,
			Tags:   tags,
			SourceLink: fmt.Sprintf("https://egrul.nalog.ru/#%s", url.QueryEscape(row.K)),
		})
	}

	return result
}
