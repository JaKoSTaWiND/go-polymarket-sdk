package transport_http

import (
	"net/http"
	"net/url"
)

// HttpClient хранит HTTP-клиент и базовый адрес API.
type HttpClient struct {
	HttpClient *http.Client // Выполняет исходящие запросы.
	BaseURL    *url.URL     // Базовый адрес для построения URL запросов.
}

// CreateHttpClient создаёт клиент для API по адресу baseURL.
// Для отправки запросов используется переданный client.
// Если baseURL не удаётся разобрать, функция возвращает ошибку.
func CreateHttpClient(client *http.Client, baseURL string) (*HttpClient, error) {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	return &HttpClient{
		HttpClient: client,
		BaseURL:    parsedURL,
	}, nil
}
