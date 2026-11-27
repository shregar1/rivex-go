package rivex

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"strconv"
	"strings"
)

type Page[T any] struct {
	Items      []T    `json:"items"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
	TotalPages int    `json:"total_pages"`
	HasNext    bool   `json:"has_next"`
	HasPrev    bool   `json:"has_prev"`
}

type PageParams struct {
	Page     int
	PageSize int
}

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 1000
)

func ParsePageParams(ctx *Context) PageParams {
	page := defaultPage
	pageSize := defaultPageSize
	if p := ctx.QueryParam("page"); p != "" {
		if v, err := strconv.Atoi(p); err == nil && v > 0 {
			page = v
		}
	}
	if ps := ctx.QueryParam("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	if ps2 := ctx.QueryParam("limit"); ps2 != "" {
		if v, err := strconv.Atoi(ps2); err == nil && v > 0 {
			pageSize = v
		}
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return PageParams{Page: page, PageSize: pageSize}
}

func NewPage[T any](items []T, total int, params PageParams) *Page[T] {
	totalPages := int(math.Ceil(float64(total) / float64(params.PageSize)))
	if totalPages < 1 {
		totalPages = 1
	}
	return &Page[T]{
		Items:      items,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
		HasNext:    params.Page < totalPages,
		HasPrev:    params.Page > 1,
	}
}

func (p *Page[T]) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *Page[T]) Limit() int {
	return p.PageSize
}

func (p *Page[T]) ToResponse() *Response {
	return JSONResponse(p)
}

type CursorPage[T any] struct {
	Items      []T    `json:"items"`
	Cursor     string `json:"cursor,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
	PageSize   int    `json:"page_size"`
}

type CursorParams struct {
	Cursor   string
	PageSize int
}

