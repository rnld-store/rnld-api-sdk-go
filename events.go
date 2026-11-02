package rnld

import (
	"encoding/json"
	"math/rand/v2"
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
// Reconecta automaticamente com backoff exponencial (1s → 30s, com jitter
// de 50–100% do delay) em caso de queda de conexão.
//
// Handlers são chamados de forma síncrona na goroutine de leitura do
// WebSocket — não bloqueie neles; despache trabalho pesado para sua própria
// goroutine.
type EventsClient struct {
	wsURL string
	// header enviado no handshake (nil = sem autenticação por header).
	header http.Header

	// hooks usados pelo MultiEventsClient.
	// onOpen é chamado após cada conexão (inclusive reconexões), depois dos
	// handlers OnConnected.
	onOpen func(conn *websocket.Conn)
	// control recebe cada mensagem bruta antes do dispatch normal; retorna
	// true se a mensagem foi consumida.
	control func(raw []byte) bool
	// shouldReconnect recebe o erro que encerrou a leitura (um
	// *websocket.CloseError quando o servidor fechou com close code); retornar
	// false encerra o loop de conexão sem reconectar.
	shouldReconnect func(err error) bool

	connMu         sync.Mutex
	conn           *websocket.Conn
	destroyed      bool
	reconnectDelay time.Duration

	writeMu sync.Mutex

	handlersMu       sync.RWMutex
	onBan            []func(BanEvent)
	onTempban        []func(TempBanEvent)
	onUnban          []func(UnbanEvent)
	onUnbanBulk      []func(UnbanBulkEvent)
	onGuildWipe      []func(GuildWipeEvent)
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
	return &EventsClient{
		wsURL:          wsURL,
		header:         http.Header{"x-api-key": []string{apiKey}},
		reconnectDelay: minReconnectDelay,
	}
}

// ── Registro de handlers ─────────────────────────────────────────────────

func addHandler[T any](e *EventsClient, list *[]T, h T) *EventsClient {
	e.handlersMu.Lock()
	*list = append(*list, h)
	e.handlersMu.Unlock()
	return e
}

func (e *EventsClient) OnBan(h func(BanEvent)) *EventsClient { return addHandler(e, &e.onBan, h) }

func (e *EventsClient) OnTempban(h func(TempBanEvent)) *EventsClient {
	return addHandler(e, &e.onTempban, h)
}

func (e *EventsClient) OnUnban(h func(UnbanEvent)) *EventsClient { return addHandler(e, &e.onUnban, h) }

// OnUnbanBulk registra um handler para lotes de desbanimento em massa. Um
// lote pode chegar em vários chunks (mesmo BatchID, Chunk 1..Chunks).
func (e *EventsClient) OnUnbanBulk(h func(UnbanBulkEvent)) *EventsClient {
	return addHandler(e, &e.onUnbanBulk, h)
}

// OnGuildWipe registra um handler para o wipe de uma guild
// (POST /mngr/wipeguild).
func (e *EventsClient) OnGuildWipe(h func(GuildWipeEvent)) *EventsClient {
	return addHandler(e, &e.onGuildWipe, h)
}

func (e *EventsClient) OnNicknameFailed(h func(NicknameFailedEvent)) *EventsClient {
	return addHandler(e, &e.onNicknameFailed, h)
}

func (e *EventsClient) OnUpdateNickname(h func(UpdateNicknameEvent)) *EventsClient {
	return addHandler(e, &e.onUpdateNickname, h)
}

func (e *EventsClient) OnUpdateRoles(h func(UpdateRolesEvent)) *EventsClient {
	return addHandler(e, &e.onUpdateRoles, h)
}

func (e *EventsClient) OnLiberation(h func(LiberationEvent)) *EventsClient {
	return addHandler(e, &e.onLiberation, h)
}

func (e *EventsClient) OnBridgeMessage(h func(BridgeMessageEvent)) *EventsClient {
	return addHandler(e, &e.onBridgeMessage, h)
}

// OnConnected registra um handler chamado quando a conexão WebSocket é
// estabelecida (ou reestabelecida após queda).
func (e *EventsClient) OnConnected(h func()) *EventsClient { return addHandler(e, &e.onConnected, h) }

// OnDisconnected registra um handler chamado quando a conexão cai;
// reconexão automática será tentada.
func (e *EventsClient) OnDisconnected(h func()) *EventsClient {
	return addHandler(e, &e.onDisconnected, h)
}

// OnError registra um handler chamado para erros de conexão/leitura.
func (e *EventsClient) OnError(h func(error)) *EventsClient { return addHandler(e, &e.onError, h) }

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

		conn, _, err := websocket.DefaultDialer.Dial(e.wsURL, e.header)
		if err != nil {
			e.emitError(err)
			e.waitReconnectDelay()
			continue
		}

		e.connMu.Lock()
		if e.destroyed {
			e.connMu.Unlock()
			_ = conn.Close()
			return
		}
		e.conn = conn
		e.reconnectDelay = minReconnectDelay
		e.connMu.Unlock()
		e.emitConnected()
		if e.onOpen != nil {
			e.onOpen(conn)
		}

		readErr := e.readLoop(conn)

		e.connMu.Lock()
		wasDestroyed := e.destroyed
		e.conn = nil
		e.connMu.Unlock()

		if wasDestroyed {
			return
		}
		e.emitDisconnected()
		if e.shouldReconnect != nil && !e.shouldReconnect(readErr) {
			return
		}
		e.waitReconnectDelay()
	}
}

