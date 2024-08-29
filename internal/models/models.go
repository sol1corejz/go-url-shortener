package models

type Request struct {
	URL string `json:"url"`
}

type Response struct {
	Result string `json:"result,omitempty"`
}

type URLData struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
