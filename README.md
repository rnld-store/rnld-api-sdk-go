# rnld-api-sdk-go

SDK oficial em Go para integração com a **RNLD API** — permite que criadores de bots Discord consultem e gerenciem whitelists de servidores FiveM. Port do SDK TypeScript [`@ronaldmiranda/api-sdk`](https://github.com/rnld-store/rnld-api-sdk), com paridade de funcionalidades.

## Instalação

```bash
go get github.com/rnld-store/rnld-api-sdk-go
```

## Início rápido

```go
import rnld "github.com/rnld-store/rnld-api-sdk-go"

client := rnld.NewClient(rnld.ClientOptions{
    APIKey:  "SUA_API_KEY",
    GuildID: "ID_DA_GUILD",
})
```

## Rotas disponíveis

Todos os métodos de `client.Whitelist` recebem `context.Context` como primeiro argumento.

### `Whitelist.Query(ctx, params)` — consultar jogador

Consulta o status de whitelist de um jogador. Retorna `(nil, nil)` se não encontrado.

```go
player, err := client.Whitelist.Query(ctx, rnld.QueryWhitelistParams{DiscordID: "123456789012345678"})
if err != nil {
    // erro de transporte/API — ver "Tratamento de erros"
}
if player == nil {
    fmt.Println("Jogador não encontrado")
} else {
    fmt.Println(player.IsStaff)             // bool
    fmt.Println(player.IsBanned)            // bool
    fmt.Println(player.BanReason)           // string
    fmt.Println(player.Identities.WlID)     // string
}
```

Parâmetros de `QueryWhitelistParams` (ao menos um obrigatório): `WlID`, `DiscordID`, `Steam`, `License`, `License2`.

---

### `Whitelist.Liberar(ctx, params)` — liberar jogador

Adiciona o jogador à whitelist.

```go
resultado, err := client.Whitelist.Liberar(ctx, rnld.LiberarParams{
    WlID:      "TOKEN_DO_JOGADOR",
    DiscordID: "123456789012345678", // opcional
})

fmt.Println(resultado.Status)   // bool
fmt.Println(resultado.Mensagem) // descrição do resultado
```

---

### `Whitelist.Remover(ctx, params)` — remover whitelist

```go
resultado, err := client.Whitelist.Remover(ctx, rnld.RemoverWhitelistParams{WlID: "TOKEN_DO_JOGADOR"})
```

---

### `Whitelist.Ban(ctx, params)` — banir jogador

`Reason` é opcional.

```go
resultado, err := client.Whitelist.Ban(ctx, rnld.BanParams{
    WlID:   "TOKEN_DO_JOGADOR",
    Reason: "Uso de cheats", // opcional
})
```

---

### `Whitelist.Desban(ctx, params)` — desbanir jogador

```go
resultado, err := client.Whitelist.Desban(ctx, rnld.DesbanParams{WlID: "TOKEN_DO_JOGADOR"})
```

---

### `Whitelist.Tempban(ctx, params)` — banimento temporário

Requer `WlID` e `Hours`; `Reason` é opcional.

```go
resultado, err := client.Whitelist.Tempban(ctx, rnld.TempbanParams{
    WlID:  "TOKEN_DO_JOGADOR",
    Hours: 24,
})
```

---

### `Whitelist.ListarStaffIds(ctx)` — todos os discord_ids de staff

```go
result, err := client.Whitelist.ListarStaffIds(ctx)
fmt.Println(result.Total)  // int
fmt.Println(result.Cached) // false na 1ª chamada, true nas seguintes
fmt.Println(result.Data)   // []string
```

---

### `Whitelist.ListarStaff(ctx, params)` — informações de staff paginadas

```go
result, err := client.Whitelist.ListarStaff(ctx, rnld.StaffListParams{Page: 1, Limit: 50})
fmt.Println(result.Pagination) // {Page, Limit, Total, TotalPages}
fmt.Println(result.Data)       // []StaffMember
```

`Page` (default 1) e `Limit` (default 50, máx 200) são opcionais — omita (zero value) para usar o default do servidor.

---

### `Whitelist.CondicaoStaff(ctx, params)` — condição de staff de um discord_id

Retorna sempre uma resposta (`IsStaff: false` quando não é staff).

```go
result, err := client.Whitelist.CondicaoStaff(ctx, rnld.StaffConditionParams{DiscordID: "148073845960605698"})
fmt.Println(result.Data.IsStaff) // bool
fmt.Println(result.Data.WlID)    // *string — nil quando não é staff
```

---

### Demais métodos

`Divergencia`, `DivergenciaFix`, `DeletarToken`, `LimparHwids`, `AtualizarMotivoBan`, `Staff`/`Destaff`, `Precadastro`, `AtualizaToken`, `MsgCommand` seguem o mesmo padrão — veja `types.go` para os parâmetros e respostas de cada um.

---

## Eventos em tempo real (`Client.Events`)

Conecta via WebSocket ao `rnld-bots-websocket` e despacha os eventos Redis retransmitidos. Reconecta automaticamente com backoff exponencial (1s → 30s) com jitter (50–100% do delay), evitando reconexões sincronizadas após queda do servidor.

```go
client.Events.
    OnConnected(func() {
        log.Println("conectado")
    }).
    OnBan(func(ev rnld.BanEvent) {
        log.Printf("%s banido: %s", ev.DiscordID, ev.Reason)
    }).
    OnLiberation(func(ev rnld.LiberationEvent) {
        log.Printf("%s liberado (%s)", ev.DiscordID, ev.WlID)
    })

client.Events.Connect()
defer client.Events.Close()
```

Handlers disponíveis: `OnBan`, `OnTempban`, `OnUnban`, `OnUnbanBulk`, `OnGuildWipe`, `OnNicknameFailed`, `OnUpdateNickname`, `OnUpdateRoles`, `OnLiberation`, `OnBridgeMessage`, `OnConnected`, `OnDisconnected`, `OnError`.

> **Atenção:** handlers são chamados de forma síncrona na goroutine de leitura do WebSocket — não bloqueie neles; despache trabalho pesado (I/O, chamadas de rede) para sua própria goroutine.

`WSURL` em `ClientOptions` sobrescreve o endpoint (default `wss://events.rnld.dev/ws`); necessário para usar `Client.Events`.

- `OnUnbanBulk` — desbanimento em massa; um lote pode chegar em vários chunks (mesmo `BatchID`, `Chunk` 1..`Chunks`) e substitui os eventos `unban` individuais nesse fluxo.
- `OnGuildWipe` — wipe de uma guild (`POST /mngr/wipeguild`), com `PlayersRemoved`, `TotalRemoved` e `Deleted` (linhas removidas por tabela).

### Várias guilds em uma conexão (`MultiEventsClient`)

Uma única conexão WebSocket (`/ws/multi`) para várias guilds. A autenticação é por mensagem: cada guild é assinada com a respectiva api_key. Os eventos são os mesmos do `EventsClient` e já trazem `GuildID`. Ao reconectar, todas as keys assinadas são reenviadas automaticamente; pedidos grandes são divididos em lotes de 100 keys.

```go
multi := rnld.NewMultiEventsClient("") // default wss://events.rnld.dev/ws/multi

multi.
    OnSubscribed(func(guildID string) { log.Println("assinada", guildID) }).
    OnSubscribeError(func(e rnld.SubscribeError) {
        log.Printf("assinatura recusada: %s (%s)", e.Reason, e.APIKeyHint) // invalid_key, limit_exceeded...
    })
multi.OnBan(func(ev rnld.BanEvent) {
    log.Printf("[%s] %s banido", ev.GuildID, ev.DiscordID)
})

multi.Subscribe("key-guild-a", "key-guild-b") // pode ser chamado antes ou depois do Connect
multi.Connect()
defer multi.Close()

multi.Unsubscribe("key-guild-a")
```

Handlers extras: `OnSubscribed`, `OnUnsubscribed`, `OnSubscribeError`. Requer `rnld-bots-websocket` com o endpoint `/ws/multi`.

---

## Tratamento de erros

```go
import (
    "errors"
    rnld "github.com/rnld-store/rnld-api-sdk-go"
)

_, err := client.Whitelist.Liberar(ctx, rnld.LiberarParams{WlID: "TOKEN"})
if err != nil {
    var unauthorized *rnld.UnauthorizedError
    var notFound *rnld.NotFoundError
    var apiErr *rnld.APIError

    switch {
    case errors.As(err, &unauthorized):
        // API Key inválida ou guild_id incorreto
    case errors.As(err, &notFound):
        // Jogador não existe
    case errors.As(err, &apiErr):
        fmt.Println(apiErr.StatusCode, apiErr.Message, apiErr.Body)
    }
}
```

> **Nota:** `Whitelist.Query` e `Whitelist.Divergencia` **não retornam erro** para 404 — retornam `(nil, nil)`. Os demais métodos retornam `*NotFoundError`/`*UnauthorizedError`/`*APIError` normalmente.

---

## Configuração (`ClientOptions`)

| Campo        | Tipo          | Obrigatório | Descrição                                                       |
|--------------|---------------|-------------|-------------------------------------------------------------------|
| `APIKey`     | `string`      | sim         | API Key do servidor (header `x-api-key`)                          |
| `GuildID`    | `string`      | sim         | ID da guild Discord                                                |
| `BaseURL`    | `string`      | não         | Sobrescreve a URL base da API REST (default `https://api.rnld.dev`) |
| `WSURL`      | `string`      | não         | Sobrescreve a URL do `rnld-bots-websocket` (default `wss://events.rnld.dev/ws`) |
| `HTTPClient` | `*http.Client`| não         | Cliente HTTP customizado (default `http.DefaultClient`)           |

## Exemplo — bot Discord

Veja um bot completo com `discordgo` em [`example/main.go`](./example/main.go).

```bash
cd example
cp .env.example .env   # preencha as variáveis
go run .
```

Comandos registrados automaticamente na guild:

| Comando | Descrição |
|---|---|
| `/wl-status` | Consulta whitelist por `discord_id` ou `wl_id` |
| `/wl-liberar` | Libera a whitelist de um jogador |
| `/wl-remover` | Remove a whitelist de um jogador |
| `/wl-ban` | Bane um jogador (motivo opcional) |
| `/wl-desban` | Desbane um jogador |

O exemplo também loga todos os eventos de `Client.Events`, incluindo `unban_bulk` e `guild_wipe`. Com `MULTI_API_KEYS` definido (api_keys separadas por vírgula), ele abre ainda um `MultiEventsClient` que assina todas essas guilds em uma única conexão `/ws/multi`. `MULTI_WS_URL` é opcional e sobrescreve o endpoint.

## Build e testes

```bash
go build ./...
go vet ./...
go test ./...
```
