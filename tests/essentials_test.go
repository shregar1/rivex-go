package rivex_test

import (
	"net/url"
	"reflect"
	"testing"
	"time"

	rivex "github.com/shregar1/rivex-go"
)

func contains(data []byte, s string) bool {
	for i := 0; i+len(s) <= len(data); i++ {
		if string(data[i:i+len(s)]) == s {
			return true
		}
	}
	return false
}


func urlValues(kv ...string) url.Values {
	v := url.Values{}
	for i := 0; i+1 < len(kv); i += 2 {
		v.Set(kv[i], kv[i+1])
	}
	return v
}














type docImpl struct {
	ownerID string
}

func (d *docImpl) GetOwnerID() string {
	return d.ownerID
}


func TestParseThrottleRate(t *testing.T) {
	rate, err := rivex.ParseThrottleRate("100/hour")
	if err != nil {
		t.Fatal(err)
	}
	if rate.Count != 100 || rate.Period != time.Hour {
		t.Errorf("got %+v", rate)
	}
}

func TestParseThrottleRateMinute(t *testing.T) {
	rate, err := rivex.ParseThrottleRate("10/min")
	if err != nil {
		t.Fatal(err)
	}
	if rate.Count != 10 || rate.Period != time.Minute {
		t.Errorf("got %+v", rate)
	}
}

func TestThrottleAllow(t *testing.T) {
	rate := rivex.NewThrottleRate(3, time.Second)
	throttle := rivex.NewThrottle(rate)
	for i := 0; i < 3; i++ {
		allowed, _ := throttle.Allow("key1")
		if !allowed {
			t.Errorf("expected allowed on attempt %d", i)
		}
	}
	allowed, _ := throttle.Allow("key1")
	if allowed {
		t.Error("expected throttled")
	}
}

func TestThrottlingMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewThrottlingMiddleware(rivex.ThrottleAnon("2/sec")))
	app.Get("/test", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"ok": "true"})
	})

	client := rivex.NewTestClient(app)
	for i := 0; i < 2; i++ {
		resp := client.Get("/test")
		if resp.StatusCode != 200 {
			t.Errorf("attempt %d: expected 200, got %d", i, resp.StatusCode)
		}
	}
	resp := client.Get("/test")
	if resp.StatusCode != 429 {
		t.Errorf("expected 429, got %d", resp.StatusCode)
	}
}

func TestSlidingWindowRateLimiter(t *testing.T) {
	limiter := rivex.NewSlidingWindowRateLimiter(3, time.Second)
	for i := 0; i < 3; i++ {
		if !limiter.Allow("key") {
			t.Errorf("attempt %d: expected allowed", i)
		}
	}
	if limiter.Allow("key") {
		t.Error("expected denied")
	}
}

func TestTokenBucketLimiter(t *testing.T) {
	limiter := rivex.NewTokenBucketLimiter(5, 1, time.Millisecond*100)
	for i := 0; i < 5; i++ {
		if !limiter.Allow("key") {
			t.Errorf("attempt %d: expected allowed", i)
		}
	}
	if limiter.Allow("key") {
		t.Error("expected denied (bucket empty)")
	}
}

func TestResponseCacheMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewResponseCacheMiddleware(rivex.WithResponseCacheTTL(time.Minute)))
	app.Get("/cached", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"hello": "world"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/cached")
	if resp.Headers["ETag"] == "" {
		t.Error("expected ETag header")
	}
	etag := resp.Headers["ETag"]
	resp = client.Get("/cached", rivex.WithHeader("If-None-Match", etag))
	if resp.StatusCode != 304 {
		t.Errorf("expected 304, got %d", resp.StatusCode)
	}
}

func TestETagMiddleware(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewETagMiddleware())
	app.Get("/test", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"hello": "world"})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/test")
	etag := resp.Headers["ETag"]
	if etag == "" {
		t.Fatal("expected ETag")
	}
	resp = client.Get("/test", rivex.WithHeader("If-None-Match", etag))
	if resp.StatusCode != 304 {
		t.Errorf("expected 304, got %d", resp.StatusCode)
	}
}










func TestGenerateSchemaFromType(t *testing.T) {
	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}
	components := make(map[string]interface{})
	schema := rivex.GenerateSchemaFromType(reflect.TypeOf(User{}), components)
	if schema == nil {
		t.Fatal("expected schema")
	}
	schemas, ok := components["schemas"].(map[string]interface{})
	if !ok {
		t.Fatal("expected components.schemas")
	}
	if _, ok := schemas["User"]; !ok {
		t.Error("expected User schema in components")
	}
}

func TestGenerateOpenAPIRequestBody(t *testing.T) {
	type Post struct {
		Title string `json:"title"`
	}
	components := make(map[string]interface{})
	req := rivex.GenerateOpenAPIRequestBody(Post{}, true, components)
	if req.Required != true {
		t.Error("expected required=true")
	}
	if req.ContentType != "application/json" {
		t.Errorf("got %s", req.ContentType)
	}
	if components["schemas"] == nil {
		t.Error("expected components populated")
	}
}

func TestBuildOperationWithRequestResponse(t *testing.T) {
	type Input struct {
		Name string `json:"name"`
	}
	type Output struct {
		ID string `json:"id"`
	}
	op := &rivex.OpenAPIOperation{
		Summary:     "Create user",
		Description: "Creates a user",
		Tags:        []string{"users"},
		RequestBody: &rivex.OpenAPIRequest{Body: Input{}, Required: true},
		Responses: map[string]*rivex.OpenAPIResponse{
			"201": {Body: Output{}, Description: "Created"},
		},
	}
	components := make(map[string]interface{})
	operation := rivex.BuildOperationWithRequestResponse("POST", "/users", op, components)
	if operation["summary"] != "Create user" {
		t.Error("expected summary")
	}
	if components["schemas"] == nil {
		t.Error("expected components populated")
	}
}

func TestAppendOpenAPIPath(t *testing.T) {
	spec := map[string]interface{}{}
	operation := map[string]interface{}{"summary": "Test"}
	rivex.AppendOpenAPIPath(spec, "GET", "/test", operation)
	paths := spec["paths"].(map[string]interface{})
	pathMethods := paths["/test"].(map[string]interface{})
	if pathMethods["get"] == nil {
		t.Error("expected GET operation")
	}
}

