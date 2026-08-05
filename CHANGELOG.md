# Changelog

## [1.0.0] - 2026-08-05
### Added
- Port inicial em Go do `@ronaldmiranda/api-sdk`, com paridade de funcionalidades com a v1.4.4 do SDK TypeScript.
- `Client.Whitelist`: `Query`, `Divergencia`, `DivergenciaFix`, `DeletarToken`, `LimparHwids`, `AtualizarMotivoBan`, `Staff`/`Destaff`, `ListarStaffIds`, `ListarStaff`, `CondicaoStaff`, `Tempban`, `Precadastro`, `AtualizaToken`, `Liberar`/`Remover`, `Ban`/`Desban`, `MsgCommand`.
- `Client.Events`: cliente WebSocket contra o `rnld-bots-websocket` com reconexão automática (backoff 1s → 30s) e handlers tipados (`OnBan`, `OnTempban`, `OnUnban`, `OnNicknameFailed`, `OnUpdateNickname`, `OnUpdateRoles`, `OnLiberation`, `OnBridgeMessage`, `OnConnected`, `OnDisconnected`, `OnError`).
- Erros tipados: `APIError`, `NotFoundError`, `UnauthorizedError` (compatíveis com `errors.As`).
- Bot de exemplo com `discordgo` em `example/`.
