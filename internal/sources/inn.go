package sources

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"osint-ru/internal/models"
)

// INNSource — поиск/проверка ИНН физлица через сервис ФНС
// Официальный открытый сервис: https://service.nalog.ru/inn.do
type INNSource struct {
	client *http.Client
}

func NewINNSource() *INNSource {
	return &INNSource{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

type innResponse struct {
	Code       string `json:"code"`
	Inn        string `json:"inn"`
	Error      string `json:"errorCode"`
	ErrorText  string `json:"errorText"`
}

func (s *INNSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "ФНС — сервис проверки ИНН",
		SourceURL: "https://service.nalog.ru/inn.do",
		Icon:      "credit-card",
		Status:    "pending",
	}

	// Нужны хотя бы фамилия и дата рождения
	if q.LastName == "" || q.BirthDate == "" {
		result.Status = "skip"
		result.Error = "Для поиска ИНН нужны: фамилия и дата рождения"
		return result
	}

	// Если ИНН уже введён — проверяем его
	if q.INN != "" {
		return s.verifyINN(q)
	}

	// Парсим дату рождения DD.MM.YYYY
	parts := strings.Split(q.BirthDate, ".")
	if len(parts) != 3 {
		result.Status = "error"
		result.Error = "Некорректный формат даты рождения (ожидается ДД.ММ.ГГГГ)"
		return result
	}

	// POST-запрос к ФНС сервису поиска ИНН
	apiURL := "https://service.nalog.ru/inn-my/find-by-fio"
	payload := fmt.Sprintf(
		`{"f":"%s","i":"%s","o":"%s","b":"%s.%s.%s"}`,
		q.LastName, q.FirstName, q.MiddleName,
		parts[2], parts[1], parts[0], // YYYY.MM.DD
	)

	req, _ := http.NewRequest("POST", apiURL, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; OSINT-RU/1.0)")
	req.Header.Set("Referer", "https://service.nalog.ru/inn.do")

	resp, err := s.client.Do(req)
	if err != nil {
		// Fallback на публичный сервис
		result.Status = "manual"
		result.Error = "Автоматический запрос не удался"
		result.SearchedURL = "https://service.nalog.ru/inn.do"
		result.Records = []models.Record{
			{
				Title: "Поиск ИНН по ФИО и дате рождения",
				Fields: []models.Field{
					{Label: "Сервис ФНС", Value: "https://service.nalog.ru/inn.do", Kind: "link"},
					{Label: "ФИО", Value: fmt.Sprintf("%s %s %s", q.LastName, q.FirstName, q.MiddleName), Kind: "text"},
					{Label: "Дата рождения", Value: q.BirthDate, Kind: "date"},
				},
				Tags: []string{"ФНС", "ИНН"},
			},
		}
		return result
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var data innResponse
	if err := json.Unmarshal(body, &data); err != nil {
		result.Status = "manual"
		result.SearchedURL = "https://service.nalog.ru/inn.do"
		result.Records = buildManualRecord(
			"Открыть поиск ИНН на ФНС",
			"https://service.nalog.ru/inn.do",
			fmt.Sprintf("Введите: %s %s %s, дата: %s", q.LastName, q.FirstName, q.MiddleName, q.BirthDate),
		)
		return result
	}

	if data.Inn != "" {
		result.Status = "found"
		result.Records = []models.Record{
			{
				Title: "ИНН найден",
				Fields: []models.Field{
					{Label: "ИНН", Value: data.Inn, Kind: "text"},
					{Label: "ФИО", Value: fmt.Sprintf("%s %s %s", q.LastName, q.FirstName, q.MiddleName), Kind: "text"},
					{Label: "Дата рождения", Value: q.BirthDate, Kind: "date"},
				},
				Tags: []string{"ФНС", "ИНН"},
				SourceLink: fmt.Sprintf("https://egrul.nalog.ru/#%s", data.Inn),
			},
		}
	} else {
		result.Status = "not_found"
	}

	return result
}

func (s *INNSource) verifyINN(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "ФНС — проверка ИНН",
		SourceURL: "https://service.nalog.ru/inn.do",
		Icon:      "credit-card",
		Status:    "found",
	}

	// Проверяем контрольное число ИНН (алгоритм ФНС)
	isValid := validateINN(q.INN)

	statusText := "Структура корректна"
	if !isValid {
		statusText = "Некорректный ИНН"
	}

	result.Records = []models.Record{
		{
			Title: fmt.Sprintf("ИНН: %s", q.INN),
			Fields: []models.Field{
				{Label: "ИНН", Value: q.INN, Kind: "text"},
				{Label: "Проверка структуры", Value: statusText, Kind: "badge"},
				{Label: "Тип", Value: innType(q.INN), Kind: "badge"},
				{Label: "Регион (по коду)", Value: innRegion(q.INN), Kind: "text"},
			},
			Tags: []string{"ФНС", "ИНН"},
			SourceLink: fmt.Sprintf("https://egrul.nalog.ru/#%s", q.INN),
		},
	}

	return result
}

func validateINN(inn string) bool {
	if len(inn) != 10 && len(inn) != 12 {
		return false
	}
	digits := make([]int, len(inn))
	for i, c := range inn {
		if c < '0' || c > '9' {
			return false
		}
		digits[i] = int(c - '0')
	}
	if len(inn) == 12 {
		// Физлицо — 12 цифр
		w1 := []int{7, 2, 4, 10, 3, 5, 9, 4, 6, 8}
		w2 := []int{3, 7, 2, 4, 10, 3, 5, 9, 4, 6, 8}
		s1 := 0
		for i, w := range w1 {
			s1 += w * digits[i]
		}
		s2 := 0
		for i, w := range w2 {
			s2 += w * digits[i]
		}
		return digits[10] == (s1%11)%10 && digits[11] == (s2%11)%10
	}
	return true
}

func innType(inn string) string {
	switch len(inn) {
	case 12:
		return "Физическое лицо"
	case 10:
		return "Юридическое лицо / ИП"
	default:
		return "Неизвестно"
	}
}

func innRegion(inn string) string {
	if len(inn) < 2 {
		return "—"
	}
	regions := map[string]string{
		"01": "Республика Адыгея",
		"02": "Республика Башкортостан",
		"03": "Республика Бурятия",
		"04": "Республика Алтай",
		"05": "Республика Дагестан",
		"07": "Кабардино-Балкарская Республика",
		"09": "Карачаево-Черкесская Республика",
		"10": "Республика Карелия",
		"11": "Республика Коми",
		"12": "Республика Марий Эл",
		"13": "Республика Мордовия",
		"14": "Республика Саха (Якутия)",
		"15": "Республика Северная Осетия-Алания",
		"16": "Республика Татарстан",
		"17": "Республика Тыва",
		"18": "Удмуртская Республика",
		"19": "Республика Хакасия",
		"20": "Чеченская Республика",
		"21": "Чувашская Республика",
		"22": "Алтайский край",
		"23": "Краснодарский край",
		"24": "Красноярский край",
		"25": "Приморский край",
		"26": "Ставропольский край",
		"27": "Хабаровский край",
		"28": "Амурская область",
		"29": "Архангельская область",
		"30": "Астраханская область",
		"31": "Белгородская область",
		"32": "Брянская область",
		"33": "Владимирская область",
		"34": "Волгоградская область",
		"35": "Вологодская область",
		"36": "Воронежская область",
		"37": "Ивановская область",
		"38": "Иркутская область",
		"39": "Калининградская область",
		"40": "Калужская область",
		"41": "Камчатский край",
		"42": "Кемеровская область",
		"43": "Кировская область",
		"44": "Костромская область",
		"45": "Курганская область",
		"46": "Курская область",
		"47": "Ленинградская область",
		"48": "Липецкая область",
		"49": "Магаданская область",
		"50": "Московская область",
		"51": "Мурманская область",
		"52": "Нижегородская область",
		"53": "Новгородская область",
		"54": "Новосибирская область",
		"55": "Омская область",
		"56": "Оренбургская область",
		"57": "Орловская область",
		"58": "Пензенская область",
		"59": "Пермский край",
		"60": "Псковская область",
		"61": "Ростовская область",
		"62": "Рязанская область",
		"63": "Самарская область",
		"64": "Саратовская область",
		"65": "Сахалинская область",
		"66": "Свердловская область",
		"67": "Смоленская область",
		"68": "Тамбовская область",
		"69": "Тверская область",
		"70": "Томская область",
		"71": "Тульская область",
		"72": "Тюменская область",
		"73": "Ульяновская область",
		"74": "Челябинская область",
		"75": "Забайкальский край",
		"76": "Ярославская область",
		"77": "г. Москва",
		"78": "г. Санкт-Петербург",
		"79": "Еврейская автономная область",
		"83": "Ненецкий АО",
		"86": "Ханты-Мансийский АО — Югра",
		"87": "Чукотский АО",
		"89": "Ямало-Ненецкий АО",
		"91": "Республика Крым",
		"92": "г. Севастополь",
	}
	code := inn[:2]
	if name, ok := regions[code]; ok {
		return fmt.Sprintf("%s (%s)", name, code)
	}
	return code
}
