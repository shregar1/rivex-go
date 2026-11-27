// Package main demonstrates the in-process TestClient.
//
// Run with: go run ./examples/testing_demo
//
// It builds a tiny Rivex app and exercises it without starting a real
// server, which is the same approach you'd use from a *_test.go file.
package main

import (
	"fmt"

	"github.com/shregar1/rivex-go"
)

type Greeting struct {
	Message string `json:"message"`
}

func buildApp() *rivex.Rivex {
	app := rivex.New(rivex.WithTitle("Testing Demo"))

	app.Get("/hello/{name}", func(ctx *rivex.Context) error {
		return ctx.SendJSON(Greeting{
			Message: "Hello, " + ctx.PathParam("name") + "!",
		})
	})

	app.Post("/echo", func(ctx *rivex.Context) error {
		var body map[string]interface{}
		if err := ctx.JSON(&body); err != nil {
			return rivex.NewHTTPException(rivex.StatusBadRequest, "invalid json")
		}
		return ctx.SendJSON(body)
	})

	return app
}

func main() {
	app := buildApp()
	client := rivex.NewTestClient(app)

	resp := client.Get("/hello/Ada")
	fmt.Printf("GET /hello/Ada -> %d\n", resp.StatusCode)

	var g Greeting
	if err := resp.JSON(&g); err != nil {
		panic(err)
	}
	fmt.Printf("  message=%q\n", g.Message)

	echoResp := client.Post("/echo", rivex.WithJSONBody(map[string]string{"hi": "there"}))
	fmt.Printf("POST /echo -> %d\n", echoResp.StatusCode)
	fmt.Printf("  body=%s\n", echoResp.Text())

	missing := client.Get("/does-not-exist")
	fmt.Printf("GET /does-not-exist -> %d\n", missing.StatusCode)
}
