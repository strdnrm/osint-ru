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

// SocialsSource — поиск профилей в социальных сетях
// VK: публичный API search.getPeople (без токена работает ограниченно)
// Остальные: прямые поисковые ссылки
type SocialsSource struct {
	client *http.Client
}

func NewSocialsSource() *SocialsSource {
	return &SocialsSource{
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// ── VK API structs ────────────────────────────────────────────────────────────

type vkSearchResp struct {
	Response struct {
		Count int       `json:"count"`
		Items []vkUser  `json:"items"`
	} `json:"response"`
	Error *vkError `json:"error"`
}

type vkUser struct {
	ID         int    `json:"id"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	IsClosed   bool   `json:"is_closed"`
	City       *vkCity `json:"city"`
	BDate      string `json:"bdate"`
	Photo      string `json:"photo_200"`
	Domain     string `json:"domain"`
}

type vkCity struct {
	Title string `json:"title"`
}

type vkError struct {
	Code    int    `json:"error_code"`
	Message string `json:"error_msg"`
}

// ─────────────────────────────────────────────────────────────────────────────

func (s *SocialsSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "Социальные сети — поиск профилей",
		SourceURL: "https://vk.com",
		Icon:      "users",
		Status:    "pending",
	}

	fio := strings.TrimSpace(q.LastName + " " + q.FirstName)
	if fio == " " || fio == "" {
		result.Status = "skip"
		result.Error = "Необходима фамилия для поиска в соцсетях"
		return result
	}

	fullFio := strings.TrimSpace(q.LastName + " " + q.FirstName + " " + q.MiddleName)
	firstName := strings.TrimSpace(q.FirstName)
	lastName  := strings.TrimSpace(q.LastName)

	var records []models.Record

	// ── 1. ВКонтакте — пытаемся API ──────────────────────────────────────────
	vkRecords := s.searchVK(firstName, lastName, q.BirthDate)
	records = append(records, vkRecords...)

	// ── 2. Прямые ссылки на поиск в каждой соцсети ───────────────────────────
	records = append(records, buildSocialLinks(fullFio, firstName, lastName, q.BirthDate)...)

	if len(records) == 0 {
		result.Status = "not_found"
		return result
	}

	result.Status = "found"
	result.Records = records
	return result
}

// searchVK — поиск через открытый VK API (search.getPeople)
func (s *SocialsSource) searchVK(firstName, lastName, bdate string) []models.Record {
	if firstName == "" && lastName == "" {
		return nil
	}

	// VK API users.search — публичный эндпоинт, работает без токена с ограничениями
	params := url.Values{
		"q":       {strings.TrimSpace(firstName + " " + lastName)},
		"count":   {"10"},
		"fields":  {"city,bdate,photo_200,domain"},
		"v":       {"5.199"},
	}

	// Пробуем без токена сначала
	apiURL := "https://api.vk.com/method/users.search?" + params.Encode()
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OSINT-RU/1.0)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var data vkSearchResp
	if err := json.Unmarshal(body, &data); err != nil || data.Error != nil {
		return nil
	}

	if data.Response.Count == 0 {
		return nil
	}

	var records []models.Record
	for _, u := range data.Response.Items {
		profileURL := fmt.Sprintf("https://vk.com/id%d", u.ID)
		if u.Domain != "" {
			profileURL = fmt.Sprintf("https://vk.com/%s", u.Domain)
		}

		city := "—"
		if u.City != nil {
			city = u.City.Title
		}

		status := "Открытый профиль"
		if u.IsClosed {
			status = "Закрытый профиль"
		}

		fields := []models.Field{
			{Label: "Имя", Value: u.FirstName + " " + u.LastName, Kind: "text"},
			{Label: "ID", Value: fmt.Sprintf("%d", u.ID), Kind: "text"},
			{Label: "Профиль", Value: profileURL, Kind: "link"},
			{Label: "Город", Value: city, Kind: "text"},
			{Label: "Статус", Value: status, Kind: "badge"},
		}
		if u.BDate != "" {
			fields = append(fields, models.Field{Label: "Дата рождения", Value: u.BDate, Kind: "date"})
		}

		records = append(records, models.Record{
			Title:      fmt.Sprintf("ВКонтакте: %s %s", u.FirstName, u.LastName),
			Fields:     fields,
			Tags:       []string{"ВКонтакте", "API"},
			SourceLink: profileURL,
		})
	}

	return records
}

// buildSocialLinks — прямые поисковые ссылки по всем соцсетям
func buildSocialLinks(fullFio, firstName, lastName, bdate string) []models.Record {
	q := url.QueryEscape(fullFio)
	qName := url.QueryEscape(firstName + " " + lastName)

	// Telegram — поиск через @username невозможен без аккаунта,
	// но можно искать через поисковики
	tgQuery := url.QueryEscape(fmt.Sprintf("site:t.me \"%s\"", fullFio))
	igQuery := url.QueryEscape(fmt.Sprintf("site:instagram.com \"%s\"", fullFio))

	return []models.Record{
		{
			Title: "ВКонтакте — поиск людей",
			Fields: []models.Field{
				{Label: "Поиск", Value: fmt.Sprintf("https://vk.com/search?c%%5Bname%%5D=1&c%%5Bsection%%5D=people&q=%s", qName), Kind: "link"},
				{Label: "Глобальный поиск", Value: fmt.Sprintf("https://vk.com/search?q=%s&section=people", q), Kind: "link"},
			},
			Tags:       []string{"ВКонтакте", "ссылки"},
			SourceLink: fmt.Sprintf("https://vk.com/search?c%%5Bsection%%5D=people&q=%s", qName),
		},
		{
			Title: "Telegram — поиск через Google",
			Fields: []models.Field{
				{Label: "Google-поиск по t.me", Value: fmt.Sprintf("https://www.google.com/search?q=%s", tgQuery), Kind: "link"},
				{Label: "Яндекс-поиск по t.me", Value: fmt.Sprintf("https://yandex.ru/search/?text=%s", url.QueryEscape(fmt.Sprintf("site:t.me \"%s\"", fullFio))), Kind: "link"},
				{Label: "Примечание", Value: "Прямой поиск людей в Telegram закрыт — только через поисковики или @username", Kind: "text"},
			},
			Tags: []string{"Telegram", "ссылки"},
		},
		{
			Title: "Instagram",
			Fields: []models.Field{
				{Label: "Поиск", Value: fmt.Sprintf("https://www.instagram.com/explore/search/keyword/?q=%s", qName), Kind: "link"},
				{Label: "Google-поиск", Value: fmt.Sprintf("https://www.google.com/search?q=%s", igQuery), Kind: "link"},
			},
			Tags: []string{"Instagram", "ссылки"},
		},
		{
			Title: "Steam — профили игроков",
			Fields: []models.Field{
				{Label: "Поиск", Value: fmt.Sprintf("https://steamcommunity.com/search/users/#text=%s", url.QueryEscape(fullFio)), Kind: "link"},
				{Label: "Поиск по имени", Value: fmt.Sprintf("https://www.google.com/search?q=%s", url.QueryEscape(fmt.Sprintf("site:steamcommunity.com \"%s\"", fullFio))), Kind: "link"},
			},
			Tags: []string{"Steam", "ссылки"},
		},
		{
			Title: "Одноклассники",
			Fields: []models.Field{
				{Label: "Поиск", Value: fmt.Sprintf("https://ok.ru/search?query=%s&st.cmd=searchResult&st.mode=People", q), Kind: "link"},
			},
			Tags: []string{"Одноклассники", "ссылки"},
		},
		{
			Title: "Facebook / Meta",
			Fields: []models.Field{
				{Label: "Поиск людей", Value: fmt.Sprintf("https://www.facebook.com/search/people/?q=%s", qName), Kind: "link"},
				{Label: "Примечание", Value: "Требует авторизации для полных результатов", Kind: "text"},
			},
			Tags: []string{"Facebook", "ссылки"},
		},
		{
			Title: "LinkedIn",
			Fields: []models.Field{
				{Label: "Поиск", Value: fmt.Sprintf("https://www.linkedin.com/search/results/people/?keywords=%s", qName), Kind: "link"},
			},
			Tags: []string{"LinkedIn", "ссылки"},
		},
		{
			Title: "TikTok",
			Fields: []models.Field{
				{Label: "Поиск", Value: fmt.Sprintf("https://www.tiktok.com/search/user?q=%s", qName), Kind: "link"},
			},
			Tags: []string{"TikTok", "ссылки"},
		},
		{
			Title: "GitHub — разработчики",
			Fields: []models.Field{
				{Label: "Поиск пользователей", Value: fmt.Sprintf("https://github.com/search?q=%s&type=users", qName), Kind: "link"},
			},
			Tags: []string{"GitHub", "ссылки"},
		},
		{
			Title: "Поиск по никнейму — Sherlock / WhatsMyName",
			Fields: []models.Field{
				{Label: "WhatsMyName Web", Value: fmt.Sprintf("https://whatsmyname.app/?q=%s", url.QueryEscape(firstName+lastName)), Kind: "link"},
				{Label: "Примечание", Value: "Поиск по возможному нику (ФамилияИмя) в 600+ сайтах", Kind: "text"},
			},
			Tags: []string{"OSINT", "username"},
		},
	}
}
