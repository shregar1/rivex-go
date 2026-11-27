package rivex_test

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	rivex "github.com/shregar1/rivex-go"
)

func TestJWTAuthRoundtrip(t *testing.T) {
	secret := "test-secret-123"
	app := rivex.New()
	app.Use(rivex.NewJWTAuth(secret))
	app.Get("/protected", func(ctx *rivex.Context) error {
		claims := rivex.GetClaims(ctx)
		if claims == nil {
			return rivex.NewHTTPException(401, "no claims")
		}
		return ctx.SendJSON(map[string]interface{}{"user": claims.Subject})
	})

	jwt := rivex.NewJWTAuth(secret)
	claims := &rivex.Claims{
		Subject:  "user-1",
		Expiry:   time.Now().Add(time.Hour).Unix(),
		IssuedAt: time.Now().Unix(),
		Roles:    []string{"admin", "editor"},
	}
	token, err := jwt.GenerateToken(claims)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if strings.Count(token, ".") != 2 {
		t.Errorf("expected JWT to have 3 segments, got %d dots in %s", strings.Count(token, "."), token)
	}

	client := rivex.NewTestClient(app)
	resp := client.Get("/protected", rivex.WithHeader("Authorization", "Bearer "+token))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Text())
	}
	var body map[string]interface{}
	resp.JSON(&body)
	if body["user"] != "user-1" {
		t.Errorf("expected user=user-1, got %v", body["user"])
	}
}

func TestJWTAuthMissingToken(t *testing.T) {
	secret := "test-secret"
	app := rivex.New()
	app.Use(rivex.NewJWTAuth(secret))
	app.Get("/protected", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/protected")
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthInvalidToken(t *testing.T) {
	secret := "test-secret"
	app := rivex.New()
	app.Use(rivex.NewJWTAuth(secret))
	app.Get("/protected", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/protected", rivex.WithHeader("Authorization", "Bearer invalid.token.here"))
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthWrongSecret(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewJWTAuth("real-secret"))
	app.Get("/protected", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	jwt := rivex.NewJWTAuth("wrong-secret")
	token, _ := jwt.GenerateToken(&rivex.Claims{Subject: "u", Expiry: time.Now().Add(time.Hour).Unix()})

	client := rivex.NewTestClient(app)
	resp := client.Get("/protected", rivex.WithHeader("Authorization", "Bearer "+token))
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestJWTAuthCookieExtractor(t *testing.T) {
	secret := "test-secret"
	app := rivex.New()
	app.Use(rivex.NewJWTAuth(secret, rivex.WithTokenExtractor(rivex.CookieTokenExtractor("auth"))))
	app.Get("/protected", func(ctx *rivex.Context) error {
		claims := rivex.GetClaims(ctx)
		if claims == nil {
			return rivex.NewHTTPException(401, "no claims")
		}
		return ctx.SendJSON(map[string]string{"user": claims.Subject})
	})

	jwt := rivex.NewJWTAuth(secret)
	token, _ := jwt.GenerateToken(&rivex.Claims{Subject: "cookie-user", Expiry: time.Now().Add(time.Hour).Unix()})

	client := rivex.NewTestClient(app)
	resp := client.Get("/protected", rivex.WithHeader("Cookie", "auth="+token))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Text())
	}
}

func TestJWTExpired(t *testing.T) {
	secret := "test-secret"
	app := rivex.New()
	app.Use(rivex.NewJWTAuth(secret))
	app.Get("/protected", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	jwt := rivex.NewJWTAuth(secret)
	token, _ := jwt.GenerateToken(&rivex.Claims{
		Subject: "u",
		Expiry:  time.Now().Add(-time.Hour).Unix(),
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/protected", rivex.WithHeader("Authorization", "Bearer "+token))
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401 for expired token, got %d", resp.StatusCode)
	}
}

func TestRequiresRolesGuard(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewJWTAuth("secret"))
	app.Use(rivex.NewGuard(rivex.RequiresUserID(), rivex.RequiresRoles("admin")))
	app.Get("/admin", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	jwt := rivex.NewJWTAuth("secret")
	adminToken, _ := jwt.GenerateToken(&rivex.Claims{
		Subject: "u1",
		Roles:   []string{"admin"},
		Expiry:  time.Now().Add(time.Hour).Unix(),
	})
	userToken, _ := jwt.GenerateToken(&rivex.Claims{
		Subject: "u2",
		Roles:   []string{"user"},
		Expiry:  time.Now().Add(time.Hour).Unix(),
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/admin", rivex.WithHeader("Authorization", "Bearer "+adminToken))
	if resp.StatusCode != 200 {
		t.Errorf("admin: expected 200, got %d", resp.StatusCode)
	}

	resp = client.Get("/admin", rivex.WithHeader("Authorization", "Bearer "+userToken))
	if resp.StatusCode != 403 {
		t.Errorf("non-admin: expected 403, got %d", resp.StatusCode)
	}
}

func TestHTTPBasicAuthResolve(t *testing.T) {
	app := rivex.New()
	basic := rivex.NewHTTPBasic()
	app.Get("/basic", func(ctx *rivex.Context) error {
		creds, err := basic.Resolve(ctx)
		if err != nil {
			return err
		}
		bc := creds.(*rivex.BasicCredentials)
		return ctx.SendJSON(map[string]string{"user": bc.Username})
	})

	encoded := base64.StdEncoding.EncodeToString([]byte("alice:secret"))
	client := rivex.NewTestClient(app)
	resp := client.Get("/basic", rivex.WithHeader("Authorization", "Basic "+encoded))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var body map[string]string
	resp.JSON(&body)
	if body["user"] != "alice" {
		t.Errorf("expected user=alice, got %s", body["user"])
	}
}

func TestAPIKeyQuery(t *testing.T) {
	apiKey := rivex.NewAPIKeyQuery("key")
	app := rivex.New()
	app.Get("/q", func(ctx *rivex.Context) error {
		val, err := apiKey.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"key": val})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/q", rivex.WithQueryParams(map[string]string{"key": "secret123"}))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, resp.Text())
	}
}

func TestAPIKeyCookie(t *testing.T) {
	apiKey := rivex.NewAPIKeyCookie("session_token")
	app := rivex.New()
	app.Get("/cookie-auth", func(ctx *rivex.Context) error {
		val, err := apiKey.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"token": val})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/cookie-auth", rivex.WithHeader("Cookie", "session_token=tok-abc"))
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestClaimsValid(t *testing.T) {
	now := time.Now()
	c := &rivex.Claims{Expiry: now.Add(time.Hour).Unix()}
	if err := c.Valid(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}

	c = &rivex.Claims{Expiry: now.Add(-time.Hour).Unix()}
	if err := c.Valid(); err == nil {
		t.Error("expected expired error")
	}

	c = &rivex.Claims{NotBefore: now.Add(time.Hour).Unix()}
	if err := c.Valid(); err == nil {
		t.Error("expected not-yet-valid error")
	}
}
