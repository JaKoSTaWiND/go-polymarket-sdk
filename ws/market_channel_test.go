package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestMarketChannelSubscriptions(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws/market" {
			http.NotFound(w, r)
			return
		}
		var upgrader websocket.Upgrader
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Error(err)
			return
		}
		defer conn.Close()
		for {
			kind, frame, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if kind != websocket.TextMessage {
				t.Errorf("received frame type %d, want text", kind)
				return
			}
			if err := conn.WriteMessage(kind, frame); err != nil {
				t.Error(err)
				return
			}
		}
	}))
	defer server.Close()
	dialer := &websocket.Dialer{
		TLSClientConfig: server.Client().Transport.(*http.Transport).TLSClientConfig.Clone(),
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			if addr != "ws-subscriptions-clob.polymarket.com:443" {
				return nil, fmt.Errorf("unexpected WS address %s", addr)
			}
			var d net.Dialer
			return d.DialContext(ctx, network, server.Listener.Addr().String())
		},
	}
	dialer.TLSClientConfig.ServerName = "example.com" // Имя в сертификате httptest.

	checkFrame := func(conn *websocket.Conn, expected string) {
		t.Helper()
		if err := conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
		var got, want map[string]any
		if err := conn.ReadJSON(&got); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(expected), &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(want, got) {
			t.Fatalf("frame = %v, want %v", got, want)
		}
	}

	id := []string{"123", "456"}
	conn, err := MarketChannelSubscribe(t.Context(), dialer, MarketSubscription{AssetIDs: id})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	checkFrame(conn, `{"type":"market","assets_ids":["123","456"]}`)

	on, off := true, false
	update := MarketSubscriptionUpdate{Operation: "subscribe", AssetIDs: []string{"789"}, Level: 3, CustomFeatureEnabled: &off}
	if err := MarketChannelSubscribtionUpdate(conn, update); err != nil {
		t.Fatal(err)
	}
	checkFrame(conn, `{"operation":"subscribe","assets_ids":["789"],"level":3,"custom_feature_enabled":false}`)
	if err := MarketChannelSubscribtionUpdate(conn, MarketSubscriptionUpdate{Operation: "unsubscribe", AssetIDs: id}); err != nil {
		t.Fatal(err)
	}
	checkFrame(conn, `{"operation":"unsubscribe","assets_ids":["123","456"]}`)

	for _, invalid := range []MarketSubscriptionUpdate{
		{Operation: "replace", AssetIDs: id},
		{Operation: "subscribe", AssetIDs: id, Level: 4},
		{Operation: "unsubscribe"},
	} {
		if err := MarketChannelSubscribtionUpdate(conn, invalid); err == nil {
			t.Fatalf("accepted invalid update %+v", invalid)
		}
	}

	conn2, err := MarketChannelSubscribe(t.Context(), dialer, MarketSubscription{
		AssetIDs: id, InitialDump: &off, Level: 1, CustomFeatureEnabled: &on,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer conn2.Close()
	checkFrame(conn2, `{"type":"market","assets_ids":["123","456"],"initial_dump":false,"level":1,"custom_feature_enabled":true}`)
}
