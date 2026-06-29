package main

import (
	"github.com/rivex-dev/rivex-go"
)

func main() {
	app := rivex.New(rivex.WithTitle("Hello API"))

	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"message": "Hello, World!"})
	})

	app.Get("/hello/{name}", func(ctx *rivex.Context) error {
		name := ctx.PathParam("name")
		greeting := ctx.QueryParam("greeting")
		if greeting == "" {
			greeting = "Hello"
		}
		return ctx.SendJSON(map[string]string{
			"message": greeting + ", " + name + "!",
		})
	})

	app.Run("127.0.0.1", 8000)
}
