package rivex_test

import (
	"testing"

	rivex "github.com/shregar1/rivex-go"
)

func TestParseVersion(t *testing.T) {
	v, err := rivex.ParseVersion("1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if v.Major != 1 || v.Minor != 2 || v.Patch != 3 {
		t.Errorf("got %+v", v)
	}
}

func TestParseVersionWithLabel(t *testing.T) {
	v, err := rivex.ParseVersion("2.0.0-beta")
	if err != nil {
		t.Fatal(err)
	}
	if v.Label != "-beta" {
		t.Errorf("got %+v", v)
	}
}

func TestVersionString(t *testing.T) {
	v := &rivex.Version{Major: 1, Minor: 2, Patch: 3}
	if v.String() != "1.2.3" {
		t.Errorf("got %s", v.String())
	}
	v = &rivex.Version{Major: 1}
	if v.String() != "1" {
		t.Errorf("got %s", v.String())
	}
}

func TestVersionCompare(t *testing.T) {
	a := &rivex.Version{Major: 1, Minor: 2, Patch: 3}
	b := &rivex.Version{Major: 1, Minor: 2, Patch: 3}
	if a.Compare(b) != 0 {
		t.Errorf("expected 0 for equal versions")
	}

	c := &rivex.Version{Major: 1, Minor: 3, Patch: 0}
	if a.Compare(c) >= 0 {
		t.Errorf("expected a < c, got %d", a.Compare(c))
	}
	if c.Compare(a) <= 0 {
		t.Errorf("expected c > a, got %d", c.Compare(a))
	}
}

func TestVersionSatisfies(t *testing.T) {
	v := &rivex.Version{Major: 1, Minor: 2, Patch: 3}
	cases := []struct {
		constraint string
		expected   bool
	}{
		{"*", true},
		{"", true},
		{"1.2.3", true},
		{"1.2.4", false},
		{">=1.0.0", true},
		{">=2.0.0", false},
		{"^1.0.0", true},
		{"^2.0.0", false},
		{"~1.2.0", true},
		{"~1.3.0", false},
	}
	for _, c := range cases {
		t.Run(c.constraint, func(t *testing.T) {
			got := v.Satisfies(c.constraint)
			if got != c.expected {
				t.Errorf("Satisfies(%q)=%v, want %v", c.constraint, got, c.expected)
			}
		})
	}
}

func TestVersioningMiddlewareHeader(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewVersionMiddleware())
	app.Get("/api", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"version": rivex.GetVersion(ctx)})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/api", rivex.WithHeader("X-API-Version", "2.1.0"))
	var body map[string]string
	resp.JSON(&body)
	if body["version"] != "2.1.0" {
		t.Errorf("expected 2.1.0, got %s", body["version"])
	}
}

func TestVersioningMiddlewareDefault(t *testing.T) {
	app := rivex.New()
	app.Use(rivex.NewVersionMiddleware(
		rivex.WithDefaultVersion("1.0.0"),
		rivex.WithVersionStrict(false),
	))
	app.Get("/api", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"version": rivex.GetVersion(ctx)})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/api")
	var body map[string]string
	resp.JSON(&body)
	if body["version"] != "1.0.0" {
		t.Errorf("expected default 1.0.0, got %s", body["version"])
	}
}

func TestVersionNegotiator(t *testing.T) {
	n := rivex.NewVersionNegotiator([]string{"1.0.0", "2.0.0", "v3"}, "1.0.0")
	if n == nil {
		t.Fatal("expected negotiator")
	}
}
