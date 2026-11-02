package rnld

// ── rnld_ban_events ──────────────────────────────────────────────────────

type BanEvent struct {
	Type      string `json:"type"`
	GuildID   string `json:"guild_id"`
	DiscordID string `json:"discord_id"`
	WlID      string `json:"wl_id"`
	Reason    string `json:"reason"`
	Ts        string `json:"ts"`
}

type TempBanEvent struct {
	Type      string `json:"type"`
	GuildID   string `json:"guild_id"`
	DiscordID string `json:"discord_id"`
	WlID      string `json:"wl_id"`
	Reason    string `json:"reason"`
	ExpiresAt string `json:"expires_at"`
	Ts        string `json:"ts"`
}

type UnbanEvent struct {
	Type      string `json:"type"`
	GuildID   string `json:"guild_id"`
	DiscordID string `json:"discord_id"`
	WlID      string `json:"wl_id"`
	Reason    string `json:"reason"`
	// Subtype é "auto_expired" quando o tempban expirou automaticamente.
	Subtype string `json:"subtype,omitempty"`
	Ts      string `json:"ts"`
}

// UnbanBulkPlayer é o alvo de um desbanimento em massa (só players com
// discord_id).
type UnbanBulkPlayer struct {
	DiscordID string `json:"discord_id"`
	WlID      string `json:"wl_id"`
	// Whitelisted é 1 se o player é whitelistado (define qual cargo de
	// cidadão devolver), 0 caso contrário.
	Whitelisted int `json:"whitelisted"`
}

// UnbanBulkEvent é um lote de desbanimento em massa. Um lote pode vir em
// vários chunks (mesmo BatchID, Chunk 1..Chunks) para limitar o tamanho da
// mensagem. Substitui os eventos "unban" individuais nesse fluxo.
type UnbanBulkEvent struct {
	Type    string `json:"type"`
	GuildID string `json:"guild_id"`
	BatchID string `json:"batch_id"`
	Chunk   int    `json:"chunk"`
	Chunks  int    `json:"chunks"`
	// Total de banimentos removidos no lote (inclui players sem discord_id).
	Total   int               `json:"total"`
	Players []UnbanBulkPlayer `json:"players"`
	Ts      string            `json:"ts"`
}

// GuildWipeEvent é o wipe de uma guild (POST /mngr/wipeguild): os dados dos
// players foram apagados em todas as tabelas com guild_id (a config/api_key
// da guild é preservada).
type GuildWipeEvent struct {
	Type    string `json:"type"`
	GuildID string `json:"guild_id"`
	// WlID é sempre nil: o wipe não é sobre um player específico.
	WlID *string `json:"wl_id"`
	// PlayersRemoved são as linhas removidas de players.
	PlayersRemoved int `json:"players_removed"`
	// TotalRemoved é a soma de todas as linhas removidas, em todas as tabelas.
	TotalRemoved int `json:"total_removed"`
	// Deleted são as linhas removidas por tabela
	// (ex.: {"players": 842, "player_tokens": 2100}).
	Deleted map[string]int `json:"deleted"`
	Ts      string         `json:"ts"`
}

// ── rnld_nickname_events ─────────────────────────────────────────────────

// NicknameFailedEvent cobre as duas variantes do evento (discriminadas por
// Reason): "member_not_found" preenche DiscordID/WlID; "discord_id_not_found"
// preenche License.
type NicknameFailedEvent struct {
	Type string `json:"type"`
	// Reason é "member_not_found" ou "discord_id_not_found".
	Reason    string `json:"reason"`
	GuildID   string `json:"guild_id"`
	DiscordID string `json:"discord_id,omitempty"`
	WlID      string `json:"wl_id,omitempty"`
	License   string `json:"license,omitempty"`
	Nickname  string `json:"nickname"`
	Ts        string `json:"ts"`
}

type UpdateNicknameEvent struct {
	Type        string `json:"type"`
	GuildID     string `json:"guild_id"`
	DiscordID   string `json:"discord_id"`
	NewNickname string `json:"new_nickname"`
	Ts          string `json:"ts"`
}

// ── rnld_roles_events ────────────────────────────────────────────────────

type UpdateRolesEvent struct {
	Type        string   `json:"type"`
	GuildID     string   `json:"guild_id"`
	DiscordID   string   `json:"discord_id"`
	RolesAdd    []string `json:"roles_add"`
	RolesRemove []string `json:"roles_remove"`
	Ts          string   `json:"ts"`
}

// ── rnld_liberation_events ───────────────────────────────────────────────

type LiberationEvent struct {
	GuildID   string `json:"guild_id"`
	DiscordID string `json:"discord_id"`
	WlID      string `json:"wl_id"`
	// Source é a origem: header X-Source (ex.: "RNLD-Manager") ou
	// "RNLD-API/Release" quando disparado pelo user-agent do sistema.
	Source string `json:"source,omitempty"`
	Ts     string `json:"ts"`
}

// ── rnld_channelbridge_events ────────────────────────────────────────────

type BridgeAttachment struct {
	URL         string  `json:"url"`
	Name        string  `json:"name"`
	ContentType *string `json:"content_type"`
}

type BridgeMessageEvent struct {
	Type              string             `json:"type"`
	SourceGuildID     string             `json:"source_guild_id"`
	SourceGuildName   string             `json:"source_guild_name"`
	SourceGuildIcon   *string            `json:"source_guild_icon"`
	SourceChannelID   string             `json:"source_channel_id"`
	SourceChannelName *string            `json:"source_channel_name"`
	AuthorID          string             `json:"author_id"`
	AuthorUsername    string             `json:"author_username"`
	AuthorTag         string             `json:"author_tag"`
	AuthorNickname    string             `json:"author_nickname"`
	AuthorAvatar      *string            `json:"author_avatar"`
	Content           string             `json:"content"`
	Attachments       []BridgeAttachment `json:"attachments"`
	Ts                string             `json:"ts"`
}
