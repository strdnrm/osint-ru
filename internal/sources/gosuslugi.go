package sources

import (
	"fmt"
	"net/url"
	"strings"

	"osint-ru/internal/models"
)

// GovLinksSource — набор публичных ссылок на государственные сервисы
// для ручной проверки данных о человеке
type GovLinksSource struct{}

func NewGovLinksSource() *GovLinksSource {
	return &GovLinksSource{}
}

func (s *GovLinksSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "Прямые ссылки на гос. реестры",
		SourceURL: "https://www.gosuslugi.ru",
		Icon:      "external-link",
		Status:    "manual",
	}

	fio := strings.TrimSpace(q.LastName + " " + q.FirstName + " " + q.MiddleName)
	inn := q.INN

	records := []models.Record{}

	// ФССП — ручной поиск
	fsspURL := fmt.Sprintf(
		"https://fssp.gov.ru/iss/ip/?is[last_name]=%s&is[first_name]=%s&is[patronymic]=%s&is[date]=%s&is[region]=-1&is[iss]=1",
		url.QueryEscape(q.LastName), url.QueryEscape(q.FirstName),
		url.QueryEscape(q.MiddleName), url.QueryEscape(q.BirthDate),
	)
	records = append(records, models.Record{
		Title: "ФССП — Банк данных исполнительных производств",
		Fields: []models.Field{
			{Label: "Поиск по физлицу", Value: fsspURL, Kind: "link"},
			{Label: "Примечание", Value: "Открытый реестр должников по ФЗ 229-ФЗ", Kind: "text"},
		},
		Tags: []string{"ФССП", "прямая ссылка"},
	})

	// ФНС — поиск ИНН
	if q.BirthDate != "" {
		records = append(records, models.Record{
			Title: "ФНС — Узнай ИНН",
			Fields: []models.Field{
				{Label: "Сервис", Value: "https://service.nalog.ru/inn.do", Kind: "link"},
				{Label: "ФИО", Value: fio, Kind: "text"},
				{Label: "Дата рождения", Value: q.BirthDate, Kind: "date"},
			},
			Tags: []string{"ФНС", "ИНН"},
		})
	}

	// ФНС — ЕГРЮЛ/ЕГРИП
	egrulQuery := fio
	if inn != "" {
		egrulQuery = inn
	}
	records = append(records, models.Record{
		Title: "ФНС — ЕГРЮЛ/ЕГРИП (ИП и юрлица)",
		Fields: []models.Field{
			{Label: "Поиск", Value: fmt.Sprintf("https://egrul.nalog.ru/#%s", url.QueryEscape(egrulQuery)), Kind: "link"},
			{Label: "Примечание", Value: "Регистрация ИП и юрлиц на физлицо", Kind: "text"},
		},
		Tags: []string{"ФНС", "ЕГРЮЛ"},
	})

	// ГАС Правосудие
	gaQuery := fmt.Sprintf(`{"query":"%s","type":"OVERALL"}`, fio)
	records = append(records, models.Record{
		Title: "ГАС Правосудие — судебные дела",
		Fields: []models.Field{
			{Label: "Поиск по текстам решений", Value: fmt.Sprintf("https://bsr.sudrf.ru/bigs/portal.html#%s", url.QueryEscape(gaQuery)), Kind: "link"},
			{Label: "Поиск по делам", Value: fmt.Sprintf("https://sudrf.ru/index.php?id=300&searchtype=sp&name=%s", url.QueryEscape(fio)), Kind: "link"},
		},
		Tags: []string{"Правосудие", "суды"},
	})

	// Федресурс
	records = append(records, models.Record{
		Title: "Федресурс — банкротства физлиц",
		Fields: []models.Field{
			{Label: "Поиск", Value: fmt.Sprintf("https://fedresurs.ru/search/persons?searchString=%s", url.QueryEscape(fio)), Kind: "link"},
			{Label: "Примечание", Value: "Сведения о процедурах банкротства физлиц", Kind: "text"},
		},
		Tags: []string{"Федресурс", "банкротство"},
	})

	// Реестр дисквалифицированных лиц ФНС
	records = append(records, models.Record{
		Title: "ФНС — Реестр дисквалифицированных лиц",
		Fields: []models.Field{
			{Label: "Поиск", Value: fmt.Sprintf("https://service.nalog.ru/disqualified.do?fio=%s", url.QueryEscape(fio)), Kind: "link"},
			{Label: "Примечание", Value: "Лица, которым запрещено занимать руководящие должности", Kind: "text"},
		},
		Tags: []string{"ФНС", "дисквалификация"},
	})

	// Реестр недобросовестных поставщиков (ФАС/ЕИС)
	records = append(records, models.Record{
		Title: "ЕИС Закупки — реестр недобросовестных поставщиков",
		Fields: []models.Field{
			{Label: "Поиск", Value: fmt.Sprintf("https://rnp.zakupki.gov.ru/rnp/public/supplier/search?searchString=%s", url.QueryEscape(fio)), Kind: "link"},
			{Label: "Примечание", Value: "Недобросовестные участники госзакупок", Kind: "text"},
		},
		Tags: []string{"ФАС", "закупки"},
	})

	// Реестр террористов Росфинмониторинг
	records = append(records, models.Record{
		Title: "Росфинмониторинг — перечень организаций и физлиц",
		Fields: []models.Field{
			{Label: "Поиск", Value: "https://www.fedsfm.ru/documents/terrorists-catalog-ryb", Kind: "link"},
			{Label: "Примечание", Value: "Публичный список — требует ручного поиска по странице", Kind: "text"},
		},
		Tags: []string{"Росфинмониторинг"},
	})

	// Kartoteka.ru — агрегатор арбитражных дел
	records = append(records, models.Record{
		Title: "Картотека арбитражных дел (КАД)",
		Fields: []models.Field{
			{Label: "Поиск", Value: fmt.Sprintf("https://kad.arbitr.ru/?lastName=%s&firstName=%s&patronymic=%s", url.QueryEscape(q.LastName), url.QueryEscape(q.FirstName), url.QueryEscape(q.MiddleName)), Kind: "link"},
			{Label: "Примечание", Value: "Арбитражные суды России", Kind: "text"},
		},
		Tags: []string{"Арбитраж", "суд"},
	})

	result.Records = records
	return result
}
