package rnld

import "testing"

func TestEventsDispatchBan(t *testing.T) {
	e := newEventsClient(testAPIKey, "wss://unused.invalid")

	var got *BanEvent
	e.OnBan(func(ev BanEvent) { got = &ev })

	e.dispatch([]byte(`{"channel":"rnld_ban_events","payload":{"type":"ban","guild_id":"123","discord_id":"111","wl_id":"WPLP","reason":"cheats","ts":"1"}}`))

	if got == nil {
		t.Fatal("expected ban handler to be called")
	}
	if got.WlID != "WPLP" || got.Reason != "cheats" {
		t.Errorf("unexpected event: %+v", got)
	}
}

func TestEventsDispatchLiberation(t *testing.T) {
	e := newEventsClient(testAPIKey, "wss://unused.invalid")

	var got *LiberationEvent
	e.OnLiberation(func(ev LiberationEvent) { got = &ev })

	e.dispatch([]byte(`{"channel":"rnld_liberation_events","payload":{"guild_id":"123","discord_id":"111","wl_id":"WPLP","source":"RNLD-Manager","ts":"1"}}`))

	if got == nil {
		t.Fatal("expected liberation handler to be called")
	}
	if got.Source != "RNLD-Manager" {
		t.Errorf("unexpected event: %+v", got)
	}
}

func TestEventsDispatchUnknownTypeIsIgnored(t *testing.T) {
	e := newEventsClient(testAPIKey, "wss://unused.invalid")

	called := false
	e.OnBan(func(ev BanEvent) { called = true })

	e.dispatch([]byte(`{"channel":"rnld_roles_events","payload":{"type":"somethingElse"}}`))

	if called {
		t.Error("expected no handler to be called for unknown type")
	}
}

func TestEventsDispatchMalformedIsIgnored(t *testing.T) {
	e := newEventsClient(testAPIKey, "wss://unused.invalid")

	// Não deve gerar panic nem erro visível.
	e.dispatch([]byte(`not json`))
}

func TestEventsDispatchUnbanBulk(t *testing.T) {
	e := newEventsClient(testAPIKey, "wss://unused.invalid")

	var got *UnbanBulkEvent
	e.OnUnbanBulk(func(ev UnbanBulkEvent) { got = &ev })

	e.dispatch([]byte(`{"channel":"rnld_ban_events","payload":{"type":"unban_bulk","guild_id":"123","batch_id":"b1","chunk":1,"chunks":2,"total":3,"players":[{"discord_id":"111","wl_id":"WPLP","whitelisted":1}],"ts":"1"}}`))

	if got == nil {
		t.Fatal("expected unban_bulk handler to be called")
	}
	if got.BatchID != "b1" || got.Chunks != 2 || len(got.Players) != 1 || got.Players[0].Whitelisted != 1 {
		t.Errorf("unexpected event: %+v", got)
	}
}

func TestEventsDispatchGuildWipe(t *testing.T) {
	e := newEventsClient(testAPIKey, "wss://unused.invalid")

	var got *GuildWipeEvent
	e.OnGuildWipe(func(ev GuildWipeEvent) { got = &ev })

	e.dispatch([]byte(`{"channel":"rnld_ban_events","payload":{"type":"guild_wipe","guild_id":"123","wl_id":null,"players_removed":842,"total_removed":2942,"deleted":{"players":842,"player_tokens":2100},"ts":"1"}}`))

	if got == nil {
		t.Fatal("expected guild_wipe handler to be called")
	}
	if got.WlID != nil || got.PlayersRemoved != 842 || got.Deleted["player_tokens"] != 2100 {
		t.Errorf("unexpected event: %+v", got)
	}
}

func TestJitterRange(t *testing.T) {
	for range 100 {
		d := jitter(minReconnectDelay)
		if d < minReconnectDelay/2 || d > minReconnectDelay {
			t.Fatalf("jitter out of range: %v", d)
		}
	}
}
