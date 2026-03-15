package sources

import (
	"fmt"
	"net/url"
	"strings"

	"osint-ru/internal/models"
)

// RosreestrSource — ссылки для поиска в Росреестре
// Открытые публичные сведения о правах на недвижимость (ограниченный публичный доступ)
type RosreestrSource struct{}

func NewRosreestrSource() *RosreestrSource {
	return &RosreestrSource{}
}

func (s *RosreestrSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "Росреестр — сведения о недвижимости",
		SourceURL: "https://rosreestr.gov.ru",
		Icon:      "home",
		Status:    "manual",
	}

	fio := strings.TrimSpace(q.LastName + " " + q.FirstName + " " + q.MiddleName)

	result.SearchedURL = "https://rosreestr.gov.ru/eservices/search-objects/"
	result.Error = "Росреестр требует регистрации для запросов; ниже — прямые ссылки на публичные сервисы"

	records := []models.Record{
		{
			Title: "Публичная кадастровая карта",
			Fields: []models.Field{
				{Label: "Сервис", Value: "pkk.rosreestr.gov.ru — просмотр объектов на карте", Kind: "text"},
				{Label: "Ссылка", Value: "https://pkk.rosreestr.gov.ru/", Kind: "link"},
				{Label: "Примечание", Value: "Поиск по кадастровому номеру или адресу", Kind: "text"},
			},
			Tags: []string{"Росреестр", "кадастр"},
		},
		{
			Title: "Запрос выписки из ЕГРН",
			Fields: []models.Field{
				{Label: "Сервис", Value: "Запрос сведений из ЕГРН через Госуслуги", Kind: "text"},
				{Label: "Ссылка", Value: "https://www.gosuslugi.ru/600326/1/form", Kind: "link"},
				{Label: "Примечание", Value: fmt.Sprintf("Выписка о правах гражданина: %s", fio), Kind: "text"},
			},
			Tags: []string{"Росреестр", "ЕГРН"},
		},
		{
			Title: "Справочная информация об объектах",
			Fields: []models.Field{
				{Label: "Ссылка", Value: fmt.Sprintf("https://rosreestr.gov.ru/eservices/search-objects/?objectType=1&query=%s", url.QueryEscape(fio)), Kind: "link"},
			},
			Tags: []string{"Росреестр"},
		},
	}

	result.Records = records
	return result
}
