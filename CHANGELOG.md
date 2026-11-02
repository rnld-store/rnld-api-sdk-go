# Changelog

## [1.2.0] - 2026-11-02
Paridade com a v1.8.0 do SDK TypeScript.
### Added
- `MultiEventsClient`: tratamento dos close codes do `rnld-bots-websocket` em `/ws/multi`, exportados como `CloseNoSubscriptions` (4001) e `CloseTooManyInvalidKeys` (4002).
  - 4001 (conexão sem nenhuma guild assinada): sem keys, a reconexão fica suspensa até o próximo `Subscribe`; com keys, reconecta normalmente.
  - 4002 (excesso de keys inválidas): dispara `OnSubscribeError` com `Reason: "too_many_invalid_keys"` e reconecta com backoff.
- `SubscribeError.Reason` pode ser `unavailable` (banco do servidor indisponível).

## [1.1.0] - 2026-11-02
Paridade com a v1.7.0 do SDK TypeScript.
### Added
- `MultiEventsClient` (`NewMultiEventsClient`) — uma única conexão WebSocket (`/ws/multi`) para várias guilds: `Subscribe(apiKeys...)` / `Unsubscribe(apiKeys...)` (lotes de 100), reassinatura automática após reconexão e handlers `OnSubscribed` / `OnUnsubscribed` / `OnSubscribeError`. Demais handlers são os mesmos do `EventsClient`. Requer `rnld-bots-websocket` com o endpoint `/ws/multi`.
- Evento `unban_bulk` (`OnUnbanBulk`, tipos `UnbanBulkEvent`/`UnbanBulkPlayer`) — desbanimento em massa em chunks por `batch_id`.
- Evento `guild_wipe` (`OnGuildWipe`, tipo `GuildWipeEvent`) — wipe de uma guild via `POST /mngr/wipeguild`.
### Changed
- `EventsClient`: reconexão com jitter (50–100% do delay) para evitar reconexões sincronizadas após queda do servidor.

## [1.0.0] - 2026-08-05
### Added
- Port inicial em Go do `@ronaldmiranda/api-sdk`, com paridade de funcionalidades com a v1.4.4 do SDK TypeScript.
- `Client.Whitelist`: `Query`, `Divergencia`, `DivergenciaFix`, `DeletarToken`, `LimparHwids`, `AtualizarMotivoBan`, `Staff`/`Destaff`, `ListarStaffIds`, `ListarStaff`, `CondicaoStaff`, `Tempban`, `Precadastro`, `AtualizaToken`, `Liberar`/`Remover`, `Ban`/`Desban`, `MsgCommand`.
- `Client.Events`: cliente WebSocket contra o `rnld-bots-websocket` com reconexão automática (backoff 1s → 30s) e handlers tipados (`OnBan`, `OnTempban`, `OnUnban`, `OnNicknameFailed`, `OnUpdateNickname`, `OnUpdateRoles`, `OnLiberation`, `OnBridgeMessage`, `OnConnected`, `OnDisconnected`, `OnError`).
- Erros tipados: `APIError`, `NotFoundError`, `UnauthorizedError` (compatíveis com `errors.As`).
- Bot de exemplo com `discordgo` em `example/`.