func ParseCursorParams(ctx *Context) CursorParams {
	cursor := ctx.QueryParam("cursor")
	pageSize := defaultPageSize
	if ps := ctx.QueryParam("page_size"); ps != "" {
		if v, err := strconv.Atoi(ps); err == nil && v > 0 {
			pageSize = v
		}
	}
	if ps2 := ctx.QueryParam("limit"); ps2 != "" {
		if v, err := strconv.Atoi(ps2); err == nil && v > 0 {
			pageSize = v
		}
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return CursorParams{Cursor: cursor, PageSize: pageSize}
}

func NewCursorPage[T any](items []T, nextCursor string, params CursorParams) *CursorPage[T] {
	hasMore := nextCursor != ""
	if params.Cursor == "" && len(items) < params.PageSize {
		hasMore = false
	}
	return &CursorPage[T]{
		Items:      items,
		NextCursor: nextCursor,
		HasMore:    hasMore,
		PageSize:   params.PageSize,
	}
}

func (c *CursorPage[T]) ToResponse() *Response {
	return JSONResponse(c)
}

func EncodeCursor(value string) string {
	return base64.URLEncoding.EncodeToString([]byte(value))
}

func DecodeCursor(cursor string) (string, error) {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func EncodeCursorJSON(v interface{}) (string, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(data), nil
}

func DecodeCursorJSON(cursor string, v interface{}) error {
	data, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}

type Paginator interface {
	Paginate(ctx *Context) (interface{}, error)
}

type OffsetPaginator[T any] struct {
	Fetch func(page, pageSize int) ([]T, int, error)
}

func NewOffsetPaginator[T any](fetch func(page, pageSize int) ([]T, int, error)) *OffsetPaginator[T] {
	return &OffsetPaginator[T]{Fetch: fetch}
}

func (p *OffsetPaginator[T]) Paginate(ctx *Context) (*Page[T], error) {
	params := ParsePageParams(ctx)
	items, total, err := p.Fetch(params.Page, params.PageSize)
	if err != nil {
		return nil, err
	}
	return NewPage(items, total, params), nil
}

type CursorPaginator[T any] struct {
	Fetch func(cursor string, pageSize int) ([]T, string, error)
}

func NewCursorPaginator[T any](fetch func(cursor string, pageSize int) ([]T, string, error)) *CursorPaginator[T] {
	return &CursorPaginator[T]{Fetch: fetch}
}

func (p *CursorPaginator[T]) Paginate(ctx *Context) (*CursorPage[T], error) {
	params := ParseCursorParams(ctx)
	items, nextCursor, err := p.Fetch(params.Cursor, params.PageSize)
	if err != nil {
		return nil, err
	}
	return NewCursorPage(items, nextCursor, params), nil
}

type PaginatedResponse struct {
	Items      json.RawMessage `json:"items"`
	Total      int             `json:"total,omitempty"`
	Page       int             `json:"page,omitempty"`
	PageSize   int             `json:"page_size"`
	TotalPages int             `json:"total_pages,omitempty"`
	HasNext    bool            `json:"has_next"`
	HasPrev    bool            `json:"has_prev"`
	NextCursor string          `json:"next_cursor,omitempty"`
	Cursor     string          `json:"cursor,omitempty"`
	HasMore    bool            `json:"has_more,omitempty"`
}

type PaginationType string

const (
	PaginationOffset PaginationType = "offset"
	PaginationCursor PaginationType = "cursor"
)

type PaginationConfig struct {
	Type     PaginationType
	Default  int
	MaxSize  int
}

type PaginationMiddleware struct {
	config PaginationConfig
}

func NewPaginationMiddleware(config ...PaginationConfig) *PaginationMiddleware {
	cfg := PaginationConfig{
		Type:    PaginationOffset,
		Default: defaultPageSize,
		MaxSize: maxPageSize,
	}
	if len(config) > 0 {
		if config[0].Default > 0 {
			cfg.Default = config[0].Default
		}
		if config[0].MaxSize > 0 {
			cfg.MaxSize = config[0].MaxSize
		}
		if config[0].Type != "" {
			cfg.Type = config[0].Type
		}
	}
	return &PaginationMiddleware{config: cfg}
}

func (m *PaginationMiddleware) Process(ctx *Context, next NextFunc) error {
	ctx.Set("pagination.config", m.config)
	return next(ctx)
}

func GetPaginationConfig(ctx *Context) PaginationConfig {
	v, ok := ctx.Get("pagination.config")
	if !ok {
		return PaginationConfig{Type: PaginationOffset, Default: defaultPageSize, MaxSize: maxPageSize}
	}
	cfg, _ := v.(PaginationConfig)
	return cfg
}

func Paginate[T any](ctx *Context, fetch func(page, pageSize int) ([]T, int, error)) (*Page[T], error) {
	params := ParsePageParams(ctx)
	items, total, err := fetch(params.Page, params.PageSize)
	if err != nil {
		return nil, err
	}
	return NewPage(items, total, params), nil
}

func CursorPaginate[T any](ctx *Context, fetch func(cursor string, pageSize int) ([]T, string, error)) (*CursorPage[T], error) {
	params := ParseCursorParams(ctx)
	items, nextCursor, err := fetch(params.Cursor, params.PageSize)
	if err != nil {
		return nil, err
	}
	return NewCursorPage(items, nextCursor, params), nil
}

func PageToMap(page interface{}) map[string]interface{} {
	data, err := json.Marshal(page)
	if err != nil {
		return map[string]interface{}{}
	}
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	return result
}

func MergePaginationParams(ctx *Context, pageParams PageParams) PageParams {
	if pageParams.Page <= 0 {
		pageParams.Page = defaultPage
	}
	if pageParams.PageSize <= 0 {
		pageParams.PageSize = defaultPageSize
	}
	if pageParams.PageSize > maxPageSize {
		pageParams.PageSize = maxPageSize
	}
	return pageParams
}

func SortParam(ctx *Context, allowedFields ...string) (string, bool) {
	sort := ctx.QueryParam("sort")
	if sort == "" {
		return "", false
	}
	desc := false
	if strings.HasPrefix(sort, "-") {
		desc = true
		sort = sort[1:]
	}
	if len(allowedFields) > 0 {
		found := false
		for _, f := range allowedFields {
			if f == sort {
				found = true
				break
			}
		}
		if !found {
			return "", false
		}
	}
	return sort, desc
}