func (e *EventsClient) readLoop(conn *websocket.Conn) error {
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}
		e.dispatch(data)
	}
}

// writeJSON serializa escritas concorrentes no socket (gorilla/websocket
// permite apenas um writer por vez).
func (e *EventsClient) writeJSON(conn *websocket.Conn, v any) error {
	e.writeMu.Lock()
	defer e.writeMu.Unlock()
	return conn.WriteJSON(v)
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

	// jitter (50–100% do delay): evita que muitos clientes reconectem em
	// sincronia quando o servidor reinicia.
	time.Sleep(jitter(delay))
}

func jitter(d time.Duration) time.Duration {
	return time.Duration(float64(d) * (0.5 + rand.Float64()*0.5))
}

// ── Dispatch ─────────────────────────────────────────────────────────────

func decodeAndEmit[T any](e *EventsClient, payload json.RawMessage, hs *[]func(T)) {
	var ev T
	if json.Unmarshal(payload, &ev) == nil {
		emitAll(e, hs, ev)
	}
}

func (e *EventsClient) dispatch(raw []byte) {
	if e.control != nil && e.control(raw) {
		return
	}

	var envelope wsEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return
	}

	if envelope.Channel == channelLiberation {
		decodeAndEmit(e, envelope.Payload, &e.onLiberation)
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
		decodeAndEmit(e, envelope.Payload, &e.onBan)
	case "tempban":
		decodeAndEmit(e, envelope.Payload, &e.onTempban)
	case "unban":
		decodeAndEmit(e, envelope.Payload, &e.onUnban)
	case "unban_bulk":
		decodeAndEmit(e, envelope.Payload, &e.onUnbanBulk)
	case "guild_wipe":
		decodeAndEmit(e, envelope.Payload, &e.onGuildWipe)
	case "nicknameFailed":
		decodeAndEmit(e, envelope.Payload, &e.onNicknameFailed)
	case "updateNickname":
		decodeAndEmit(e, envelope.Payload, &e.onUpdateNickname)
	case "updateRoles":
		decodeAndEmit(e, envelope.Payload, &e.onUpdateRoles)
	case "bridge_message":
		decodeAndEmit(e, envelope.Payload, &e.onBridgeMessage)
	}
}

// ── Emit ─────────────────────────────────────────────────────────────────

func emitAll[T any](e *EventsClient, list *[]func(T), ev T) {
	e.handlersMu.RLock()
	hs := append([]func(T){}, (*list)...)
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

func (e *EventsClient) emitError(err error) { emitAll(e, &e.onError, err) }
