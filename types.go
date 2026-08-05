package rnld

// ---------- Query ----------

// QueryWhitelistParams identifica o jogador a consultar. Ao menos um campo
// deve ser informado.
type QueryWhitelistParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// DiscordID do jogador.
	DiscordID string
	// Steam é o Steam Hex.
	Steam string
	// License principal (Rockstar).
	License string
	// License2 é a license secundária.
	License2 string
}

type BanHistoryEntry struct {
	Action       string  `json:"action"`
	BanReason    *string `json:"ban_reason"`
	BanExpiresAt *string `json:"ban_expires_at"`
	ChangedAt    string  `json:"changed_at"`
}

type PlayerIdentities struct {
	GuildID     string `json:"guild_id"`
	DiscordID   string `json:"discord_id"`
	SteamHex    string `json:"steamHex"`
	License     string `json:"license"`
	License2    string `json:"license2"`
	Fivem       string `json:"fivem"`
	Live        string `json:"live"`
	Whitelisted int    `json:"whitelisted"`
	// ID é o token amigável (encoded).
	ID string `json:"id"`
	// WlID é o token amigável (encoded) — igual a ID.
	WlID string `json:"wl_id"`
}

type QueryWhitelistResponse struct {
	IsStaff      bool              `json:"isStaff"`
	IsBanned     bool              `json:"isBanned"`
	BanReason    string            `json:"banReason"`
	BanExpiresAt *string           `json:"banExpiresAt"`
	BanHistory   []BanHistoryEntry `json:"banHistory"`
	LastIP       *string           `json:"last_ip"`
	Identities   PlayerIdentities  `json:"identities"`
	Code         int               `json:"code"`
	Mensagem     string            `json:"mensagem"`
}

type DefaultResponse struct {
	Status   bool   `json:"status"`
	Mensagem string `json:"mensagem"`
	Code     int    `json:"code,omitempty"`
}

// ---------- Divergência ----------

type DivergenciaParams struct {
	// Code é o código da divergência.
	Code string
}

type DivergenceData struct {
	DivergenceID   string `json:"divergenceId"`
	PlayerID       string `json:"player_id"`
	DiscordID      string `json:"discord_id"`
	SteamHex       string `json:"steamhex"`
	License        string `json:"license"`
	License2       string `json:"license2"`
	Resolved       int    `json:"resolved"`
	DivergenceCode string `json:"divergence_code"`
}

type DivergenciaResponse struct {
	Status   bool           `json:"status"`
	Data     DivergenceData `json:"data"`
	Mensagem string         `json:"mensagem"`
	Code     int            `json:"code"`
}

type DivergenciaFixParams struct {
	// DivergenceID é o ID da divergência.
	DivergenceID string
}

type DivergenciaFixIdentitiesUpdated struct {
	DiscordID string `json:"discord_id"`
	SteamHex  string `json:"steamhex"`
	License   string `json:"license"`
	License2  string `json:"license2"`
}

type DivergenciaFixData struct {
	DivergenceID      string                          `json:"divergenceId"`
	PlayerID          string                          `json:"player_id"`
	IdentitiesUpdated DivergenciaFixIdentitiesUpdated `json:"identities_updated"`
}

type DivergenciaFixResponse struct {
	Status   bool               `json:"status"`
	Data     DivergenciaFixData `json:"data"`
	Mensagem string             `json:"mensagem"`
	Code     int                `json:"code"`
}

type DeletarTokenParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// ActorDiscordID é o Discord ID do ator responsável pela ação.
	ActorDiscordID string
}

type DeletarTokenResponse = DefaultResponse

type LimparHwidsParams struct {
	// WlID é o token amigável do jogador.
	WlID string
}

type LimparHwidsResponse = DefaultResponse

// ---------- Liberar ----------

type LiberarParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// DiscordID do jogador.
	DiscordID string
}

type LiberarResponse = DefaultResponse

type BanReasonParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// Reason é o novo motivo do banimento.
	Reason string
}

type BanReasonResponse = DefaultResponse

type StaffParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// DiscordID do jogador (opcional).
	DiscordID string
}

type DestaffParams struct {
	// WlID é o token amigável do jogador.
	WlID string
}

type StaffResponse = DefaultResponse

// ---------- Staff (leitura) ----------

type StaffMember struct {
	// WlID é o token amigável do jogador.
	WlID        string `json:"wl_id"`
	DiscordID   string `json:"discord_id"`
	Whitelisted bool   `json:"whitelisted"`
	Banned      bool   `json:"banned"`
}

type StaffIdsResponse struct {
	Status bool     `json:"status"`
	Code   int      `json:"code"`
	Total  int      `json:"total"`
	Cached bool     `json:"cached"`
	Data   []string `json:"data"`
}

type StaffListParams struct {
	// Page (default 1).
	Page int
	// Limit por página (default 50, máx 200).
	Limit int
}

type StaffPagination struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

type StaffListResponse struct {
	Status     bool            `json:"status"`
	Code       int             `json:"code"`
	Cached     bool            `json:"cached"`
	Pagination StaffPagination `json:"pagination"`
	Data       []StaffMember   `json:"data"`
}

type StaffConditionParams struct {
	// DiscordID a consultar.
	DiscordID string
}

type StaffConditionData struct {
	DiscordID   string  `json:"discord_id"`
	IsStaff     bool    `json:"isStaff"`
	WlID        *string `json:"wl_id"`
	Whitelisted *bool   `json:"whitelisted"`
	Banned      *bool   `json:"banned"`
}

type StaffConditionResponse struct {
	Status bool               `json:"status"`
	Code   int                `json:"code"`
	Cached bool               `json:"cached"`
	Data   StaffConditionData `json:"data"`
}

type TempbanParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// Hours é a duração do banimento temporário em horas.
	Hours int
	// Reason do banimento temporário.
	Reason string
}

type TempbanResponse struct {
	Status    bool   `json:"status"`
	Code      int    `json:"code"`
	Topic     string `json:"topic"`
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	Mensagem  string `json:"mensagem"`
}

type PrecadastroParams struct {
	// DiscordID do jogador.
	DiscordID string
	// Steam Hex.
	Steam string
	// License principal (Rockstar).
	License string
	// License2 secundária.
	License2 string
	// IsBanned marca como banido no pré-cadastro.
	IsBanned bool
	// Reason do banimento (quando aplicável).
	Reason string
}

type PrecadastroResponse struct {
	Status   bool   `json:"status"`
	Mensagem string `json:"mensagem"`
	Code     int    `json:"code"`
	Data     struct {
		WlID string `json:"wl_id"`
	} `json:"data"`
}

type AtualizaTokenParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// DiscordID do jogador.
	DiscordID string
	// Steam Hex.
	Steam string
	// License principal (Rockstar).
	License string
	// License2 secundária.
	License2 string
}

type AtualizaTokenResponse = DefaultResponse

// ---------- Remover whitelist ----------

type RemoverWhitelistParams struct {
	// WlID é o token amigável do jogador.
	WlID string
}

// ---------- Ban / Desban ----------

type BanParams struct {
	// WlID é o token amigável do jogador.
	WlID string
	// Reason do banimento.
	Reason string
}

type DesbanParams struct {
	// WlID é o token amigável do jogador.
	WlID string
}

// ---------- Msg Command ----------

type MsgCommandParams struct {
	// Topic da mensagem.
	Topic string
	// Extra são chaves adicionais dinâmicas enviadas no payload.
	Extra map[string]any
}

type MsgCommandResponse struct {
	Code   int `json:"code"`
	Result any `json:"result"`
}
