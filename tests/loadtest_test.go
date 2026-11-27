package rivex_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

var errEOF = io.EOF
var _ = url.Error{}
var _ = errors.New

// LoadTestResult holds the outcome of a single load test run.
type LoadTestResult struct {
	URL         string        `json:"url"`
	Concurrency int           `json:"concurrency"`
	Duration    time.Duration `json:"duration_ns"`
	Requests    int64         `json:"requests"`
	Bytes       int64         `json:"bytes"`
	Errors      int64         `json:"errors"`
	Status2xx   int64         `json:"status_2xx"`
	Status4xx   int64         `json:"status_4xx"`
	Status5xx   int64         `json:"status_5xx"`
	Timeouts    int64         `json:"timeouts"`
	RPS         float64       `json:"rps"`
	Throughput  float64       `json:"throughput_mb_s"`
	P50         time.Duration `json:"p50_ns"`
	P90         time.Duration `json:"p90_ns"`
	P99         time.Duration `json:"p99_ns"`
	P999        time.Duration `json:"p999_ns"`
	Max         time.Duration `json:"max_ns"`
	Mean        time.Duration `json:"mean_ns"`
}

// LoadTester is a minimal in-process load tester (similar to wrk/hey).
// It spawns N workers, each sending requests continuously for the given
// duration. Latencies are recorded with high precision and percentiles
// are computed at the end.
type LoadTester struct {
	URL         string
	Method      string
	Body        []byte
	Headers     map[string]string
	Concurrency int
	Duration    time.Duration
	Timeout     time.Duration
	Client      *http.Client
}

// NewLoadTester creates a load tester with sensible defaults.
func NewLoadTester(url string, concurrency int, duration time.Duration) *LoadTester {
	return &LoadTester{
		URL:         url,
		Method:      "GET",
		Concurrency: concurrency,
		Duration:    duration,
		Timeout:     30 * time.Second,
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        concurrency * 4,
				MaxIdleConnsPerHost: concurrency * 4,
				MaxConnsPerHost:     concurrency * 4,
				DisableCompression:  true,
			},
		},
	}
}

func (lt *LoadTester) WithMethod(method string, body []byte) *LoadTester {
	lt.Method = method
	lt.Body = body
	return lt
}

func (lt *LoadTester) WithHeader(k, v string) *LoadTester {
	if lt.Headers == nil {
		lt.Headers = make(map[string]string)
	}
	lt.Headers[k] = v
	return lt
}

// Run executes the load test and returns a result with percentiles.
func (lt *LoadTester) Run() LoadTestResult {
	var (
		requests  int64
		bytes     int64
		errors    int64
		status2xx int64
		status4xx int64
		status5xx int64
		timeouts  int64
	)

	deadline := time.Now().Add(lt.Duration)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()

	// Each worker writes its latencies to its own slice to avoid contention.
	// We aggregate after.
	workerLatencies := make([][]time.Duration, lt.Concurrency)
	for i := range workerLatencies {
		workerLatencies[i] = make([]time.Duration, 0, 1024)
	}

	var wg sync.WaitGroup
	for w := 0; w < lt.Concurrency; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			workerLatencies[workerID] = workerLatencies[workerID][:0]
			for {
				select {
				case <-ctx.Done():
					return
				default:
				}
				start := time.Now()
				req, err := http.NewRequestWithContext(ctx, lt.Method, lt.URL, nil)
				if err != nil {
					atomic.AddInt64(&errors, 1)
					continue
				}
				for k, v := range lt.Headers {
					req.Header.Set(k, v)
				}
				if lt.Body != nil {
					req.Body = &byteReaderN{data: lt.Body}
					req.ContentLength = int64(len(lt.Body))
				}
				resp, err := lt.Client.Do(req)
				elapsed := time.Since(start)
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					if isTimeout(err) {
						atomic.AddInt64(&timeouts, 1)
					}
					atomic.AddInt64(&errors, 1)
					continue
				}
				atomic.AddInt64(&requests, 1)
				n, _ := readAndDiscard(resp)
				atomic.AddInt64(&bytes, int64(n))
				switch {
				case resp.StatusCode >= 200 && resp.StatusCode < 300:
					atomic.AddInt64(&status2xx, 1)
				case resp.StatusCode >= 400 && resp.StatusCode < 500:
					atomic.AddInt64(&status4xx, 1)
				case resp.StatusCode >= 500:
					atomic.AddInt64(&status5xx, 1)
				}
				resp.Body.Close()
				// Append in-place: keep writing into the worker slot.
				workerLatencies[workerID] = append(workerLatencies[workerID], elapsed)
			}
		}(w)
	}
	wg.Wait()

	// Aggregate latencies and compute percentiles.
	var allLatencies []time.Duration
	for _, lats := range workerLatencies {
		allLatencies = append(allLatencies, lats...)
	}
	sort.Slice(allLatencies, func(i, j int) bool { return allLatencies[i] < allLatencies[j] })

	var sum time.Duration
	for _, l := range allLatencies {
		sum += l
	}
	mean := time.Duration(0)
	if len(allLatencies) > 0 {
		mean = sum / time.Duration(len(allLatencies))
	}

	result := LoadTestResult{
		URL:         lt.URL,
		Concurrency: lt.Concurrency,
		Duration:    lt.Duration,
		Requests:    requests,
		Bytes:       bytes,
		Errors:      errors,
		Status2xx:   status2xx,
		Status4xx:   status4xx,
		Status5xx:   status5xx,
		Timeouts:    timeouts,
		RPS:         float64(requests) / lt.Duration.Seconds(),
		Throughput:  float64(bytes) / lt.Duration.Seconds() / (1024 * 1024),
		Mean:        mean,
	}
	if n := len(allLatencies); n > 0 {
		result.P50 = allLatencies[n*50/100]
		result.P90 = allLatencies[n*90/100]
		result.P99 = allLatencies[n*99/100]
		result.P999 = allLatencies[n*999/1000]
		result.Max = allLatencies[n-1]
	}
	return result
}

func (r LoadTestResult) String() string {
	return fmt.Sprintf(
		"RPS=%-8.0f | p50=%-8s p90=%-8s p99=%-8s p99.9=%-8s | err=%d | %d reqs in %v",
		r.RPS,
		r.P50.Round(time.Microsecond),
		r.P90.Round(time.Microsecond),
		r.P99.Round(time.Microsecond),
		r.P999.Round(time.Microsecond),
		r.Errors,
		r.Requests,
		r.Duration.Round(time.Millisecond),
	)
}

func isTimeout(err error) bool {
	if err == nil {
		return false
	}
	if t, ok := err.(interface{ Timeout() bool }); ok {
		return t.Timeout()
	}
	return false
}

type byteReaderN struct {
	data []byte
	pos  int
}

func (r *byteReaderN) Read(p []byte) (int, error) {
	if r.pos >= len(r.data) {
		return 0, errEOF
	}
	n := copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}

func (r *byteReaderN) Close() error { return nil }

func readAndDiscard(resp *http.Response) (int, error) {
	defer resp.Body.Close()
	buf := make([]byte, 32*1024)
	total := 0
	for {
		n, err := resp.Body.Read(buf)
		total += n
		if err != nil {
			return total, nil
		}
	}
}
