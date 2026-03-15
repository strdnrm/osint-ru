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

// FSSPSource — поиск в банке данных исполнительных производств ФССП
// Использует официальный публичный API ФССП: https://api-ip.fssp.gov.ru/
// (открытый доступ к публичной части БД согласно ФЗ №229-ФЗ ст. 6.1)
type FSSPSource struct {
	client *http.Client
}

func NewFSSPSource() *FSSPSource {
	return &FSSPSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type fsspResponse struct {
	Status   int          `json:"status"`
	Count    int          `json:"count"`
	CountAll string       `json:"countAll"`
	Records  []fsspRecord `json:"records"`
	Message  string       `json:"message"`
}

type fsspRecord struct {
	DebtorName    string `json:"debtor_name"`
	DebtorAddress string `json:"debtor_address"`
	DebtorDOB     string `json:"debtor_dob"`
	ProcessTitle  string `json:"process_title"`
	ProcessDate   string `json:"process_date"`
	Subject       string `json:"subject"`
	Sum           string `json:"sum"`
	DocOrg        string `json:"document_organization"`
	DocType       string `json:"document_type"`
	OfficerName   string `json:"officer_name"`
	StopIP        string `json:"stopIP"`
}

func (s *FSSPSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "ФССП — Банк данных исполнительных производств",
		SourceURL: "https://fssp.gov.ru/iss/ip/",
		Icon:      "gavel",
		Status:    "pending",
	}

	if q.LastName == "" || q.FirstName == "" {
		result.Status = "error"
		result.Error = "Необходимы фамилия и имя"
		return result
	}

	// Официальный публичный API ФССП
	baseURL := "https://api-ip.fssp.gov.ru/api/v1.0/search/physical"
	params := url.Values{
		"lastname":  {strings.TrimSpace(q.LastName)},
		"firstname": {strings.TrimSpace(q.FirstName)},
		"region":    {"-1"}, // все регионы
	}
	if q.MiddleName != "" {
		params.Set("secondname", strings.TrimSpace(q.MiddleName))
	}
	if q.BirthDate != "" {
		params.Set("birthdate", q.BirthDate)
	}

	reqURL := baseURL + "?" + params.Encode()
	result.SearchedURL = reqURL

	resp, err := s.client.Get(reqURL)
	if err != nil {
		// Fallback: формируем ссылку на ручной поиск
		result.Status = "manual"
		result.Error = fmt.Sprintf("API недоступен, выполните поиск вручную")
		result.SearchedURL = fmt.Sprintf("https://fssp.gov.ru/iss/ip/?is[last_name]=%s&is[first_name]=%s&is[patronymic]=%s&is[date]=%s&is[region]=-1&is[iss]=1&is[page]=1",
			url.QueryEscape(q.LastName), url.QueryEscape(q.FirstName),
			url.QueryEscape(q.MiddleName), url.QueryEscape(q.BirthDate))
		return result
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		result.Status = "error"
		result.Error = "Ошибка чтения ответа"
		return result
	}

	var data fsspResponse
	if err := json.Unmarshal(body, &data); err != nil {
		result.Status = "error"
		result.Error = fmt.Sprintf("Ошибка разбора ответа: %v", err)
		return result
	}

	if data.Count == 0 {
		result.Status = "not_found"
		return result
	}

	result.Status = "found"
	for _, rec := range data.Records {
		fields := []models.Field{
			{Label: "ФИО должника", Value: rec.DebtorName, Kind: "text"},
			{Label: "Дата рождения", Value: rec.DebtorDOB, Kind: "date"},
			{Label: "Адрес / место рождения", Value: rec.DebtorAddress, Kind: "text"},
			{Label: "Номер производства", Value: rec.ProcessTitle, Kind: "text"},
			{Label: "Дата возбуждения", Value: rec.ProcessDate, Kind: "date"},
			{Label: "Предмет взыскания", Value: rec.Subject, Kind: "badge"},
			{Label: "Сумма задолженности", Value: rec.Sum + " руб.", Kind: "money"},
			{Label: "Отдел СП", Value: rec.DocOrg, Kind: "text"},
			{Label: "Пристав", Value: rec.OfficerName, Kind: "text"},
		}
		if rec.StopIP != "" {
			fields = append(fields, models.Field{Label: "Окончание ИП", Value: rec.StopIP, Kind: "badge"})
		}
		result.Records = append(result.Records, models.Record{
			Title:  rec.ProcessTitle,
			Fields: fields,
			Tags:   []string{"ФССП", "исполнительное производство"},
		})
	}

	return result
}
