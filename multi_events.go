package rnld

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// maxKeysPerMessage é o mesmo limite por mensagem que o rnld-bots-websocket
// aceita em /ws/multi.
const maxKeysPerMessage = 100

const defaultMultiWSURL = defaultWSURL + "/multi"

// controlReply é a resposta de controle do servidor (tem `op` em vez de
// `channel`).
type controlReply struct {
	Op         string `json:"op"`
	GuildID    string `json:"guild_id"`
	APIKeyHint string `json:"api_key_hint"`
	Reason     string `json:"reason"`
}

type controlRequest struct {
	Op      string   `json:"op"`
	APIKeys []string `json:"api_keys"`
}

// SubscribeError descreve uma assinatura recusada pelo servidor.
type SubscribeError struct {
	// Reason é o motivo da recusa (ex.: "invalid_key", "limit_exceeded").
	Reason string
	// APIKeyHint são os primeiros caracteres da api_key recusada, quando
	// informados pelo servidor.
	APIKeyHint string
	// GuildID da guild, quando informado pelo servidor.
	GuildID string
}

// MultiEventsClient mantém uma única conexão WebSocket (/ws/multi) para
// várias guilds.
//
// Em vez de uma conexão por api_key, a conexão assina guilds por mensagem,
// provando cada uma com a respectiva api_key. Os eventos são os mesmos do
// EventsClient (os handlers OnBan, OnUnban etc. estão disponíveis via
// embedding) e já trazem guild_id no payload. Ao reconectar, todas as keys
// assinadas são reenviadas automaticamente.
//
// Eventos extras: OnSubscribed, OnUnsubscribed e OnSubscribeError.
type MultiEventsClient struct {
	*EventsClient

	keysMu sync.Mutex
	keys   []string // ordem de inserção
	keySet map[string]struct{}
	// active é a conexão que já recebeu o subscribe inicial; Subscribe e
	// Unsubscribe só enviam direto por ela.
	active *websocket.Conn

	handlersMu       sync.RWMutex
	onSubscribed     []func(guildID string)
	onUnsubscribed   []func(guildID string)
	onSubscribeError []func(SubscribeError)
}

// NewMultiEventsClient cria um MultiEventsClient. Se wsURL for vazio, usa
// wss://events.rnld.dev/ws/multi. A autenticação é por mensagem (Subscribe),
// não por header.
func NewMultiEventsClient(wsURL string) *MultiEventsClient {
	if wsURL == "" {
		wsURL = defaultMultiWSURL
	}
	m := &MultiEventsClient{
		EventsClient: &EventsClient{wsURL: wsURL, reconnectDelay: minReconnectDelay},
		keySet:       map[string]struct{}{},
	}
	m.EventsClient.onOpen = m.handleOpen
	m.EventsClient.control = m.handleControl
	m.EventsClient.OnDisconnected(m.clearActive)
	return m
}

// Subscribe assina as guilds dessas api_keys. Se já conectado envia agora;
// senão, ao conectar.
func (m *MultiEventsClient) Subscribe(apiKeys ...string) *MultiEventsClient {
	m.keysMu.Lock()
	defer m.keysMu.Unlock()
	for _, k := range apiKeys {
		if _, ok := m.keySet[k]; !ok {
			m.keySet[k] = struct{}{}
			m.keys = append(m.keys, k)
		}
	}
	m.send(m.active, "subscribe", apiKeys)
	return m
}

// Unsubscribe deixa de receber eventos das guilds dessas api_keys.
func (m *MultiEventsClient) Unsubscribe(apiKeys ...string) *MultiEventsClient {
	m.keysMu.Lock()
	defer m.keysMu.Unlock()
	remove := map[string]struct{}{}
	for _, k := range apiKeys {
		remove[k] = struct{}{}
		delete(m.keySet, k)
	}
	kept := m.keys[:0]
	for _, k := range m.keys {
		if _, ok := remove[k]; !ok {
			kept = append(kept, k)
		}
	}
	m.keys = kept
	m.send(m.active, "unsubscribe", apiKeys)
	return m
}

// Connect inicia a conexão WebSocket em background.
func (m *MultiEventsClient) Connect() *MultiEventsClient {
	m.EventsClient.Connect()
	return m
}

// OnSubscribed registra um handler chamado quando o servidor confirma a
// assinatura de uma guild.
func (m *MultiEventsClient) OnSubscribed(h func(guildID string)) *MultiEventsClient {
	m.handlersMu.Lock()
	m.onSubscribed = append(m.onSubscribed, h)
	m.handlersMu.Unlock()
	return m
}

// OnUnsubscribed registra um handler chamado quando o servidor confirma a
// remoção da assinatura de uma guild.
func (m *MultiEventsClient) OnUnsubscribed(h func(guildID string)) *MultiEventsClient {
	m.handlersMu.Lock()
	m.onUnsubscribed = append(m.onUnsubscribed, h)
	m.handlersMu.Unlock()
	return m
}

// OnSubscribeError registra um handler chamado quando o servidor recusa uma
// assinatura (ex.: invalid_key, limit_exceeded).
func (m *MultiEventsClient) OnSubscribeError(h func(SubscribeError)) *MultiEventsClient {
	m.handlersMu.Lock()
	m.onSubscribeError = append(m.onSubscribeError, h)
	m.handlersMu.Unlock()
	return m
}

func (m *MultiEventsClient) handleOpen(conn *websocket.Conn) {
	m.keysMu.Lock()
	defer m.keysMu.Unlock()
	m.active = conn
	m.send(conn, "subscribe", m.keys)
}

func (m *MultiEventsClient) clearActive() {
	m.keysMu.Lock()
	m.active = nil
	m.keysMu.Unlock()
}

func (m *MultiEventsClient) handleControl(raw []byte) bool {
	var reply controlReply
	if err := json.Unmarshal(raw, &reply); err != nil || reply.Op == "" {
		return false
	}

	m.handlersMu.RLock()
	subscribed := append([]func(string){}, m.onSubscribed...)
	unsubscribed := append([]func(string){}, m.onUnsubscribed...)
	subErrors := append([]func(SubscribeError){}, m.onSubscribeError...)
	m.handlersMu.RUnlock()

	switch reply.Op {
	case "subscribed":
		for _, h := range subscribed {
			h(reply.GuildID)
		}
	case "unsubscribed":
		for _, h := range unsubscribed {
			h(reply.GuildID)
		}
	case "error":
		ev := SubscribeError{Reason: reply.Reason, APIKeyHint: reply.APIKeyHint, GuildID: reply.GuildID}
		for _, h := range subErrors {
			h(ev)
		}
	}
	return true
}

// send envia o pedido em lotes de maxKeysPerMessage. Deve ser chamado com
// keysMu travado.
func (m *MultiEventsClient) send(conn *websocket.Conn, op string, apiKeys []string) {
	if conn == nil {
		return
	}
	for i := 0; i < len(apiKeys); i += maxKeysPerMessage {
		end := min(i+maxKeysPerMessage, len(apiKeys))
		batch := append([]string(nil), apiKeys[i:end]...)
		// falha de escrita = conexão caindo; o readLoop detecta e a
		// reconexão reenvia todas as keys.
		if err := m.writeJSON(conn, controlRequest{Op: op, APIKeys: batch}); err != nil {
			return
		}
	}
}
