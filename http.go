package rnld

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func httpGet(ctx context.Context, client *http.Client, rawURL, apiKey string, query map[string]string, out any) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return err
	}

	q := parsed.Query()
	for k, v := range query {
		if v == "" {
			continue
		}
		q.Set(k, v)
	}
	parsed.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	return doRequest(client, req, out)
}

func httpPost(ctx context.Context, client *http.Client, rawURL, apiKey string, body any, out any) error {
	req, err := newBodyRequest(ctx, http.MethodPost, rawURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("x-source", "bot-sdk-go")

	return doRequest(client, req, out)
}

func httpDelete(ctx context.Context, client *http.Client, rawURL, apiKey string, body any, out any) error {
	req, err := newBodyRequest(ctx, http.MethodDelete, rawURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	return doRequest(client, req, out)
}

func httpPatch(ctx context.Context, client *http.Client, rawURL, apiKey string, body any, out any) error {
	req, err := newBodyRequest(ctx, http.MethodPatch, rawURL, body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)

	return doRequest(client, req, out)
}

func newBodyRequest(ctx context.Context, method, rawURL string, body any) (*http.Request, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return http.NewRequestWithContext(ctx, method, rawURL, bytes.NewReader(payload))
}

type errorBody struct {
	Mensagem string `json:"mensagem"`
}

func doRequest(client *http.Client, req *http.Request, out any) error {
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return &APIError{StatusCode: res.StatusCode, Message: fmt.Sprintf("HTTP %d: resposta inválida", res.StatusCode)}
	}

	var data json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &data); err != nil {
			return &APIError{StatusCode: res.StatusCode, Message: fmt.Sprintf("HTTP %d: resposta inválida", res.StatusCode)}
		}
	}

	var eb errorBody
	_ = json.Unmarshal(data, &eb)

	switch {
	case res.StatusCode == http.StatusUnauthorized || res.StatusCode == http.StatusForbidden:
		return newUnauthorizedError(eb.Mensagem)
	case res.StatusCode == http.StatusNotFound:
		return newNotFoundError(eb.Mensagem)
	case res.StatusCode < 200 || res.StatusCode >= 300:
		msg := eb.Mensagem
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", res.StatusCode)
		}
		var body any
		_ = json.Unmarshal(data, &body)
		return &APIError{StatusCode: res.StatusCode, Message: msg, Body: body}
	}

	if out == nil || len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, out)
}
