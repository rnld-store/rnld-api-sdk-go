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
