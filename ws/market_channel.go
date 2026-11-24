package ws

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/gorilla/websocket"
)

const marketChannelURL = "wss://ws-subscriptions-clob.polymarket.com/ws/market"

// MarketSubscription задаёт начальную подписку на публичный рыночный канал.
type MarketSubscription struct {
	AssetIDs             []string `json:"assets_ids"`                       // ID токенов исходов рынка.
	InitialDump          *bool    `json:"initial_dump,omitempty"`           // Запросить снимок стакана. По умолчанию true.
	Level                int      `json:"level,omitempty"`                  // Уровень 1–3. При 0 сервер использует 2.
	CustomFeatureEnabled *bool    `json:"custom_feature_enabled,omitempty"` // Включить best_bid_ask, new_market и market_resolved. По умолчанию false.
}

// MarketSubscriptionUpdate изменяет набор токенов на открытом соединении.
type MarketSubscriptionUpdate struct {
	Operation            string   `json:"operation"`                        // subscribe добавляет токены, unsubscribe убирает.
	AssetIDs             []string `json:"assets_ids"`                       // ID токенов для операции.
	Level                int      `json:"level,omitempty"`                  // Уровень 1–3. При 0 поле опускается.
	CustomFeatureEnabled *bool    `json:"custom_feature_enabled,omitempty"` // nil опускает поле, указатель передаёт true или false.
}

// MarketChannelSubscribe открывает соединение через dialer и отправляет начальную подписку sub.
// ctx управляет подключением и отправкой первого сообщения. При ошибке функция возвращает nil и ошибку.
// Вызывающий код читает события и ответы PONG, отправляет текстовый PING каждые 10 секунд и закрывает соединение.
//
//	Проверка sub -> подключение -> отправка подписки -> передача соединения вызывающему коду
func MarketChannelSubscribe(ctx context.Context, dialer *websocket.Dialer, sub MarketSubscription) (*websocket.Conn, error) {
	if err := validateMarketSubscription(sub.AssetIDs, sub.Level); err != nil {
		return nil, err
	}
	return openMarketChannel(ctx, dialer, sub)
}

// openMarketChannel подключается через dialer, отправляет sub и возвращает соединение вызывающему коду.
// При ошибке подключения, записи или отмене ctx во время отправки возвращаются nil и ошибка.
func openMarketChannel(ctx context.Context, dialer *websocket.Dialer, sub MarketSubscription) (*websocket.Conn, error) {
	conn, _, err := dialer.DialContext(ctx, marketChannelURL, nil)
	if err != nil {
		return nil, fmt.Errorf("connect to market channel: %w", err)
	}

	// Отмена прерывает заблокированную запись только до передачи соединения вызывающему коду.
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	err = conn.WriteJSON(struct {
		Type               string `json:"type"` // Имя канала в первом сообщении.
		MarketSubscription        // Параметры начальной подписки.
	}{Type: "market", MarketSubscription: sub})
	if !stop() {
		return nil, ctx.Err()
	}
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("send market subscription: %w", err)
	}
	return conn, nil
}

// MarketChannelSubscribtionUpdate отправляет update по открытому соединению conn.
// nil означает успешную запись. Вызывающий код сериализует запись обновлений и PING в одно соединение.
func MarketChannelSubscribtionUpdate(conn *websocket.Conn, update MarketSubscriptionUpdate) error {
	if conn == nil {
		return errors.New("market channel connection is nil")
	}
	if update.Operation != "subscribe" && update.Operation != "unsubscribe" {
		return fmt.Errorf("invalid market subscription operation %q", update.Operation)
	}
	if err := validateMarketSubscription(update.AssetIDs, update.Level); err != nil {
		return err
	}
	if err := conn.WriteJSON(update); err != nil {
		return fmt.Errorf("send market subscription update: %w", err)
	}
	return nil
}

// validateMarketSubscription проверяет непустые идентификаторы в assetIDs и допустимый уровень level.
func validateMarketSubscription(assetIDs []string, level int) error {
	if len(assetIDs) == 0 || slices.Contains(assetIDs, "") {
		return errors.New("market subscription requires non-empty asset IDs")
	}
	if level < 0 || level > 3 {
		return fmt.Errorf("invalid market subscription level %d", level)
	}
	return nil
}
