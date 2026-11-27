package rivex_test

import (
	"net/http"
	"testing"

	rivex "github.com/shregar1/rivex-go"
)

func TestPageBasic(t *testing.T) {
	items := []string{"a", "b", "c"}
	page := rivex.NewPage[string](items, 10, rivex.PageParams{Page: 1, PageSize: 3})
	if page.Total != 10 {
		t.Errorf("expected Total=10, got %d", page.Total)
	}
	if page.Page != 1 {
		t.Errorf("expected Page=1, got %d", page.Page)
	}
	if page.PageSize != 3 {
		t.Errorf("expected PageSize=3, got %d", page.PageSize)
	}
	if page.TotalPages != 4 {
		t.Errorf("expected TotalPages=4, got %d", page.TotalPages)
	}
	if !page.HasNext {
		t.Error("expected HasNext=true on page 1")
	}
	if page.HasPrev {
		t.Error("expected HasPrev=false on page 1")
	}
}

func TestPageLastPage(t *testing.T) {
	page := rivex.NewPage[string](nil, 10, rivex.PageParams{Page: 4, PageSize: 3})
	if page.HasNext {
		t.Error("expected HasNext=false on last page")
	}
	if !page.HasPrev {
		t.Error("expected HasPrev=true on last page")
	}
}

func TestPageOffset(t *testing.T) {
	page := rivex.NewPage[string](nil, 0, rivex.PageParams{Page: 3, PageSize: 20})
	if page.Offset() != 40 {
		t.Errorf("expected Offset=40, got %d", page.Offset())
	}
	if page.Limit() != 20 {
		t.Errorf("expected Limit=20, got %d", page.Limit())
	}
}

func TestPageEmptyTotal(t *testing.T) {
	page := rivex.NewPage[string](nil, 0, rivex.PageParams{Page: 1, PageSize: 20})
	if page.TotalPages != 1 {
		t.Errorf("expected TotalPages=1 for empty result, got %d", page.TotalPages)
	}
}

func TestCursorPage(t *testing.T) {
	page := rivex.NewCursorPage[string]([]string{"a", "b"}, "next-cursor", rivex.CursorParams{PageSize: 2})
	if !page.HasMore {
		t.Error("expected HasMore=true with nextCursor")
	}
	if page.NextCursor != "next-cursor" {
		t.Errorf("expected next cursor, got %s", page.NextCursor)
	}
}

func TestCursorEncodeDecode(t *testing.T) {
	encoded := rivex.EncodeCursor("opaque-value")
	decoded, err := rivex.DecodeCursor(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded != "opaque-value" {
		t.Errorf("expected opaque-value, got %s", decoded)
	}
}

func TestCursorJSON(t *testing.T) {
	payload := map[string]interface{}{"id": 42, "name": "test"}
	encoded, err := rivex.EncodeCursorJSON(payload)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]interface{}
	if err := rivex.DecodeCursorJSON(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if int(decoded["id"].(float64)) != 42 {
		t.Errorf("expected id=42, got %v", decoded["id"])
	}
}

func TestPaginationQueryParams(t *testing.T) {
	app := rivex.New()
	app.Get("/items", func(ctx *rivex.Context) error {
		params := rivex.ParsePageParams(ctx)
		return ctx.SendJSON(map[string]int{"page": params.Page, "size": params.PageSize})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/items?page=2&page_size=50")
	var body map[string]int
	resp.JSON(&body)
	if body["page"] != 2 {
		t.Errorf("expected page=2, got %d", body["page"])
	}
	if body["size"] != 50 {
		t.Errorf("expected size=50, got %d", body["size"])
	}
}

func TestPaginationDefaultParams(t *testing.T) {
	app := rivex.New()
	app.Get("/items", func(ctx *rivex.Context) error {
		params := rivex.ParsePageParams(ctx)
		return ctx.SendJSON(map[string]int{"page": params.Page, "size": params.PageSize})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/items")
	var body map[string]int
	resp.JSON(&body)
	if body["page"] != 1 {
		t.Errorf("expected default page=1, got %d", body["page"])
	}
	if body["size"] != 20 {
		t.Errorf("expected default size=20, got %d", body["size"])
	}
}

func TestPaginationMaxSize(t *testing.T) {
	app := rivex.New()
	app.Get("/items", func(ctx *rivex.Context) error {
		params := rivex.ParsePageParams(ctx)
		return ctx.SendJSON(map[string]int{"size": params.PageSize})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/items?page_size=99999")
	var body map[string]int
	resp.JSON(&body)
	if body["size"] > 1000 {
		t.Errorf("expected size clamped, got %d", body["size"])
	}
}

func TestOffsetPaginator(t *testing.T) {
	fetch := func(page, pageSize int) ([]string, int, error) {
		start := (page - 1) * pageSize
		items := []string{}
		for i := start; i < start+pageSize && i < 50; i++ {
			items = append(items, "item")
		}
		return items, 50, nil
	}
	paginator := rivex.NewOffsetPaginator(fetch)
	app := rivex.New()
	app.Get("/items", func(ctx *rivex.Context) error {
		page, err := paginator.Paginate(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(page)
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/items?page=1&page_size=10")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	var page rivex.Page[string]
	resp.JSON(&page)
	if page.Total != 50 {
		t.Errorf("expected Total=50, got %d", page.Total)
	}
	if len(page.Items) != 10 {
		t.Errorf("expected 10 items, got %d", len(page.Items))
	}
}

func TestCursorPaginator(t *testing.T) {
	fetch := func(cursor string, pageSize int) ([]string, string, error) {
		if cursor == "end" {
			return nil, "", nil
		}
		return []string{"a", "b", "c"}, "next", nil
	}
	paginator := rivex.NewCursorPaginator(fetch)
	app := rivex.New()
	app.Get("/items", func(ctx *rivex.Context) error {
		page, err := paginator.Paginate(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(page)
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/items?cursor=start")
	var page rivex.CursorPage[string]
	resp.JSON(&page)
	if !page.HasMore {
		t.Error("expected HasMore=true")
	}
}

func TestSortParam(t *testing.T) {
	app := rivex.New()
	app.Get("/users", func(ctx *rivex.Context) error {
		field, desc := rivex.SortParam(ctx, "name", "email")
		return ctx.SendJSON(map[string]interface{}{"field": field, "desc": desc})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/users?sort=-email")
	var body map[string]interface{}
	resp.JSON(&body)
	if body["field"] != "email" {
		t.Errorf("expected field=email, got %v", body["field"])
	}
	if body["desc"] != true {
		t.Errorf("expected desc=true, got %v", body["desc"])
	}
}

func TestSortParamNotAllowed(t *testing.T) {
	app := rivex.New()
	app.Get("/users", func(ctx *rivex.Context) error {
		field, _ := rivex.SortParam(ctx, "name", "email")
		return ctx.SendJSON(map[string]string{"field": field})
	})

	client := rivex.NewTestClient(app)
	resp := client.Get("/users?sort=password")
	var body map[string]string
	resp.JSON(&body)
	if body["field"] != "" {
		t.Errorf("expected empty for disallowed sort, got %s", body["field"])
	}
}
