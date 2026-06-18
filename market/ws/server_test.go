package ws

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func TestHeartbeatIntervalFromEnv(t *testing.T) {
	t.Setenv("WS_HEARTBEAT_INTERVAL_SECS", "")
	if got := heartbeatIntervalFromEnv(); got != 30*time.Second {
		t.Fatalf("default heartbeat interval = %s, want 30s", got)
	}

	t.Setenv("WS_HEARTBEAT_INTERVAL_SECS", "7")
	if got := heartbeatIntervalFromEnv(); got != 7*time.Second {
		t.Fatalf("configured heartbeat interval = %s, want 7s", got)
	}

	t.Setenv("WS_HEARTBEAT_INTERVAL_SECS", "bad")
	if got := heartbeatIntervalFromEnv(); got != 30*time.Second {
		t.Fatalf("invalid heartbeat interval = %s, want default 30s", got)
	}
}

func TestStaleWebSocketConnectionIsClosed(t *testing.T) {
	hub := NewHub(zap.NewNop())
	go hub.Run()

	server := NewServer(hub, nil, zap.NewNop(), 0)
	server.heartbeatInterval = 20 * time.Millisecond

	httpServer := httptest.NewServer(http.HandlerFunc(server.handleWebSocket))
	defer httpServer.Close()

	wsURL := "ws" + strings.TrimPrefix(httpServer.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	defer conn.Close()

	conn.SetPingHandler(func(string) error {
		return nil
	})
	conn.SetReadDeadline(time.Now().Add(750 * time.Millisecond))

	_, _, err = conn.ReadMessage()
	if err == nil {
		t.Fatal("expected stale connection to be closed")
	}
}
