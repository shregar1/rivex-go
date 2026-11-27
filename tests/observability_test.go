package rivex_test

import (
	"log"
	"os"
	"testing"

	rivex "github.com/shregar1/rivex-go"
)

func TestMetricsMiddleware(t *testing.T) {
	m := rivex.NewMetricsMiddleware()
	app := rivex.New()
	app.Use(m)
	app.Get("/x", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})
	app.Get("/y", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	client.Get("/x")
	client.Get("/x")
	client.Get("/y")

	snap := m.Snapshot()
	if snap["total_requests"].(int64) != 3 {
		t.Errorf("expected 3 requests, got %v", snap["total_requests"])
	}
	if snap["total_bytes_out"].(int64) == 0 {
		t.Error("expected non-zero bytes_out")
	}
}

func TestMetricsMiddlewareError(t *testing.T) {
	m := rivex.NewMetricsMiddleware()
	app := rivex.New()
	app.Use(m)
	app.Get("/err", func(ctx *rivex.Context) error {
		return rivex.NewHTTPException(500, "fail")
	})

	client := rivex.NewTestClient(app)
	client.Get("/err")

	snap := m.Snapshot()
	if snap["total_errors"].(int64) != 1 {
		t.Errorf("expected 1 error, got %v", snap["total_errors"])
	}
}

func TestMetricsMiddlewareSkipPath(t *testing.T) {
	m := rivex.NewMetricsMiddleware(rivex.WithMetricsSkipPaths("/health"))
	app := rivex.New()
	app.Use(m)
	app.Get("/health", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	client.Get("/health")

	snap := m.Snapshot()
	if snap["total_requests"].(int64) != 0 {
		t.Errorf("expected 0 requests (path skipped), got %v", snap["total_requests"])
	}
}

func TestLoggingMiddleware(t *testing.T) {
	var buf []byte
	logger := log.New(&byteSliceWriter{buf: &buf}, "", 0)
	m := rivex.NewLoggingMiddleware(rivex.WithLogger(logger))
	app := rivex.New()
	app.Use(m)
	app.Get("/x", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	client.Get("/x")
	if len(buf) == 0 {
		t.Error("expected log output")
	}
}

type byteSliceWriter struct {
	buf *[]byte
}

func (w *byteSliceWriter) Write(p []byte) (int, error) {
	*w.buf = append(*w.buf, p...)
	return len(p), nil
}

func TestHealthCheck(t *testing.T) {
	hc := rivex.NewHealthCheck(rivex.WithHealthVersion("1.0"))
	app := rivex.New()
	app.Use(rivex.NewRequestID()) // need a middleware that doesn't intercept
	hcHandler := func(ctx *rivex.Context) error {
		return hc.Process(ctx, func(ctx *rivex.Context) error {
			return nil
		})
	}
	app.Get("/health", hcHandler)

	client := rivex.NewTestClient(app)
	resp := client.Get("/health")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestHealthCheckDegraded(t *testing.T) {
	hc := rivex.NewHealthCheck(
		rivex.WithHealthCheck("db", func() (string, error) {
			return "", os.ErrNotExist
		}),
	)
	app := rivex.New()
	app.Get("/health", func(ctx *rivex.Context) error {
		return hc.Process(ctx, func(ctx *rivex.Context) error {
			return nil
		})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/health")
	if resp.StatusCode != 503 {
		t.Errorf("expected 503 for degraded, got %d", resp.StatusCode)
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	m := rivex.NewRateLimitMiddleware(100, 5)
	app := rivex.New()
	app.Use(m)
	app.Get("/x", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/x")
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestIPRateLimitKey(t *testing.T) {
	app := rivex.New()
	app.Get("/", func(ctx *rivex.Context) error {
		key := rivex.IPRateLimitKey(ctx)
		return ctx.SendJSON(map[string]string{"key": key})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/")
	var body map[string]string
	resp.JSON(&body)
	// The test client may not set RemoteAddr, so we accept any non-empty result or "unknown"
	if body["key"] == "" {
		t.Error("expected non-empty key")
	}
}
