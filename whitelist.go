package rnld

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// WhitelistClient fala com as rotas administrativas autenticadas por
// x-api-key do rnld-api.
type WhitelistClient struct {
	apiKey     string
	guildID    string
	baseURL    string
	httpClient *http.Client
}

func newWhitelistClient(apiKey, guildID, baseURL string, httpClient *http.Client) *WhitelistClient {
	return &WhitelistClient{apiKey: apiKey, guildID: guildID, baseURL: baseURL, httpClient: httpClient}
}

func setIfNotEmpty(m map[string]any, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// Query consulta o status de whitelist de um jogador. Ao menos um
// identificador em params deve ser informado.
//
// Retorna (nil, nil) se o jogador não for encontrado (404).
func (c *WhitelistClient) Query(ctx context.Context, params QueryWhitelistParams) (*QueryWhitelistResponse, error) {
	url := c.baseURL + "/whitelists/query"

	body := map[string]any{"guild_id": c.guildID}
	setIfNotEmpty(body, "wl_id", params.WlID)
	setIfNotEmpty(body, "discordId", params.DiscordID)
	setIfNotEmpty(body, "steam", params.Steam)
	setIfNotEmpty(body, "license", params.License)
	setIfNotEmpty(body, "license2", params.License2)

	var out QueryWhitelistResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		var nf *NotFoundError
		if errors.As(err, &nf) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

// Divergencia consulta uma divergência de whitelist pelo código.
//
// Retorna (nil, nil) se a divergência não for encontrada (404).
func (c *WhitelistClient) Divergencia(ctx context.Context, params DivergenciaParams) (*DivergenciaResponse, error) {
	url := c.baseURL + "/whitelists/divergence"

	query := map[string]string{"guild_id": c.guildID, "code": params.Code}

	var out DivergenciaResponse
	if err := httpGet(ctx, c.httpClient, url, c.apiKey, query, &out); err != nil {
		var nf *NotFoundError
		if errors.As(err, &nf) {
			return nil, nil
		}
		return nil, err
	}
	return &out, nil
}

// DivergenciaFix resolve uma divergência de whitelist. Requer DivergenceID.
func (c *WhitelistClient) DivergenciaFix(ctx context.Context, params DivergenciaFixParams) (*DivergenciaFixResponse, error) {
	url := c.baseURL + "/whitelists/divergencefix"

	body := map[string]any{"guild_id": c.guildID, "divergenceId": params.DivergenceID}

	var out DivergenciaFixResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// DeletarToken deleta o token de whitelist de um jogador. Requer WlID;
// ActorDiscordID é opcional.
func (c *WhitelistClient) DeletarToken(ctx context.Context, params DeletarTokenParams) (*DeletarTokenResponse, error) {
	url := c.baseURL + "/whitelists/tokenclean"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}
	setIfNotEmpty(body, "actorDiscordId", params.ActorDiscordID)

	var out DeletarTokenResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// LimparHwids limpa os HWIDs vinculados a um jogador. Requer WlID.
func (c *WhitelistClient) LimparHwids(ctx context.Context, params LimparHwidsParams) (*LimparHwidsResponse, error) {
	url := c.baseURL + "/whitelists/refresh"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}

	var out LimparHwidsResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AtualizarMotivoBan atualiza o motivo de banimento de um jogador. Requer
// WlID e Reason.
func (c *WhitelistClient) AtualizarMotivoBan(ctx context.Context, params BanReasonParams) (*BanReasonResponse, error) {
	url := c.baseURL + "/whitelists/banreason"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID, "reason": params.Reason}

	var out BanReasonResponse
	if err := httpPatch(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Staff adiciona condição de staff ao token. Requer WlID; DiscordID é
// opcional.
func (c *WhitelistClient) Staff(ctx context.Context, params StaffParams) (*StaffResponse, error) {
	url := c.baseURL + "/whitelists/staff"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}
	setIfNotEmpty(body, "discordId", params.DiscordID)

	var out StaffResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Destaff remove condição de staff do token. Requer WlID.
func (c *WhitelistClient) Destaff(ctx context.Context, params DestaffParams) (*StaffResponse, error) {
	url := c.baseURL + "/whitelists/staff"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}

	var out StaffResponse
	if err := httpDelete(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListarStaffIds lista todos os discord_ids com condição de staff na guild.
//
// A primeira consulta é quente (MySQL) e popula o cache Redis; as demais
// são servidas do cache até uma mutação de staff invalidá-lo. O campo
// Cached da resposta indica a origem.
func (c *WhitelistClient) ListarStaffIds(ctx context.Context) (*StaffIdsResponse, error) {
	url := c.baseURL + "/whitelists/staff/ids"

	query := map[string]string{"guild_id": c.guildID}

	var out StaffIdsResponse
	if err := httpGet(ctx, c.httpClient, url, c.apiKey, query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListarStaff lista, de forma paginada, as informações de staff da guild.
// Page (default 1) e Limit (default 50, máx 200) são opcionais.
func (c *WhitelistClient) ListarStaff(ctx context.Context, params StaffListParams) (*StaffListResponse, error) {
	url := c.baseURL + "/whitelists/staff/list"

	query := map[string]string{"guild_id": c.guildID}
	if params.Page != 0 {
		query["page"] = strconv.Itoa(params.Page)
	}
	if params.Limit != 0 {
		query["limit"] = strconv.Itoa(params.Limit)
	}

	var out StaffListResponse
	if err := httpGet(ctx, c.httpClient, url, c.apiKey, query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CondicaoStaff consulta a condição de staff de um único discord_id.
// Retorna sempre uma resposta (IsStaff = false quando o id não é staff).
func (c *WhitelistClient) CondicaoStaff(ctx context.Context, params StaffConditionParams) (*StaffConditionResponse, error) {
	url := c.baseURL + "/whitelists/staff/condition"

	query := map[string]string{"guild_id": c.guildID, "discord_id": params.DiscordID}

	var out StaffConditionResponse
	if err := httpGet(ctx, c.httpClient, url, c.apiKey, query, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Tempban aplica banimento temporário. Requer WlID e Hours; Reason é
// opcional.
func (c *WhitelistClient) Tempban(ctx context.Context, params TempbanParams) (*TempbanResponse, error) {
	url := c.baseURL + "/whitelists/tempban"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID, "hours": params.Hours}
	setIfNotEmpty(body, "reason", params.Reason)

	var out TempbanResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Precadastro pré-cadastra um jogador. Requer ao menos um identificador
// entre DiscordID, Steam, License ou License2.
func (c *WhitelistClient) Precadastro(ctx context.Context, params PrecadastroParams) (*PrecadastroResponse, error) {
	url := c.baseURL + "/whitelists/precadastro"

	if params.DiscordID == "" && params.Steam == "" && params.License == "" && params.License2 == "" {
		return nil, fmt.Errorf("é necessário informar ao menos um identificador: discordId, steam, license ou license2")
	}

	body := map[string]any{"guild_id": c.guildID, "isBanned": params.IsBanned}
	setIfNotEmpty(body, "discordId", params.DiscordID)
	setIfNotEmpty(body, "steam", params.Steam)
	setIfNotEmpty(body, "license", params.License)
	setIfNotEmpty(body, "license2", params.License2)
	setIfNotEmpty(body, "reason", params.Reason)

	var out PrecadastroResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// AtualizaToken atualiza os identificadores de um token. Requer WlID e ao
// menos um entre DiscordID, Steam, License ou License2.
func (c *WhitelistClient) AtualizaToken(ctx context.Context, params AtualizaTokenParams) (*AtualizaTokenResponse, error) {
	url := c.baseURL + "/whitelists/update"

	if params.DiscordID == "" && params.Steam == "" && params.License == "" && params.License2 == "" {
		return nil, fmt.Errorf("é necessário informar ao menos um identificador para atualizar: discordId, steam, license ou license2")
	}

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}
	setIfNotEmpty(body, "discordId", params.DiscordID)
	setIfNotEmpty(body, "steam", params.Steam)
	setIfNotEmpty(body, "license", params.License)
	setIfNotEmpty(body, "license2", params.License2)

	var out AtualizaTokenResponse
	if err := httpPatch(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Liberar libera (whitelist) um jogador. Requer WlID ou DiscordID.
func (c *WhitelistClient) Liberar(ctx context.Context, params LiberarParams) (*LiberarResponse, error) {
	url := c.baseURL + "/whitelists/liberar"

	body := map[string]any{"guild_id": c.guildID}
	setIfNotEmpty(body, "wl_id", params.WlID)
	setIfNotEmpty(body, "discordId", params.DiscordID)

	var out LiberarResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Remover remove a whitelist de um jogador. Requer WlID.
func (c *WhitelistClient) Remover(ctx context.Context, params RemoverWhitelistParams) (*LiberarResponse, error) {
	url := c.baseURL + "/whitelists/liberar"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}

	var out LiberarResponse
	if err := httpDelete(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Ban bane um jogador. Requer WlID; Reason é opcional.
func (c *WhitelistClient) Ban(ctx context.Context, params BanParams) (*LiberarResponse, error) {
	url := c.baseURL + "/whitelists/ban"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}
	setIfNotEmpty(body, "reason", params.Reason)

	var out LiberarResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Desban desbane um jogador. Requer WlID.
func (c *WhitelistClient) Desban(ctx context.Context, params DesbanParams) (*LiberarResponse, error) {
	url := c.baseURL + "/whitelists/ban"

	body := map[string]any{"guild_id": c.guildID, "wl_id": params.WlID}

	var out LiberarResponse
	if err := httpDelete(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// MsgCommand envia comando dinâmico para o endpoint de mensagens. Requer
// Topic; Extra aceita quaisquer outras chaves no payload.
func (c *WhitelistClient) MsgCommand(ctx context.Context, params MsgCommandParams) (*MsgCommandResponse, error) {
	url := c.baseURL + "/msg/command"

	body := map[string]any{"guild_id": c.guildID, "topic": params.Topic}
	for k, v := range params.Extra {
		body[k] = v
	}

	var out MsgCommandResponse
	if err := httpPost(ctx, c.httpClient, url, c.apiKey, body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
