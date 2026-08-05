package rnld

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

const testAPIKey = "test-key"
const testGuildID = "123456789"

type capturedRequest struct {
	Method string
	URL    string
	Query  url.Values
	Body   map[string]any
}

func startServer(t *testing.T, status int, respBody any) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured.Method = r.Method
		captured.URL = r.URL.Path
		captured.Query = r.URL.Query()

		if r.Method != http.MethodGet && r.Body != nil {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			captured.Body = body
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(respBody)
	}))
	t.Cleanup(srv.Close)
	return srv, captured
}

func newTestClient(baseURL string) *WhitelistClient {
	return newWhitelistClient(testAPIKey, testGuildID, baseURL, http.DefaultClient)
}

func TestWhitelistQuery(t *testing.T) {
	t.Run("retorna dados do jogador em caso de sucesso", func(t *testing.T) {
		srv, _ := startServer(t, 200, map[string]any{
			"isStaff": false, "isBanned": false, "banReason": "N/A", "banExpiresAt": nil,
			"banHistory": []any{}, "last_ip": "127.0.0.1",
			"identities": map[string]any{
				"guild_id": testGuildID, "discord_id": "111", "steamHex": "", "license": "",
				"license2": "", "fivem": "", "live": "", "whitelisted": 1, "id": "WPLP", "wl_id": "WPLP",
			},
			"code": 200, "mensagem": "OK",
		})
		client := newTestClient(srv.URL)

		result, err := client.Query(context.Background(), QueryWhitelistParams{DiscordID: "111"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		if result.Identities.WlID != "WPLP" {
			t.Errorf("expected wl_id WPLP, got %q", result.Identities.WlID)
		}
	})

	t.Run("retorna nil quando jogador nao encontrado", func(t *testing.T) {
		srv, _ := startServer(t, 404, map[string]any{"mensagem": "Não encontrado"})
		client := newTestClient(srv.URL)

		result, err := client.Query(context.Background(), QueryWhitelistParams{DiscordID: "999"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result, got %+v", result)
		}
	})

	t.Run("retorna UnauthorizedError em 403", func(t *testing.T) {
		srv, _ := startServer(t, 403, map[string]any{"mensagem": "Acesso negado"})
		client := newTestClient(srv.URL)

		_, err := client.Query(context.Background(), QueryWhitelistParams{DiscordID: "111"})
		var unauthorized *UnauthorizedError
		if !errors.As(err, &unauthorized) {
			t.Fatalf("expected UnauthorizedError, got %v", err)
		}
	})
}

func TestWhitelistLiberar(t *testing.T) {
	srv, _ := startServer(t, 200, map[string]any{"status": true, "mensagem": "Whitelist liberada."})
	client := newTestClient(srv.URL)

	result, err := client.Liberar(context.Background(), LiberarParams{WlID: "WPLP"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Status {
		t.Errorf("expected status true")
	}
}

func TestWhitelistDivergencia(t *testing.T) {
	t.Run("retorna dados e envia GET com query params", func(t *testing.T) {
		srv, captured := startServer(t, 200, map[string]any{
			"status": true,
			"data": map[string]any{
				"divergenceId": "DIV-WPLP-25749", "player_id": "WPLP", "discord_id": "148073845960605698",
				"steamhex": "x", "license": "y", "license2": "z", "resolved": 0,
				"divergence_code": "DIV-WPLP-25749",
			},
			"mensagem": "Divergência encontrada com sucesso.", "code": 200,
		})
		client := newTestClient(srv.URL)

		result, err := client.Divergencia(context.Background(), DivergenciaParams{Code: "DIV-WPLP-25749"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Data.DivergenceID != "DIV-WPLP-25749" {
			t.Errorf("unexpected divergenceId: %q", result.Data.DivergenceID)
		}

		if captured.URL != "/whitelists/divergence" {
			t.Errorf("unexpected path: %q", captured.URL)
		}
		if captured.Query.Get("guild_id") != testGuildID {
			t.Errorf("unexpected guild_id: %q", captured.Query.Get("guild_id"))
		}
		if captured.Query.Get("code") != "DIV-WPLP-25749" {
			t.Errorf("unexpected code: %q", captured.Query.Get("code"))
		}
		if captured.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", captured.Method)
		}
	})

	t.Run("retorna nil quando nao encontrada", func(t *testing.T) {
		srv, _ := startServer(t, 404, map[string]any{"mensagem": "Não encontrado"})
		client := newTestClient(srv.URL)

		result, err := client.Divergencia(context.Background(), DivergenciaParams{Code: "X"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result != nil {
			t.Errorf("expected nil result")
		}
	})
}

func TestWhitelistDivergenciaFix(t *testing.T) {
	srv, captured := startServer(t, 200, map[string]any{
		"status": true, "mensagem": "ok", "code": 200,
		"data": map[string]any{
			"divergenceId": "DIV-WPLP-25749", "player_id": "WPLP",
			"identities_updated": map[string]any{
				"discord_id": "148073845960605698", "steamhex": "x", "license": "y", "license2": "z",
			},
		},
	})
	client := newTestClient(srv.URL)

	result, err := client.DivergenciaFix(context.Background(), DivergenciaFixParams{DivergenceID: "DIV-WPLP-25749"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Data.IdentitiesUpdated.DiscordID != "148073845960605698" {
		t.Errorf("unexpected discord_id: %q", result.Data.IdentitiesUpdated.DiscordID)
	}

	if captured.URL != "/whitelists/divergencefix" || captured.Method != http.MethodPost {
		t.Errorf("unexpected request: %s %s", captured.Method, captured.URL)
	}
	if captured.Body["divergenceId"] != "DIV-WPLP-25749" {
		t.Errorf("unexpected body: %+v", captured.Body)
	}
}

func TestWhitelistDeletarToken(t *testing.T) {
	t.Run("inclui actorDiscordId quando informado", func(t *testing.T) {
		srv, captured := startServer(t, 200, map[string]any{"status": true, "mensagem": "ok"})
		client := newTestClient(srv.URL)

		_, err := client.DeletarToken(context.Background(), DeletarTokenParams{WlID: "WPLP", ActorDiscordID: "148073845960605698"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.Body["actorDiscordId"] != "148073845960605698" {
			t.Errorf("unexpected body: %+v", captured.Body)
		}
	})

	t.Run("omite actorDiscordId quando nao informado", func(t *testing.T) {
		srv, captured := startServer(t, 200, map[string]any{"status": true, "mensagem": "ok"})
		client := newTestClient(srv.URL)

		_, err := client.DeletarToken(context.Background(), DeletarTokenParams{WlID: "WPLP"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := captured.Body["actorDiscordId"]; ok {
			t.Errorf("expected actorDiscordId to be omitted, got %+v", captured.Body)
		}
	})
}

func TestWhitelistStaffDestaff(t *testing.T) {
	t.Run("staff envia POST com discordId", func(t *testing.T) {
		srv, captured := startServer(t, 200, map[string]any{"status": true, "mensagem": "ok"})
		client := newTestClient(srv.URL)

		_, err := client.Staff(context.Background(), StaffParams{WlID: "WPLP", DiscordID: "148073845960605698"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.Method != http.MethodPost || captured.URL != "/whitelists/staff" {
			t.Errorf("unexpected request: %s %s", captured.Method, captured.URL)
		}
		if captured.Body["discordId"] != "148073845960605698" {
			t.Errorf("unexpected body: %+v", captured.Body)
		}
	})

	t.Run("destaff envia DELETE sem discordId", func(t *testing.T) {
		srv, captured := startServer(t, 200, map[string]any{"status": true, "mensagem": "ok"})
		client := newTestClient(srv.URL)

		_, err := client.Destaff(context.Background(), DestaffParams{WlID: "WPLP"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", captured.Method)
		}
		if _, ok := captured.Body["discordId"]; ok {
			t.Errorf("expected discordId to be absent, got %+v", captured.Body)
		}
	})
}

func TestWhitelistTempban(t *testing.T) {
	fixture := map[string]any{
		"status": true, "code": 200, "topic": "tempban", "token": "WPLP",
		"expires_at": "2026-05-06T02:17:21.043Z", "mensagem": "banido",
	}

	t.Run("envia hours e reason", func(t *testing.T) {
		srv, captured := startServer(t, 200, fixture)
		client := newTestClient(srv.URL)

		result, err := client.Tempban(context.Background(), TempbanParams{WlID: "WPLP", Hours: 24, Reason: "cheats"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Topic != "tempban" || result.Token != "WPLP" {
			t.Errorf("unexpected result: %+v", result)
		}
		if captured.Body["hours"].(float64) != 24 || captured.Body["reason"] != "cheats" {
			t.Errorf("unexpected body: %+v", captured.Body)
		}
	})

	t.Run("omite reason quando nao informado", func(t *testing.T) {
		srv, captured := startServer(t, 200, fixture)
		client := newTestClient(srv.URL)

		_, err := client.Tempban(context.Background(), TempbanParams{WlID: "WPLP", Hours: 24})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := captured.Body["reason"]; ok {
			t.Errorf("expected reason to be omitted, got %+v", captured.Body)
		}
	})
}

func TestWhitelistPrecadastro(t *testing.T) {
	fixture := map[string]any{"status": true, "mensagem": "ok", "code": 200, "data": map[string]any{"wl_id": "XP9T"}}

	t.Run("aplica isBanned=false por padrao e omite identificadores ausentes", func(t *testing.T) {
		srv, captured := startServer(t, 200, fixture)
		client := newTestClient(srv.URL)

		_, err := client.Precadastro(context.Background(), PrecadastroParams{Steam: "11000010865fc9a"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.Body["isBanned"] != false {
			t.Errorf("expected isBanned=false, got %+v", captured.Body["isBanned"])
		}
		if _, ok := captured.Body["discordId"]; ok {
			t.Errorf("expected discordId to be omitted, got %+v", captured.Body)
		}
	})

	t.Run("erro local quando nenhum identificador informado", func(t *testing.T) {
		client := newTestClient("http://unused.invalid")

		_, err := client.Precadastro(context.Background(), PrecadastroParams{})
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestWhitelistAtualizaToken(t *testing.T) {
	t.Run("erro local quando nenhum identificador informado", func(t *testing.T) {
		client := newTestClient("http://unused.invalid")

		_, err := client.AtualizaToken(context.Background(), AtualizaTokenParams{WlID: "WPLP"})
		if err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("envia PATCH com identificadores informados", func(t *testing.T) {
		srv, captured := startServer(t, 200, map[string]any{"status": true, "mensagem": "ok"})
		client := newTestClient(srv.URL)

		_, err := client.AtualizaToken(context.Background(), AtualizaTokenParams{WlID: "WPLP", License: "abc"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", captured.Method)
		}
		if captured.Body["license"] != "abc" {
			t.Errorf("unexpected body: %+v", captured.Body)
		}
	})
}

func TestWhitelistBanDesban(t *testing.T) {
	srv, _ := startServer(t, 200, map[string]any{"status": true, "mensagem": "ok"})
	client := newTestClient(srv.URL)

	banResult, err := client.Ban(context.Background(), BanParams{WlID: "WPLP", Reason: "cheats"})
	if err != nil || !banResult.Status {
		t.Fatalf("ban failed: %v %+v", err, banResult)
	}

	desbanResult, err := client.Desban(context.Background(), DesbanParams{WlID: "WPLP"})
	if err != nil || !desbanResult.Status {
		t.Fatalf("desban failed: %v %+v", err, desbanResult)
	}
}

func TestWhitelistMsgCommand(t *testing.T) {
	srv, captured := startServer(t, 200, map[string]any{"code": 200, "result": map[string]any{"ok": true, "id": 123}})
	client := newTestClient(srv.URL)

	result, err := client.MsgCommand(context.Background(), MsgCommandParams{
		Topic: "alert",
		Extra: map[string]any{"foo": "bar", "retries": 3},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Code != 200 {
		t.Errorf("unexpected code: %d", result.Code)
	}
	if captured.Body["topic"] != "alert" || captured.Body["foo"] != "bar" {
		t.Errorf("unexpected body: %+v", captured.Body)
	}
}

func TestWhitelistListarStaffIds(t *testing.T) {
	srv, captured := startServer(t, 200, map[string]any{
		"status": true, "code": 200, "total": 2, "cached": false, "data": []string{"111", "222"},
	})
	client := newTestClient(srv.URL)

	result, err := client.ListarStaffIds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 || len(result.Data) != 2 {
		t.Errorf("unexpected result: %+v", result)
	}
	if captured.URL != "/whitelists/staff/ids" || captured.Method != http.MethodGet {
		t.Errorf("unexpected request: %s %s", captured.Method, captured.URL)
	}
}

func TestWhitelistListarStaff(t *testing.T) {
	fixture := map[string]any{
		"status": true, "code": 200, "cached": true,
		"pagination": map[string]any{"page": 2, "limit": 10, "total": 25, "totalPages": 3},
		"data":       []any{map[string]any{"wl_id": "WPLP", "discord_id": "111", "whitelisted": true, "banned": false}},
	}

	t.Run("envia page e limit quando informados", func(t *testing.T) {
		srv, captured := startServer(t, 200, fixture)
		client := newTestClient(srv.URL)

		result, err := client.ListarStaff(context.Background(), StaffListParams{Page: 2, Limit: 10})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.Pagination.Page != 2 {
			t.Errorf("unexpected page: %d", result.Pagination.Page)
		}
		if captured.Query.Get("page") != "2" || captured.Query.Get("limit") != "10" {
			t.Errorf("unexpected query: %+v", captured.Query)
		}
	})

	t.Run("omite page/limit quando nao informados", func(t *testing.T) {
		srv, captured := startServer(t, 200, fixture)
		client := newTestClient(srv.URL)

		_, err := client.ListarStaff(context.Background(), StaffListParams{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if captured.Query.Has("page") || captured.Query.Has("limit") {
			t.Errorf("expected page/limit to be omitted, got %+v", captured.Query)
		}
	})
}

func TestWhitelistCondicaoStaff(t *testing.T) {
	srv, captured := startServer(t, 200, map[string]any{
		"status": true, "code": 200, "cached": false,
		"data": map[string]any{"discord_id": "111", "isStaff": true, "wl_id": "WPLP", "whitelisted": true, "banned": false},
	})
	client := newTestClient(srv.URL)

	result, err := client.CondicaoStaff(context.Background(), StaffConditionParams{DiscordID: "111"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Data.IsStaff || result.Data.WlID == nil || *result.Data.WlID != "WPLP" {
		t.Errorf("unexpected result: %+v", result.Data)
	}
	if captured.Query.Get("discord_id") != "111" {
		t.Errorf("unexpected query: %+v", captured.Query)
	}
}
