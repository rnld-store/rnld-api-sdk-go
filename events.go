package rnld

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const channelLiberation = "rnld_liberation_events"

const (
	minReconnectDelay = 1 * time.Second
	maxReconnectDelay = 30 * time.Second
)

// wsEnvelope é o formato enviado pelo rnld-bots-websocket.
type wsEnvelope struct {
	Channel string          `json:"channel"`
	Payload json.RawMessage `json:"payload"`
}

// EventsClient conecta via WebSocket ao rnld-bots-websocket e despacha os
// eventos Redis retransmitidos (ban, tempban, unban, liberation, etc).
// Reconecta automaticamente com backoff exponencial (1s → 30s) em caso de
// queda de conexão.
//
// Handlers são chamados de forma síncrona na goroutine de leitura do
// WebSocket — não bloqueie neles; despache trabalho pesado para sua própria
// goroutine.
type EventsClient struct {
	apiKey string
	wsURL  string

	connMu         sync.Mutex
	conn           *websocket.Conn
	destroyed      bool
	reconnectDelay time.Duration

	handlersMu       sync.RWMutex
	onBan            []func(BanEvent)
	onTempban        []func(TempBanEvent)
	onUnban          []func(UnbanEvent)
	onNicknameFailed []func(NicknameFailedEvent)
	onUpdateNickname []func(UpdateNicknameEvent)
	onUpdateRoles    []func(UpdateRolesEvent)
	onLiberation     []func(LiberationEvent)
	onBridgeMessage  []func(BridgeMessageEvent)
	onConnected      []func()
	onDisconnected   []func()
	onError          []func(error)
}

func newEventsClient(apiKey, wsURL string) *EventsClient {
	return &EventsClient{apiKey: apiKey, wsURL: wsURL, reconnectDelay: minReconnectDelay}
}

// ── Registro de handlers ─────────────────────────────────────────────────

