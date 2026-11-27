package main

import (
	"github.com/shregar1/rivex-go"
)

// Hard-coded "user database" keyed by token. In a real app, you'd verify the
// JWT signature or look the token up in a session store.
var users = map[string]map[string]interface{}{
	"demo-token":  {"id": 1, "name": "Ada", "role": "admin"},
	"guest-token": {"id": 2, "name": "Guest", "role": "viewer"},
}

func main() {
	app := rivex.New(rivex.WithTitle("Auth Bearer Example"))

	bearer := rivex.NewHTTPBearer()

	currentUser := rivex.DependencyFunc(func(ctx *rivex.Context) (interface{}, error) {
		creds, err := bearer.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		token := creds.(*rivex.HTTPAuthorizationCredentials).Credentials
		user, ok := users[token]
		if !ok {
			return nil, rivex.NewHTTPException(rivex.StatusUnauthorized, "Invalid bearer token")
		}
		return user, nil
	})

	app.Get("/public", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"message": "anyone can see this"})
	})

	app.Get("/me", func(ctx *rivex.Context) error {
		user, err := currentUser.Resolve(ctx)
		if err != nil {
			return err
		}
		return ctx.SendJSON(user)
	})

	app.Get("/admin", func(ctx *rivex.Context) error {
		raw, err := currentUser.Resolve(ctx)
		if err != nil {
			return err
		}
		user := raw.(map[string]interface{})
		if user["role"] != "admin" {
			return rivex.NewHTTPException(rivex.StatusForbidden, "Admins only")
		}
		return ctx.SendJSON(map[string]interface{}{
			"message": "welcome to the admin panel",
			"user":    user,
		})
	})

	app.Run("127.0.0.1", 8000)
}
