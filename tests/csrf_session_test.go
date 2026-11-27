package rivex_test

import (
	"testing"
	"time"

	rivex "github.com/shregar1/rivex-go"
)

func TestSessionMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewSessionMiddleware())
	app.Get("/session", func(ctx *rivex.Context) error {
		session := rivex.GetSession(ctx)
		if session == nil {
			return rivex.NewHTTPException(500, "no session")
		}
		session.Set("user", "alice")
		return ctx.SendJSON(map[string]string{"id": session.ID, "user": session.GetString("user")})
	})
	app.Get("/check", func(ctx *rivex.Context) error {
		session := rivex.GetSession(ctx)
		if session == nil {
			return rivex.NewHTTPException(500, "no session")
		}
		return ctx.SendJSON(map[string]string{"user": session.GetString("user")})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/session")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	cookieHeader := resp.Headers["Set-Cookie"]
	if cookieHeader == "" {
		t.Fatal("expected Set-Cookie header")
	}
}

func TestSessionStore(t *testing.T) {
	store := rivex.NewInMemorySessionStore()
	if store == nil {
		t.Fatal("expected store")
	}
	session := rivex.NewSession("test-id", time.Hour)
	session.Set("key", "value")
	store.Set("test-id", session)

	got, ok := store.Get("test-id")
	if !ok {
		t.Fatal("expected to get session")
	}
	if got.GetString("key") != "value" {
		t.Errorf("got %s", got.GetString("key"))
	}

	store.Delete("test-id")
	_, ok = store.Get("test-id")
	if ok {
		t.Error("expected to be deleted")
	}
}

func TestSessionExpiration(t *testing.T) {
	session := rivex.NewSession("id", time.Millisecond*50)
	time.Sleep(time.Millisecond * 100)
	if !session.IsExpired() {
		t.Error("expected session to be expired")
	}
}

func TestSessionTouch(t *testing.T) {
	session := rivex.NewSession("id", time.Millisecond*50)
	session.Touch(time.Hour)
	if session.IsExpired() {
		t.Error("expected session to not be expired after touch")
	}
}

func TestSessionClear(t *testing.T) {
	session := rivex.NewSession("id", time.Hour)
	session.Set("a", 1)
	session.Set("b", 2)
	session.Clear()
	if _, ok := session.Get("a"); ok {
		t.Error("expected a to be cleared")
	}
	if _, ok := session.Get("b"); ok {
		t.Error("expected b to be cleared")
	}
}

func TestCSRFMiddlewareSafeMethod(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewCSRF())
	app.Get("/safe", func(ctx *rivex.Context) error {
		token := rivex.GetCSRFToken(ctx)
		return ctx.SendJSON(map[string]string{"token": token})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/safe")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestCSRFMiddlewareMissingToken(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewCSRF())
	app.Post("/unsafe", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Post("/unsafe")
	if resp.StatusCode != 403 {
		t.Errorf("expected 403 for missing CSRF token, got %d", resp.StatusCode)
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token1 := rivex.GenerateCSRFToken()
	token2 := rivex.GenerateCSRFToken()
	if token1 == token2 {
		t.Error("expected different tokens")
	}
	if len(token1) < 32 {
		t.Errorf("expected token length >= 32, got %d", len(token1))
	}
}