func (e *EventsClient) OnBan(h func(BanEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onBan = append(e.onBan, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnTempban(h func(TempBanEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onTempban = append(e.onTempban, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnUnban(h func(UnbanEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onUnban = append(e.onUnban, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnNicknameFailed(h func(NicknameFailedEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onNicknameFailed = append(e.onNicknameFailed, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnUpdateNickname(h func(UpdateNicknameEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onUpdateNickname = append(e.onUpdateNickname, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnUpdateRoles(h func(UpdateRolesEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onUpdateRoles = append(e.onUpdateRoles, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnLiberation(h func(LiberationEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onLiberation = append(e.onLiberation, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnBridgeMessage(h func(BridgeMessageEvent)) *EventsClient {
	e.handlersMu.Lock()
	e.onBridgeMessage = append(e.onBridgeMessage, h)
	e.handlersMu.Unlock()
	return e
}

// OnConnected registra um handler chamado quando a conexão WebSocket é
// estabelecida (ou reestabelecida após queda).
func (e *EventsClient) OnConnected(h func()) *EventsClient {
	e.handlersMu.Lock()
	e.onConnected = append(e.onConnected, h)
	e.handlersMu.Unlock()
	return e
}

// OnDisconnected registra um handler chamado quando a conexão cai;
// reconexão automática será tentada.
func (e *EventsClient) OnDisconnected(h func()) *EventsClient {
	e.handlersMu.Lock()
	e.onDisconnected = append(e.onDisconnected, h)
	e.handlersMu.Unlock()
	return e
}

// OnError registra um handler chamado para erros de conexão/leitura.
func (e *EventsClient) OnError(h func(error)) *EventsClient {
	e.handlersMu.Lock()
	e.onError = append(e.onError, h)
	e.handlersMu.Unlock()
	return e
}

// ── Conexão ──────────────────────────────────────────────────────────────

// Connect inicia a conexão WebSocket em background. Reconecta
// automaticamente em caso de queda.
func (e *EventsClient) Connect() *EventsClient {
	e.connMu.Lock()
	e.destroyed = false
	e.connMu.Unlock()

	go e.connectLoop()
	return e
}

// Close encerra a conexão e cancela reconexões futuras.
func (e *EventsClient) Close() {
	e.connMu.Lock()
	e.destroyed = true
	conn := e.conn
	e.conn = nil
	e.connMu.Unlock()

	if conn != nil {
		_ = conn.Close()
	}
}

func (e *EventsClient) connectLoop() {
	for {
		e.connMu.Lock()
		destroyed := e.destroyed
		e.connMu.Unlock()
		if destroyed {
			return
		}

		conn, _, err := websocket.DefaultDialer.Dial(e.wsURL, http.Header{"x-api-key": []string{e.apiKey}})
		if err != nil {
			e.emitError(err)
			e.waitReconnectDelay()
			continue
		}

		e.connMu.Lock()
		e.conn = conn
		e.reconnectDelay = minReconnectDelay
		e.connMu.Unlock()
		e.emitConnected()

		e.readLoop(conn)

		e.connMu.Lock()
		wasDestroyed := e.destroyed
		e.conn = nil
		e.connMu.Unlock()

		if wasDestroyed {
			return
		}
		e.emitDisconnected()
		e.waitReconnectDelay()
	}
}

func (e *EventsClient) readLoop(conn *websocket.Conn) {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		e.dispatch(data)
	}
}

func (e *EventsClient) waitReconnectDelay() {
	e.connMu.Lock()
	delay := e.reconnectDelay
	next := delay * 2
	if next > maxReconnectDelay {
		next = maxReconnectDelay
	}
	e.reconnectDelay = next
	e.connMu.Unlock()

	time.Sleep(delay)
}

// ── Dispatch ─────────────────────────────────────────────────────────────

func (e *EventsClient) dispatch(raw []byte) {
	var envelope wsEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return
	}

	if envelope.Channel == channelLiberation {
		var ev LiberationEvent
		if err := json.Unmarshal(envelope.Payload, &ev); err == nil {
			e.emitLiberation(ev)
		}
		return
	}

	var typed struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(envelope.Payload, &typed); err != nil || typed.Type == "" {
		return
	}

	switch typed.Type {
	case "ban":
		var ev BanEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitBan(ev)
		}
	case "tempban":
		var ev TempBanEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitTempban(ev)
		}
	case "unban":
		var ev UnbanEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitUnban(ev)
		}
	case "nicknameFailed":
		var ev NicknameFailedEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitNicknameFailed(ev)
		}
	case "updateNickname":
		var ev UpdateNicknameEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitUpdateNickname(ev)
		}
	case "updateRoles":
		var ev UpdateRolesEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitUpdateRoles(ev)
		}
	case "bridge_message":
		var ev BridgeMessageEvent
		if json.Unmarshal(envelope.Payload, &ev) == nil {
			e.emitBridgeMessage(ev)
		}
	}
}

// ── Emit ─────────────────────────────────────────────────────────────────

func (e *EventsClient) emitBan(ev BanEvent) {
	e.handlersMu.RLock()
	hs := append([]func(BanEvent){}, e.onBan...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitTempban(ev TempBanEvent) {
	e.handlersMu.RLock()
	hs := append([]func(TempBanEvent){}, e.onTempban...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitUnban(ev UnbanEvent) {
	e.handlersMu.RLock()
	hs := append([]func(UnbanEvent){}, e.onUnban...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitNicknameFailed(ev NicknameFailedEvent) {
	e.handlersMu.RLock()
	hs := append([]func(NicknameFailedEvent){}, e.onNicknameFailed...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitUpdateNickname(ev UpdateNicknameEvent) {
	e.handlersMu.RLock()
	hs := append([]func(UpdateNicknameEvent){}, e.onUpdateNickname...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitUpdateRoles(ev UpdateRolesEvent) {
	e.handlersMu.RLock()
	hs := append([]func(UpdateRolesEvent){}, e.onUpdateRoles...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitLiberation(ev LiberationEvent) {
	e.handlersMu.RLock()
	hs := append([]func(LiberationEvent){}, e.onLiberation...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitBridgeMessage(ev BridgeMessageEvent) {
	e.handlersMu.RLock()
	hs := append([]func(BridgeMessageEvent){}, e.onBridgeMessage...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(ev)
	}
}

func (e *EventsClient) emitConnected() {
	e.handlersMu.RLock()
	hs := append([]func(){}, e.onConnected...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h()
	}
}

func (e *EventsClient) emitDisconnected() {
	e.handlersMu.RLock()
	hs := append([]func(){}, e.onDisconnected...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h()
	}
}

func (e *EventsClient) emitError(err error) {
	e.handlersMu.RLock()
	hs := append([]func(error){}, e.onError...)
	e.handlersMu.RUnlock()
	for _, h := range hs {
		h(err)
	}
}
