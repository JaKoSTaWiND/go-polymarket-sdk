package internal

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// Client объединяет HTTP-клиент и WebSocket-клиент SDK.
type Client struct {
	HttpClient *http.Client      // Выполняет исходящие HTTP-запросы.
	WSClient   *websocket.Dialer // Открывает WebSocket-соединения по указанному URL.
}

// ClientProvider возвращает новый Client со стандартными настройками HTTP и WebSocket.
func ClientProvider() *Client {
	dialer := *websocket.DefaultDialer
	return &Client{
		HttpClient: &http.Client{},
		WSClient:   &dialer,
	}
}
