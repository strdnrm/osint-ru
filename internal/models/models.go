package models

// SearchQuery — входные данные от пользователя
type SearchQuery struct {
	LastName   string `json:"last_name"`
	FirstName  string `json:"first_name"`
	MiddleName string `json:"middle_name"`
	BirthDate  string `json:"birth_date"` // DD.MM.YYYY
	INN        string `json:"inn"`
	Phone      string `json:"phone"`
	Email      string `json:"email"`
	Region     string `json:"region"`
}

// SourceResult — результат поиска из одного источника
type SourceResult struct {
	Source      string      `json:"source"`
	SourceURL   string      `json:"source_url"`
	Icon        string      `json:"icon"`
	Status      string      `json:"status"` // "found", "not_found", "error", "pending"
	Records     []Record    `json:"records"`
	RawData     interface{} `json:"raw_data,omitempty"`
	Error       string      `json:"error,omitempty"`
	SearchedURL string      `json:"searched_url,omitempty"`
}

// Record — одна запись из результатов
type Record struct {
	Title      string            `json:"title"`
	Fields     []Field           `json:"fields"`
	Tags       []string          `json:"tags,omitempty"`
	SourceLink string            `json:"source_link,omitempty"`
	Extra      map[string]string `json:"extra,omitempty"`
}

// Field — поле внутри записи
type Field struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Kind  string `json:"kind,omitempty"` // "text","badge","link","date","money"
}

// SearchResponse — итоговый ответ API
type SearchResponse struct {
	Query   SearchQuery    `json:"query"`
	Results []SourceResult `json:"results"`
}
