package rivex_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Helpers (avoid conflict with apples_test.go)
func nethhttptestStart(handler http.Handler) *httptest.Server {
	server := httptest.NewUnstartedServer(handler)
	server.Config.ReadTimeout = 30 * time.Second
	server.Config.WriteTimeout = 30 * time.Second
	server.Start()
	return server
}

func nethhttpFastClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 200,
			MaxConnsPerHost:     200,
			DisableCompression:  true,
		},
	}
}

func nethhttpRecorder() *httptest.ResponseRecorder {
	return httptest.NewRecorder()
}

func nethhttpRequest(method, target string) *http.Request {
	return httptest.NewRequest(method, target, nil)
}

func nethhttpRequestWithBody(method, target string, body []byte) *http.Request {
	return httptest.NewRequest(method, target, makeBytesReader(body))
}

func makeBytesReader(data []byte) *bytesReaderN {
	return &bytesReaderN{data: data}
}

type bytesReaderN struct {
	data []byte
	pos  int
}

func (r *bytesReaderN) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

var _ = fmt.Sprintf

type BenchPayload struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

var benchPayloads = func() []BenchPayload {
	p := make([]BenchPayload, 100)
	for i := 0; i < 100; i++ {
		p[i] = BenchPayload{
			ID:    fmt.Sprintf("u%d", i),
			Name:  fmt.Sprintf("User %d", i),
			Email: fmt.Sprintf("user%d@example.com", i),
			Age:   20 + i%50,
		}
	}
	return p
}()

func benchJSONListHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(benchPayloads)
}

func benchTextHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte("Hello, World! This is a plain text response."))
}

// Test at the SAME setup as Rivex benchmarks (TestClient in-process equivalent)
// For net/http, this means testing the handler function directly.

func BenchmarkNetHTTP_JSONList_DirectCall(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := nethhttpRecorder()
			req := nethhttpRequest("GET", "/users")
			benchJSONListHandler(rec, req)
		}
	})
}

func BenchmarkNetHTTP_Text_DirectCall(b *testing.B) {
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := nethhttpRecorder()
			req := nethhttpRequest("GET", "/text")
			benchTextHandler(rec, req)
		}
	})
}

func BenchmarkNetHTTP_JSONCreate_DirectCall(b *testing.B) {
	body, _ := json.Marshal(BenchPayload{Name: "Alice", Email: "alice@example.com", Age: 30})
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := nethhttpRecorder()
			req := nethhttpRequestWithBody("POST", "/users", body)
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var p BenchPayload
				json.NewDecoder(r.Body).Decode(&p)
				p.ID = "new-user"
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(201)
				json.NewEncoder(w).Encode(p)
			})
			handler.ServeHTTP(rec, req)
		}
	})
}

// Test with httptest (real server, network roundtrip)
func BenchmarkNetHTTP_JSONList_HTTP(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", benchJSONListHandler)
	server := nethhttptestStart(mux)
	defer server.Close()
	client := nethhttpFastClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(server.URL + "/users")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}

func BenchmarkNetHTTP_Text_HTTP(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/text", benchTextHandler)
	server := nethhttptestStart(mux)
	defer server.Close()
	client := nethhttpFastClient()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(server.URL + "/text")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}

// Test with 8 cores to see scaling
func BenchmarkNetHTTP_JSONList_HTTP_8Core(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", benchJSONListHandler)
	server := nethhttptestStart(mux)
	defer server.Close()
	client := nethhttpFastClient()
	b.ResetTimer()
	b.SetParallelism(8)
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			resp, _ := client.Get(server.URL + "/users")
			if resp != nil {
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		}
	})
}

// High-concurrency test
func BenchmarkNetHTTP_JSONList_Concurrent(b *testing.B) {
	mux := http.NewServeMux()
	mux.HandleFunc("/users", benchJSONListHandler)
	server := nethhttptestStart(mux)
	defer server.Close()
	clients := make([]*http.Client, 32)
	for i := range clients {
		clients[i] = nethhttpFastClient()
	}
	var total int64
	b.ResetTimer()
	var wg sync.WaitGroup
	for w := 0; w < 32; w++ {
		wg.Add(1)
		go func(c *http.Client) {
			defer wg.Done()
			for i := 0; i < b.N; i++ {
				resp, _ := c.Get(server.URL + "/users")
				if resp != nil {
					io.Copy(io.Discard, resp.Body)
					resp.Body.Close()
				}
				atomic.AddInt64(&total, 1)
			}
		}(clients[w])
	}
	wg.Wait()
}

// Compare with a more optimized handler (no context, minimal work)
func BenchmarkNetHTTP_Minimal_HTTP(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"id":"u1","name":"Alice","email":"a@b.com","age":30}`))
	})
	server := nethhttptestStart(handler)
	defer server.Close()
	client := nethhttpFastClient()
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

// Static response (no marshaling) - upper bound
func BenchmarkNetHTTP_Static_HTTP(b *testing.B) {
	staticBody := []byte(`{"id":"u1","name":"Alice","email":"alice@example.com","age":30}`)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(staticBody)
	})
	server := nethhttptestStart(handler)
	defer server.Close()
	client := nethhttpFastClient()
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
