package rivex_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	rivex "github.com/shregar1/rivex-go"
)

type BenchUser struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Email     string   `json:"email"`
	Age       int      `json:"age"`
	Roles     []string `json:"roles"`
	Active    bool     `json:"active"`
	Score     float64  `json:"score"`
	CreatedAt string   `json:"created_at"`
	Tags      []string `json:"tags"`
}

type BenchPost struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	AuthorID string `json:"author_id"`
	Views    int    `json:"views"`
	Rating   float64 `json:"rating"`
}

var benchUsers = func() []BenchUser {
	users := make([]BenchUser, 100)
	roles := []string{"user", "admin", "editor", "viewer"}
	tags := []string{"alpha", "beta", "gamma", "delta"}
	for i := 0; i < 100; i++ {
		users[i] = BenchUser{
			ID:        fmt.Sprintf("user-%d", i),
			Name:      fmt.Sprintf("User Number %d", i),
			Email:     fmt.Sprintf("user%d@example.com", i),
			Age:       20 + i%50,
			Roles:     []string{roles[i%4], roles[(i+1)%4]},
			Active:    i%3 != 0,
			Score:     float64(i) * 1.5,
			CreatedAt: "2024-01-01T00:00:00Z",
			Tags:      []string{tags[i%4], tags[(i+2)%4]},
		}
	}
	return users
}()

var benchLongText = strings.Repeat("The quick brown fox jumps over the lazy dog. ", 50)
var benchShortText = "Hello, World! This is a plain text response from the Rivex framework benchmark."

func newBenchApp() *rivex.Rivex {
	app := rivex.New()
	app.Get("/json/users", func(ctx *rivex.Context) error {
		return ctx.SendJSON(benchUsers)
	})
	app.Get("/json/user/{id}", func(ctx *rivex.Context) error {
		id := ctx.PathParam("id")
		for _, u := range benchUsers {
			if u.ID == id {
				return ctx.SendJSON(u)
			}
		}
		return ctx.SendJSONWithStatus(map[string]string{"detail": "not found"}, 404)
	})
	app.Post("/json/users", func(ctx *rivex.Context) error {
		var u BenchUser
		if err := ctx.JSON(&u); err != nil {
			return rivex.NewHTTPException(400, "bad request")
		}
		u.ID = "new-user"
		return ctx.SendJSONWithStatus(u, 201)
	})
	app.Get("/json/post/{id}", func(ctx *rivex.Context) error {
		return ctx.SendJSON(&BenchPost{
			ID:      ctx.PathParam("id"),
			Title:   "Bench Post",
			Content: strings.Repeat("Lorem ipsum dolor sit amet. ", 20),
			Views:   42,
			Rating:  4.5,
		})
	})
	app.Get("/text", func(ctx *rivex.Context) error {
		return ctx.SendText(benchShortText)
	})
	app.Get("/text/long", func(ctx *rivex.Context) error {
		return ctx.SendText(benchLongText)
	})
	app.Get("/json/empty", func(ctx *rivex.Context) error {
		return ctx.SendJSONWithStatus(map[string]string{"status": "ok"}, 200)
	})
	app.Post("/json/post", func(ctx *rivex.Context) error {
		var p BenchPost
		if err := ctx.JSON(&p); err != nil {
			return rivex.NewHTTPException(400, "bad request")
		}
		return ctx.SendJSONWithStatus(p, 201)
	})
	app.Get("/json/dict", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]interface{}{
			"id":       "p1",
			"title":    "Title",
			"views":    100,
			"metadata": map[string]string{"key": "value"},
		})
	})
	return app
}

func BenchmarkJSONList100(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp := client.Get("/json/users")
			_ = resp.Text()
		}
	})
}

func BenchmarkJSONSingleItem(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Get("/json/user/user-42")
		}
	})
}

func BenchmarkJSONCreate(b *testing.B) {
	app := newBenchApp()
	body, _ := json.Marshal(BenchUser{Name: "Alice", Email: "alice@example.com", Age: 30})
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Post("/json/users", rivex.WithBody(body))
		}
	})
}

func BenchmarkJSONPostLarge(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Get("/json/post/post-1")
		}
	})
}

func BenchmarkJSONDict(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Get("/json/dict")
		}
	})
}

func BenchmarkJSONEmpty(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Get("/json/empty")
		}
	})
}

func BenchmarkPlainTextShort(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Get("/text")
		}
	})
}

func BenchmarkPlainTextLong(b *testing.B) {
	app := newBenchApp()
	client := rivex.NewTestClient(app)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = client.Get("/text/long")
		}
	})
}

func BenchmarkJSONMarshalOnly(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = json.Marshal(benchUsers)
		}
	})
}

func BenchmarkJSONUnmarshalOnly(b *testing.B) {
	data, _ := json.Marshal(benchUsers)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var v []BenchUser
			_ = json.Unmarshal(data, &v)
		}
	})
}

func BenchmarkJSONSingleMarshal(b *testing.B) {
	u := benchUsers[0]
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = json.Marshal(u)
		}
	})
}



func BenchmarkHTTPJSONNet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/json/users":
			writeJSONResponse(w, benchUsers)
		case "/text":
			writeTextResponse(w, benchShortText)
		case "/json/empty":
			writeJSONResponse(w, map[string]string{"status": "ok"})
		}
	}))
	defer server.Close()
	client := server.Client()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(server.URL + "/json/users")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}

func BenchmarkHTTPTextNet(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeTextResponse(w, benchShortText)
	}))
	defer server.Close()
	client := server.Client()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(server.URL + "/")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}

func writeJSONResponse(w http.ResponseWriter, v interface{}) {
	data, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}

func writeTextResponse(w http.ResponseWriter, s string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(s))
}






var (
	_ = sync.Mutex{}
	_ = context.Background
	_ = io.Discard
)
