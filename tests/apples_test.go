package rivex_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

)

// Benchmark setup:
//   - Same handler logic (parse request, build response, marshal JSON)
//   - Same payload (100 users)
//   - Same routing (single path)
//   - Same connection pool settings
//   - Compared: Rivex StreamingHTTP vs net/http ServeMux vs gin/Echo equivalent

type ApplesUser struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

var applesUsers = func() []ApplesUser {
	users := make([]ApplesUser, 100)
	for i := 0; i < 100; i++ {
		users[i] = ApplesUser{
			ID:    fmt.Sprintf("u%d", i),
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i%50,
		}
	}
	return users
}()

// User list — returns JSON of 100 users (same payload).
// Both Rivex and net/http use IDENTICAL handler bodies.

func listUsersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(applesUsers)
	w.Write(data)
}

func getUserHandler(w http.ResponseWriter, r *http.Request) {
	// Match Rivex's path scheme: /user/{id} (singular) to avoid stdlib mux
	// conflicts with /users (plural) prefix.
	id := strings.TrimPrefix(r.URL.Path, "/user/")
	w.Header().Set("Content-Type", "application/json")
	for _, u := range applesUsers {
		if u.ID == id {
			data, _ := json.Marshal(u)
			w.Write(data)
			return
		}
	}
	w.WriteHeader(404)
	w.Write([]byte(`{"detail":"not found"}`))
}

func textHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Hello, World! This is a plain text response for apples-to-apples comparison."))
}

func createUserHandler(w http.ResponseWriter, r *http.Request) {
	var u ApplesUser
	if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
		w.WriteHeader(400)
		return
	}
	u.ID = "new-user"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(201)
	data, _ := json.Marshal(u)
	w.Write(data)
}

// Use /create_user to avoid stdlib mux conflict with GET /users.
// Rivex test uses the same path for fair comparison.

// ---------- Setup helpers ----------

type applesServer struct {
	addr    string
	closeFn func()
}

func startNetHTTPServer(handler http.Handler) *applesServer {
	server := httptest.NewUnstartedServer(handler)
	server.Config.ReadTimeout = 30 * time.Second
	server.Config.WriteTimeout = 30 * time.Second
	server.Start()
	return &applesServer{
		addr:    server.URL,
		closeFn: server.Close,
	}
}


// ---------- Benchmarks ----------

func BenchmarkApples_NetHTTP_JSON100(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", listUsersHandler)
	srv := startNetHTTPServer(mux)
	defer srv.closeFn()
	client := newFastHTTPClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, err := client.Get(srv.addr + "/users")
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}


func BenchmarkApples_NetHTTP_JSONSingle(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/", getUserHandler)
	srv := startNetHTTPServer(mux)
	defer srv.closeFn()
	client := newFastHTTPClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(srv.addr + "/user/u42")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}


func BenchmarkApples_NetHTTP_JSONCreate(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/create_user", createUserHandler)
	srv := startNetHTTPServer(mux)
	defer srv.closeFn()
	client := newFastHTTPClient()
	body := mustMarshal(ApplesUser{Name: "Alice", Email: "alice@example.com", Age: 30})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Post(srv.addr+"/create_user", "application/json", readerOf(body))
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}


func BenchmarkApples_NetHTTP_Text(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/text", textHandler)
	srv := startNetHTTPServer(mux)
	defer srv.closeFn()
	client := newFastHTTPClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(srv.addr + "/text")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}


func BenchmarkApples_NetHTTP_404(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/user/", getUserHandler)
	srv := startNetHTTPServer(mux)
	defer srv.closeFn()
	client := newFastHTTPClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(srv.addr + "/user/nonexistent")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}


func BenchmarkApples_NetHTTP_Parallel10(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", listUsersHandler)
	mux.HandleFunc("/text", textHandler)
	srv := startNetHTTPServer(mux)
	defer srv.closeFn()
	clients := make([]*http.Client, 16)
	for i := range clients {
		clients[i] = newFastHTTPClient()
	}
	b.ResetTimer()
	var wg sync.WaitGroup
	for w := 0; w < 16; w++ {
		wg.Add(1)
		go func(c *http.Client) {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				resp, _ := c.Get(srv.addr + "/users")
				if resp != nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
			}
		}(clients[w])
	}
	wg.Wait()
}


// ---------- Helpers ----------

func newFastHTTPClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
			DisableCompression:  true,
		},
	}
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}

func readerOf(data []byte) io.Reader {
	return &bytesReader{data: data}
}

type bytesReader struct {
	data []byte
	pos  int
}

func (r *bytesReader) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
