package rivex_test

import (
	"strings"
	"testing"

	rivex "github.com/shregar1/rivex-go"
)

func TestSecurityHeadersDefault(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewSecurityHeaders())
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	expectedHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Strict-Transport-Security",
		"Referrer-Policy",
	}
	for _, h := range expectedHeaders {
		if resp.Headers[h] == "" {
			t.Errorf("missing %s header", h)
		}
	}
	if resp.Headers["X-Content-Type-Options"] != "nosniff" {
		t.Errorf("expected nosniff, got %s", resp.Headers["X-Content-Type-Options"])
	}
	if resp.Headers["X-Frame-Options"] != "DENY" {
		t.Errorf("expected DENY, got %s", resp.Headers["X-Frame-Options"])
	}
}

func TestSecurityHeadersCustom(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewSecurityHeaders(
		rivex.WithXFrameOptions("SAMEORIGIN"),
		rivex.WithHSTS(63072000, true, true),
	))
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	if resp.Headers["X-Frame-Options"] != "SAMEORIGIN" {
		t.Errorf("expected SAMEORIGIN, got %s", resp.Headers["X-Frame-Options"])
	}
	if !strings.Contains(resp.Headers["Strict-Transport-Security"], "max-age=63072000") {
		t.Errorf("expected custom max-age, got %s", resp.Headers["Strict-Transport-Security"])
	}
	if !strings.Contains(resp.Headers["Strict-Transport-Security"], "preload") {
		t.Errorf("expected preload, got %s", resp.Headers["Strict-Transport-Security"])
	}
}

func TestCSPMiddlewareDefault(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewCSP())
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	csp := resp.Headers["Content-Security-Policy"]
	if csp == "" {
		t.Error("expected CSP header")
	}
	if !strings.Contains(csp, "default-src") {
		t.Errorf("expected default-src, got %s", csp)
	}
}

func TestCSPMiddlewareReportOnly(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewCSP(rivex.WithCSPReportOnly(true)))
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	if resp.Headers["Content-Security-Policy-Report-Only"] == "" {
		t.Error("expected report-only header")
	}
}

func TestHSTSMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewHSTS(rivex.WithHSTSMaxAge(31536000)))
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	hsts := resp.Headers["Strict-Transport-Security"]
	if !strings.Contains(hsts, "max-age=31536000") {
		t.Errorf("expected max-age=31536000, got %s", hsts)
	}
	if !strings.Contains(hsts, "includeSubDomains") {
		t.Errorf("expected includeSubDomains, got %s", hsts)
	}
}

func TestXFrameOptionsMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewXFrameOptions("SAMEORIGIN"))
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	if resp.Headers["X-Frame-Options"] != "SAMEORIGIN" {
		t.Errorf("expected SAMEORIGIN, got %s", resp.Headers["X-Frame-Options"])
	}
}

func TestReferrerPolicyMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewReferrerPolicy("no-referrer"))
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	if resp.Headers["Referrer-Policy"] != "no-referrer" {
		t.Errorf("expected no-referrer, got %s", resp.Headers["Referrer-Policy"])
	}
}

func TestNewRestrictiveCSP(t *testing.T) {
	csp := rivex.NewRestrictiveCSP()
	if csp == nil {
		t.Fatal("expected csp")
	}
}

func TestUserAgentMiddlewareMissing(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewUserAgentMiddleware())
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	if resp.StatusCode != 400 {
		t.Errorf("expected 400 for missing UA, got %d", resp.StatusCode)
	}
}

func TestIPFilterMiddlewareBlocked(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewIPFilterMiddleware(
		rivex.WithBlacklistedIPs("10.0.0.0/8"),
	))
	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 (test client uses 127.0.0.1), got %d", resp.StatusCode)
	}
}
