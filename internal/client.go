package internal

import (
	"net/http"
)

// HttpClient хранит HTTP-клиент и базовый адрес API.
type Client struct {
	HttpClient *http.Client // Выполняет исходящие запросы.
}

func ClientProvider() *Client {
	return &Client{
		HttpClient: &http.Client{
		},
	}
}