package main

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/rivex-dev/rivex-go"
)

var requestCounter int64

// requestLogger prints one line per request after the handler runs.
func requestLogger(ctx *rivex.Context, next rivex.NextFunc) error {
	start := time.Now()
	err := next(ctx)
	fmt.Printf("[%s] %s %s -> %v\n",
		time.Now().Format(time.RFC3339), ctx.Method(), ctx.Path(), time.Since(start))
	return err
}

// requestCounterMW assigns each request a monotonic ID accessible via ctx.Get.
func requestCounterMW(ctx *rivex.Context, next rivex.NextFunc) error {
	id := atomic.AddInt64(&requestCounter, 1)
	ctx.Set("request_num", id)
	return next(ctx)
}

func main() {
	app := rivex.New(rivex.WithTitle("Middleware Chain Example"))

	app.Use(
		rivex.MiddlewareFunc(requestLogger),
		rivex.MiddlewareFunc(requestCounterMW),
		rivex.NewRequestID(),
		rivex.NewCORS(rivex.AllowOrigins("*")),
	)

	app.Get("/", func(ctx *rivex.Context) error {
		num, _ := ctx.Get("request_num")
		return ctx.SendJSON(map[string]interface{}{
			"message":     "middleware chain ran",
			"request_num": num,
		})
	})

	app.Get("/slow", func(ctx *rivex.Context) error {
		time.Sleep(100 * time.Millisecond)
		return ctx.SendJSON(map[string]string{"message": "slept 100ms"})
	})

	app.Run("127.0.0.1", 8000)
}
