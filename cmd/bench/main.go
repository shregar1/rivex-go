package main

import (
	"log"

	rivex "github.com/rivex-dev/rivex-go"
)

func main() {
	app := rivex.New()

	app.Get("/", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"message": "Hello, World!"})
	})

	app.Get("/json", func(ctx *rivex.Context) error {
		data := make([]int, 10)
		for i := range data {
			data[i] = i
		}
		return ctx.SendJSON(map[string]interface{}{
			"framework": "rivex",
			"language":  "go",
			"data":      data,
		})
	})

	app.Get("/plaintext", func(ctx *rivex.Context) error {
		return ctx.SendText("Hello, World!")
	})

	log.Fatal(app.Run("127.0.0.1", 3002))
}
