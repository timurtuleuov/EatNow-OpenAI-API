package model

// TavilySearchRequest параметры запроса
type TavilySearchRequest struct {
	APIKey            string   `json:"api_key"`
	Query             string   `json:"query"`
	SearchDepth       string   `json:"search_depth,omitempty"` // "basic" или "advanced"
	IncludeAnswer     bool     `json:"include_answer,omitempty"`
	IncludeRawContent bool     `json:"include_raw_content,omitempty"`
	MaxResults        int      `json:"max_results,omitempty"`
	IncludeDomains    []string `json:"include_domains,omitempty"`
}

// TavilySearchResult структура одного результата
type TavilySearchResult struct {
	Title   string  `json:"title"`
	URL     string  `json:"url"`
	Content string  `json:"content"`
	Score   float64 `json:"score"`
}

// TavilyResponse основной ответ API
type TavilyResponse struct {
	Answer  string               `json:"answer,omitempty"`
	Results []TavilySearchResult `json:"results"`
}
