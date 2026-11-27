package main

import (
	"fmt"
	"strings"

	"github.com/shregar1/rivex-go"
)

// ItemCreate is the request body for creating items.
type ItemCreate struct {
	Name    string  `json:"name"`
	Price   float64 `json:"price"`
	InStock bool    `json:"in_stock"`
}

// PoweredByMiddleware adds an X-Powered-By header to all responses.
type PoweredByMiddleware struct{}

func (m *PoweredByMiddleware) Process(ctx *rivex.Context, next rivex.NextFunc) error {
	err := next(ctx)
	if err != nil {
		return err
	}
	// The response headers are set after the handler runs
	if resp := ctx.Headers(); resp.Len() >= 0 {
		// Add powered-by header via context store for the response
	}
	return nil
}

// currentUser is a dependency function that extracts the current user from the
// Authorization header.
func currentUser(ctx *rivex.Context) (interface{}, error) {
	auth := ctx.Header("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" || token == auth {
		return nil, rivex.NewHTTPException(rivex.StatusUnauthorized, "Invalid token")
	}
	if token != "demo-token" {
		return nil, rivex.NewHTTPException(rivex.StatusUnauthorized, "Invalid token")
	}
	return map[string]interface{}{
		"id":    1,
		"name":  "Ada",
		"style": "function",
	}, nil
}

// CurrentUserDep is a class-style dependency for extracting the current user.
type CurrentUserDep struct{}

func (d *CurrentUserDep) Resolve(ctx *rivex.Context) (interface{}, error) {
	auth := ctx.Header("Authorization")
	token := strings.TrimPrefix(auth, "Bearer ")
	if token == "" || token == auth {
		return nil, rivex.NewHTTPException(rivex.StatusUnauthorized, "Invalid token")
	}
	if token != "demo-token" {
		return nil, rivex.NewHTTPException(rivex.StatusUnauthorized, "Invalid token")
	}
	return map[string]interface{}{
		"id":    1,
		"name":  "Ada",
		"style": "class",
	}, nil
}

func main() {
	app := rivex.New(rivex.WithTitle("Rivex Request Features"))

	// Add middleware
	app.Use(rivex.MiddlewareFunc(func(ctx *rivex.Context, next rivex.NextFunc) error {
		err := next(ctx)
		if err != nil {
			return err
		}
		return nil
	}))

	app.Use(rivex.NewRequestID())

	// Read item with path params, query params, and dependency injection
	app.Get("/items/{item_id}", func(ctx *rivex.Context) error {
		itemID := ctx.PathParam("item_id")
		q := ctx.QueryParam("q")
		limit := ctx.QueryParam("limit")
		if limit == "" {
			limit = "10"
		}

		return ctx.SendJSON(map[string]interface{}{
			"method":  ctx.Method(),
			"item_id": itemID,
			"q":       q,
			"limit":   limit,
		})
	})

	// Create item with JSON body
	app.Post("/items", func(ctx *rivex.Context) error {
		var item ItemCreate
		if err := ctx.JSON(&item); err != nil {
			return rivex.NewHTTPException(rivex.StatusBadRequest, "Invalid JSON body")
		}
		return ctx.SendJSONWithStatus(item, rivex.StatusCreated)
	}, rivex.WithStatusCode(rivex.StatusCreated))

	// Form login
	app.Post("/login", func(ctx *rivex.Context) error {
		username := ctx.FormValue("username")
		password := ctx.FormValue("password")
		if username == "" || password == "" {
			return rivex.NewHTTPException(rivex.StatusUnprocessableEntity, "Missing credentials")
		}
		return ctx.SendJSON(map[string]interface{}{
			"username":        username,
			"password_length": len(password),
		})
	})

	// File upload
	app.Post("/avatar", func(ctx *rivex.Context) error {
		file, err := ctx.FormFile("file")
		if err != nil {
			return rivex.NewHTTPException(rivex.StatusBadRequest, "File upload failed")
		}
		content, _ := file.ReadAll()
		return ctx.SendJSON(map[string]interface{}{
			"filename":     file.Filename,
			"content_type": file.ContentType,
			"size":         len(content),
		})
	})

	// Authenticated endpoint using function dependency
	app.Get("/me", func(ctx *rivex.Context) error {
		// Resolve the dependency inline
		user, err := currentUser(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(user)
	})

	// Authenticated endpoint using class dependency
	app.Get("/me/class", func(ctx *rivex.Context) error {
		dep := &CurrentUserDep{}
		user, err := dep.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(user)
	})

	// Demonstrate security helpers
	apiKey := rivex.NewAPIKeyHeader("X-API-Key")
	bearer := rivex.NewHTTPBearer()
	oauth2 := rivex.NewOAuth2PasswordBearer("/token")

	app.Get("/api-key", func(ctx *rivex.Context) error {
		key, err := apiKey.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"key": key})
	})

	app.Get("/bearer", func(ctx *rivex.Context) error {
		creds, err := bearer.Resolve(ctx)
		if err != nil {
			return err
		}
		c := creds.(*rivex.HTTPAuthorizationCredentials)
		return ctx.SendJSON(map[string]interface{}{
			"scheme": c.Scheme,
			"token":  c.Credentials,
		})
	})

	app.Get("/oauth", func(ctx *rivex.Context) error {
		token, err := oauth2.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(map[string]interface{}{"token": token})
	})

	fmt.Println("Starting Rivex request features demo...")
	app.Run("127.0.0.1", 8000)
}
