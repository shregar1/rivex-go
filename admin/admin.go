package admin

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

//go:embed static/index.html
var spaHTML string

//go:embed static
var staticFS embed.FS

var (
	startTime = time.Now()
	mu        sync.RWMutex
	app       = AppInfo{
		Name:    "Rivex",
		Version: "0.2.5",
	}
	partials   = map[string]PartialHandler{}
	actions    = map[string]actionMeta{}
	requests   int64
	errors4xx  int64
	errors5xx  int64
)

type AppInfo struct {
	Name    string
	Version string
	Env     string
	Host    string
	Port    int
}

type PartialHandler func() interface{}
type ActionHandler func() (interface{}, error)

func SetAppInfo(info AppInfo) {
	mu.Lock()
	app = info
	mu.Unlock()
}

func RegisterPartial(name string, handler PartialHandler) {
	mu.Lock()
	partials[name] = handler
	mu.Unlock()
}

func RegisterAction(name, title, description string, dangerous bool, handler ActionHandler) {
	mu.Lock()
	actions[name] = actionMeta{
		Title:       title,
		Description: description,
		Dangerous:   dangerous,
		handler:     handler,
	}
	mu.Unlock()
}

func IncRequests()     { mu.Lock(); requests++; mu.Unlock() }
func Inc4xx()          { mu.Lock(); errors4xx++; mu.Unlock() }
func Inc5xx()          { mu.Lock(); errors5xx++; mu.Unlock() }

type actionMeta struct {
	Title       string
	Description string
	Dangerous   bool
	handler     ActionHandler
}

func RegisterRoutes(mux interface {
	Handle(pattern string, handler http.Handler)
}, adminToken string) {
	staticSub, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(staticSub))

	auth := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Admin-Token")
			if token == "" {
				if c, _ := r.Cookie("rivex_admin_token"); c != nil {
					token = c.Value
				}
			}
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token != "" && adminToken != "" && token == adminToken {
				next(w, r)
				return
			}
			if adminToken == "" {
				next(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"detail": "unauthorized"})
		}
	}

	mux.Handle("/admin/static/", http.StripPrefix("/admin/static/", fileServer))

	mux.Handle("/admin", auth(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/admin" && r.URL.Path != "/admin/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(spaHTML))
	}))

	noCache := func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Cache-Control", "no-store")
			next(w, r)
		}
	}

	jsonResp := func(w http.ResponseWriter, data interface{}) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}

	mux.Handle("/admin/api/meta", auth(noCache(func(w http.ResponseWriter, r *http.Request) {
		mu.RLock()
		pKeys := make([]string, 0, len(partials))
		for k := range partials {
			pKeys = append(pKeys, k)
		}
		aList := make([]map[string]interface{}, 0)
		for name, a := range actions {
			aList = append(aList, map[string]interface{}{
				"name": name, "title": a.Title,
				"description": a.Description, "dangerous": a.Dangerous,
			})
		}
		appInfo := app
		mu.RUnlock()
		sort.Strings(pKeys)
		jsonResp(w, map[string]interface{}{
			"partials": pKeys,
			"actions":  aList,
			"app_name": appInfo.Name,
			"version":  appInfo.Version,
		})
	})))

	mux.Handle("/admin/api/partials/", auth(noCache(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/admin/api/partials/")
		mu.RLock()
		h, ok := partials[name]
		mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		jsonResp(w, h())
	})))

	mux.Handle("/admin/api/actions/", auth(noCache(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/admin/api/actions/")
		mu.RLock()
		a, ok := actions[name]
		mu.RUnlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		result, err := a.handler()
		if err != nil {
			jsonResp(w, map[string]interface{}{"error": err.Error()})
			return
		}
		jsonResp(w, result)
	})))
}

func init() {
	RegisterPartial("overview", func() interface{} {
		mu.RLock()
		a := app
		mu.RUnlock()
		uptime := time.Since(startTime)
		buildInfo, _ := debug.ReadBuildInfo()
		goVersion := runtime.Version()
		if buildInfo != nil {
			goVersion = buildInfo.GoVersion
		}
		return map[string]interface{}{
			"service":     a.Name,
			"version":     a.Version,
			"environment": a.Env,
			"bind":        fmt.Sprintf("%s:%d", a.Host, a.Port),
			"go_version":  goVersion,
			"started":     startTime.Format(time.RFC3339),
			"uptime":      fmt.Sprintf("%dh%dm", int(uptime.Hours()), int(uptime.Minutes())%60),
		}
	})

	RegisterPartial("runtime", func() interface{} {
		mu.RLock()
		req := requests
		e4 := errors4xx
		e5 := errors5xx
		mu.RUnlock()
		return map[string]interface{}{
			"requests_served": req,
			"errors_4xx":      e4,
			"errors_5xx":      e5,
			"goroutines":      runtime.NumGoroutine(),
		}
	})

	RegisterPartial("config", func() interface{} {
		env := os.Environ()
		sort.Strings(env)
		redacted := map[string]bool{
			"token": true, "secret": true, "password": true, "key": true,
			"auth": true, "credential": true, "api_key": true, "jwt": true,
		}
		cfg := make([]map[string]string, 0)
		for _, e := range env {
			parts := strings.SplitN(e, "=", 2)
			k := parts[0]
			v := ""
			if len(parts) > 1 {
				v = parts[1]
			}
			kl := strings.ToLower(k)
			isSecret := false
			for rk := range redacted {
				if strings.Contains(kl, rk) {
					isSecret = true
					break
				}
			}
			if isSecret {
				v = "••••••••"
			} else if len(v) > 80 {
				v = v[:80] + "…"
			}
			cfg = append(cfg, map[string]string{"key": k, "value": v})
		}
		return map[string]interface{}{"config": cfg}
	})

	RegisterPartial("logs", func() interface{} {
		logFile := os.Getenv("RIVEX_LOG_FILE")
		if logFile == "" {
			return map[string]interface{}{"lines": []string{}, "error": "no log file configured (set RIVEX_LOG_FILE)"}
		}
		data, err := os.ReadFile(logFile)
		if err != nil {
			return map[string]interface{}{"lines": []string{}, "error": err.Error()}
		}
		all := strings.Split(string(data), "\n")
		if len(all) > 100 {
			all = all[len(all)-100:]
		}
		total := 0
		var limited []string
		for _, l := range all {
			total += len(l)
			if total > 65536 {
				break
			}
			limited = append(limited, l)
		}
		return map[string]interface{}{"lines": limited}
	})

	RegisterPartial("actions", func() interface{} {
		mu.RLock()
		list := make([]map[string]interface{}, 0)
		for name, a := range actions {
			list = append(list, map[string]interface{}{
				"name": name, "title": a.Title,
				"description": a.Description, "dangerous": a.Dangerous,
			})
		}
		mu.RUnlock()
		return map[string]interface{}{"actions": list}
	})

	RegisterPartial("setup", func() interface{} {
		return map[string]interface{}{
			"admin_path": "env-based token (RIVEX_ADMIN_TOKEN)",
			"schema":     "ready",
		}
	})

	RegisterAction("reload_config", "Reload Config", "Re-read environment variables and config files", false, func() (interface{}, error) {
		return map[string]string{"status": "config reload triggers env re-read on next request"}, nil
	})

	RegisterAction("ping", "Ping", "Check if the admin API is responsive", false, func() (interface{}, error) {
		return map[string]interface{}{"pong": true, "latency_ms": 0}, nil
	})

	RegisterAction("gc", "Run GC", "Force a garbage collection cycle", false, func() (interface{}, error) {
		runtime.GC()
		return map[string]string{"status": "gc completed"}, nil
	})
}
