package rnld

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeMultiServer responde como o rnld-bots-websocket em /ws/multi e
// registra os pedidos de controle recebidos.
type fakeMultiServer struct {
	t   *testing.T
	srv *httptest.Server
	url string

	mu       sync.Mutex
	received []controlRequest
	sockets  []*websocket.Conn
	writeMu  sync.Mutex
}

func startMultiServer(t *testing.T) *fakeMultiServer {
	t.Helper()
	f := &fakeMultiServer{t: t}
	upgrader := websocket.Upgrader{}

	f.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "" {
			t.Errorf("expected no x-api-key header, got %q", r.Header.Get("x-api-key"))
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		f.mu.Lock()
		f.sockets = append(f.sockets, conn)
		f.mu.Unlock()

		for {
			var msg controlRequest
			if err := conn.ReadJSON(&msg); err != nil {
				return
			}
			f.mu.Lock()
			f.received = append(f.received, msg)
			f.mu.Unlock()
			for _, key := range msg.APIKeys {
				if strings.HasPrefix(key, "bad") {
					f.send(conn, map[string]any{"op": "error", "api_key_hint": key[:4], "reason": "invalid_key"})
				} else {
					op := "unsubscribed"
					if msg.Op == "subscribe" {
						op = "subscribed"
					}
					f.send(conn, map[string]any{"op": op, "guild_id": "g-" + key})
				}
			}
		}
	}))
	f.url = "ws" + strings.TrimPrefix(f.srv.URL, "http") + "/ws/multi"
	t.Cleanup(func() {
		f.mu.Lock()
		for _, c := range f.sockets {
			_ = c.Close()
		}
		f.mu.Unlock()
		f.srv.Close()
	})
	return f
}

func (f *fakeMultiServer) send(conn *websocket.Conn, v any) {
	f.writeMu.Lock()
	defer f.writeMu.Unlock()
	_ = conn.WriteJSON(v)
}

func (f *fakeMultiServer) snapshot() []controlRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]controlRequest(nil), f.received...)
}

func (f *fakeMultiServer) reset() {
	f.mu.Lock()
	f.received = nil
	f.mu.Unlock()
}

func (f *fakeMultiServer) socket(i int) *websocket.Conn {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.sockets[i]
}

func waitFor(t *testing.T, ch <-chan struct{}, what string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout waiting for %s", what)
	}
}

func eventually(t *testing.T, cond func() bool, what string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("condition not met: %s", what)
}

func connectedSignal(m *MultiEventsClient) <-chan struct{} {
	ch := make(chan struct{}, 10)
	m.OnConnected(func() { ch <- struct{}{} })
	return ch
}

func TestMultiSubscribesOnConnect(t *testing.T) {
	f := startMultiServer(t)
	m := NewMultiEventsClient(f.url)
	t.Cleanup(m.Close)

	var mu sync.Mutex
	var acks []string
	m.OnSubscribed(func(g string) { mu.Lock(); acks = append(acks, g); mu.Unlock() })
	m.Subscribe("k1", "k2")

	connected := connectedSignal(m)
	m.Connect()
	waitFor(t, connected, "connected")

	eventually(t, func() bool { mu.Lock(); defer mu.Unlock(); return len(acks) == 2 }, "2 acks")
	want := []controlRequest{{Op: "subscribe", APIKeys: []string{"k1", "k2"}}}
	if got := f.snapshot(); !reflect.DeepEqual(got, want) {
		t.Errorf("unexpected requests: %+v", got)
	}
}

func TestMultiSendsImmediatelyWhenConnected(t *testing.T) {
	f := startMultiServer(t)
	m := NewMultiEventsClient(f.url)
	t.Cleanup(m.Close)
	connected := connectedSignal(m)
	m.Connect()
	waitFor(t, connected, "connected")

	m.Subscribe("k1")
	m.Unsubscribe("k1")

	want := []controlRequest{
		{Op: "subscribe", APIKeys: []string{"k1"}},
		{Op: "unsubscribe", APIKeys: []string{"k1"}},
	}
	eventually(t, func() bool { return reflect.DeepEqual(f.snapshot(), want) }, "subscribe+unsubscribe")
}

func TestMultiBatchesKeys(t *testing.T) {
	f := startMultiServer(t)
	m := NewMultiEventsClient(f.url)
	t.Cleanup(m.Close)
	connected := connectedSignal(m)
	m.Connect()
	waitFor(t, connected, "connected")

	keys := make([]string, 250)
	for i := range keys {
		keys[i] = fmt.Sprintf("k%d", i)
	}
	m.Subscribe(keys...)

	eventually(t, func() bool {
		got := f.snapshot()
		if len(got) != 3 {
			return false
		}
		return len(got[0].APIKeys) == 100 && len(got[1].APIKeys) == 100 && len(got[2].APIKeys) == 50
	}, "3 batches of 100/100/50")
}

func TestMultiDispatchesEventsAndSubscribeErrors(t *testing.T) {
	f := startMultiServer(t)
	m := NewMultiEventsClient(f.url)
	t.Cleanup(m.Close)

	errs := make(chan SubscribeError, 1)
	m.OnSubscribeError(func(e SubscribeError) { errs <- e })
	bans := make(chan BanEvent, 1)
	m.OnBan(func(ev BanEvent) { bans <- ev })

	connected := connectedSignal(m)
	m.Connect()
	waitFor(t, connected, "connected")

	m.Subscribe("bad-key")
	f.send(f.socket(0), map[string]any{"channel": "rnld_ban_events", "payload": map[string]any{"type": "ban", "guild_id": "g1"}})

	select {
	case ev := <-bans:
		if ev.GuildID != "g1" {
			t.Errorf("unexpected ban: %+v", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for ban")
	}
	select {
	case e := <-errs:
		if e != (SubscribeError{Reason: "invalid_key", APIKeyHint: "bad-"}) {
			t.Errorf("unexpected subscribe error: %+v", e)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for subscribe error")
	}
}

func TestMultiResubscribesAfterReconnect(t *testing.T) {
	f := startMultiServer(t)
	m := NewMultiEventsClient(f.url)
	t.Cleanup(m.Close)
	connected := connectedSignal(m)
	m.Connect()
	waitFor(t, connected, "connected")

	m.Subscribe("k1", "k2")
	m.Unsubscribe("k1")
	eventually(t, func() bool { return len(f.snapshot()) == 2 }, "initial requests")
	f.reset()

	_ = f.socket(0).Close()
	waitFor(t, connected, "reconnected")

	want := []controlRequest{{Op: "subscribe", APIKeys: []string{"k2"}}}
	eventually(t, func() bool { return reflect.DeepEqual(f.snapshot(), want) }, "resubscribe k2 only")
}
