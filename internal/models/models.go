// Package models содержит структуры данных, которые используются для обмена
// информацией в запросах и ответах в приложении, а также для представления
// информации о сокращённых URL.
package models

// Request представляет структуру для обработки входящих запросов на создание
// сокращённого URL. Содержит одно поле URL, которое является оригинальной
// ссылкой, которую необходимо сократить.
type Request struct {
	// URL — оригинальный URL, который нужно сократить.
	URL string `json:"url"`
}

// Response представляет структуру для ответа на запрос создания сокращённого URL.
// Содержит поле Result, которое возвращает результат операции.
type Response struct {
	// Result — строка с результатом операции, например, сокращённый URL.
	// Может быть пустым, если результат отсутствует.
	Result string `json:"result,omitempty"`
}

// URLData представляет данные о сокращённом URL. Содержит информацию о сокращённом
// URL, оригинальной ссылке, статусе удаления и идентификаторе пользователя.
// Используется для хранения информации о каждом сокращённом URL.
type URLData struct {
	// UUID — уникальный идентификатор для каждой записи о сокращённом URL.
	UUID string `json:"uuid"`

	// ShortURL — сокращённый URL.
	ShortURL string `json:"short_url"`

	// OriginalURL — оригинальный URL, на который ссылается сокращённый URL.
	OriginalURL string `json:"original_url"`

	// DeletedFlag — флаг, указывающий на то, был ли удалён этот URL.
	// Если значение true, URL был удалён.
	DeletedFlag bool `json:"is_deleted"`

	// UserUUID — уникальный идентификатор пользователя, который создал этот URL.
	UserUUID string `json:"user_uuid"`

	// CorrelationID — идентификатор для отслеживания запросов в системе,
	// используется для связывания запросов и ответов.
	CorrelationID string `json:"correlation_id"`
}

// BatchRequest представляет структуру для пакетных запросов на создание
// сокращённых URL. Включает в себя уникальный идентификатор для отслеживания
// запроса (CorrelationID) и оригинальный URL, который требуется сократить.
type BatchRequest struct {
	// CorrelationID — уникальный идентификатор для отслеживания пакетного запроса.
	CorrelationID string `json:"correlation_id"`

	// OriginalURL — оригинальный URL, который необходимо сократить в пакетном запросе.
	OriginalURL string `json:"original_url"`
}

// BatchResponse представляет структуру для ответа на пакетный запрос,
// возвращая сокращённый URL вместе с CorrelationID для отслеживания.
type BatchResponse struct {
	// CorrelationID — уникальный идентификатор для отслеживания пакетного запроса.
	CorrelationID string `json:"correlation_id"`

	// ShortURL — сокращённый URL, который был создан в результате пакетного запроса.
	ShortURL string `json:"short_url"`
}
