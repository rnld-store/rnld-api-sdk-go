// Package rnld é o SDK oficial em Go para integração com a RNLD API —
// whitelist e query de jogadores de servidores FiveM, mais consumo de
// eventos em tempo real via WebSocket.
package rnld

import "net/http"

const (
	defaultBaseURL = "https://api.rnld.dev"
	defaultWSURL   = "wss://events.rnld.dev/ws"
)

// ClientOptions configura um Client.
type ClientOptions struct {
	// APIKey do servidor (header x-api-key).
	APIKey string
	// GuildID da guild Discord do servidor.
	GuildID string
	// BaseURL sobrescreve a URL base da API REST (default https://api.rnld.dev).
	BaseURL string
	// WSURL sobrescreve a URL do rnld-bots-websocket (default
	// wss://events.rnld.dev/ws). Necessário para Client.Events.
	WSURL string
	// HTTPClient customizado. Se nil, usa http.DefaultClient.
	HTTPClient *http.Client
}

// Client é o entry point do SDK: agrupa o cliente HTTP de whitelist e o
// cliente de eventos WebSocket.
type Client struct {
	Whitelist *WhitelistClient
	Events    *EventsClient
}

// NewClient cria um novo Client a partir de ClientOptions.
func NewClient(opts ClientOptions) *Client {
	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	wsURL := opts.WSURL
	if wsURL == "" {
		wsURL = defaultWSURL
	}

	httpClient := opts.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &Client{
		Whitelist: newWhitelistClient(opts.APIKey, opts.GuildID, baseURL, httpClient),
		Events:    newEventsClient(opts.APIKey, wsURL),
	}
}
