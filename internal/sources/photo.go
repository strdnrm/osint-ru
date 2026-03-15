package sources

import (
	"encoding/base64"
	"fmt"
	"net/url"

	"osint-ru/internal/models"
)

// PhotoSource — reverse image search через публичные поисковики
// Yandex, Google, Bing, TinEye — все принимают URL изображения
// Для загруженного файла — используем data URI (Yandex поддерживает через форму)
type PhotoSource struct{}

func NewPhotoSource() *PhotoSource {
	return &PhotoSource{}
}

func (s *PhotoSource) Search(q models.SearchQuery) models.SourceResult {
	result := models.SourceResult{
		Source:    "Поиск по фото — обратный поиск изображения",
		SourceURL: "https://yandex.ru/images/",
		Icon:      "camera",
		Status:    "manual",
	}

	// Нет ни URL, ни base64 фото
	if q.PhotoURL == "" && q.PhotoBase64 == "" {
		result.Status = "skip"
		result.Error = "Фото не загружено — пропускаем поиск по изображению"
		return result
	}

	imageURL := q.PhotoURL

	// Если загружен файл (base64) — используем Yandex через прямую ссылку
	// (base64 передаём через data URI — только Yandex CBiR поддерживает напрямую)
	var yandexURL, googleURL, bingURL, tinyeyeURL string

	if imageURL != "" {
		// Есть внешний URL изображения — все движки принимают
		enc := url.QueryEscape(imageURL)
		yandexURL  = fmt.Sprintf("https://yandex.ru/images/search?url=%s&rpt=imageview", enc)
		googleURL  = fmt.Sprintf("https://lens.google.com/uploadbyurl?url=%s", enc)
		bingURL    = fmt.Sprintf("https://www.bing.com/images/search?view=detailv2&iss=sbi&q=imgurl:%s", enc)
		tinyeyeURL = fmt.Sprintf("https://tineye.com/search/?url=%s", enc)
	} else {
		// Есть base64 — Yandex через форму загрузки (ссылка на страницу)
		// Google Lens — аналогично
		yandexURL  = "https://yandex.ru/images/" // откроется форма, файл загружен отдельно
		googleURL  = "https://lens.google.com/"
		bingURL    = "https://www.bing.com/visualsearch"
		tinyeyeURL = "https://tineye.com/"
	}

	records := []models.Record{
		{
			Title: "Яндекс Картинки — обратный поиск",
			Fields: []models.Field{
				{Label: "Ссылка", Value: yandexURL, Kind: "link"},
				{Label: "Примечание", Value: "Лучший вариант для поиска по лицу в Рунете. Находит соцсети, новости, форумы.", Kind: "text"},
			},
			Tags:       []string{"Яндекс", "reverse image"},
			SourceLink: yandexURL,
		},
		{
			Title: "Google Lens — поиск по изображению",
			Fields: []models.Field{
				{Label: "Ссылка", Value: googleURL, Kind: "link"},
				{Label: "Примечание", Value: "Google Lens — находит похожие фото по всему миру.", Kind: "text"},
			},
			Tags:       []string{"Google", "reverse image"},
			SourceLink: googleURL,
		},
		{
			Title: "Bing Visual Search",
			Fields: []models.Field{
				{Label: "Ссылка", Value: bingURL, Kind: "link"},
				{Label: "Примечание", Value: "Microsoft Bing — поиск по изображению, хорошо работает с соцсетями.", Kind: "text"},
			},
			Tags:       []string{"Bing", "reverse image"},
			SourceLink: bingURL,
		},
		{
			Title: "TinEye — точное совпадение",
			Fields: []models.Field{
				{Label: "Ссылка", Value: tinyeyeURL, Kind: "link"},
				{Label: "Примечание", Value: "TinEye ищет точные и изменённые копии фото. Полезно для проверки подлинности.", Kind: "text"},
			},
			Tags:       []string{"TinEye", "reverse image"},
			SourceLink: tinyeyeURL,
		},
	}

	// Если есть base64 — добавляем инструкцию
	if q.PhotoBase64 != "" && q.PhotoURL == "" {
		// Определяем размер для информации
		decoded, _ := base64.StdEncoding.DecodeString(q.PhotoBase64)
		sizeKB := len(decoded) / 1024
		records = append([]models.Record{{
			Title: "Инструкция по загрузке",
			Fields: []models.Field{
				{Label: "Размер файла", Value: fmt.Sprintf("%d КБ", sizeKB), Kind: "text"},
				{Label: "Способ", Value: "Нажми на значок камеры в строке поиска Яндекс / Google и загрузи файл вручную", Kind: "text"},
				{Label: "Совет", Value: "Яндекс.Картинки лучше всего ищет по лицам в российском сегменте интернета", Kind: "text"},
			},
			Tags: []string{"инструкция"},
		}}, records...)
	}

	result.Status = "manual"
	result.Records = records
	return result
}
