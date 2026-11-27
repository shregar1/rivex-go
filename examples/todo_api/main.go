package main

import (
	"sync"

	"github.com/shregar1/rivex-go"
)

// Todo represents a todo item.
type Todo struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// TodoCreate is the request body for creating or updating a todo.
type TodoCreate struct {
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

// In-memory store
var (
	db     = make(map[int]*Todo)
	nextID = 1
	mu     sync.RWMutex
)

func main() {
	app := rivex.New(
		rivex.WithTitle("Todo API"),
		rivex.WithDebug(true),
	)

	// Add CORS middleware
	app.Use(rivex.NewCORS(rivex.AllowOrigins("*")))

	// Create a router with a prefix
	router := rivex.NewRouter(rivex.WithPrefix("/api/v1"))

	// List all todos
	router.Get("/todos", func(ctx *rivex.Context) error {
		mu.RLock()
		defer mu.RUnlock()

		todos := make([]*Todo, 0, len(db))
		for _, t := range db {
			todos = append(todos, t)
		}
		return ctx.SendJSON(todos)
	})

	// Get a single todo
	router.Get("/todos/{todo_id}", func(ctx *rivex.Context) error {
		mu.RLock()
		defer mu.RUnlock()

		id := mustParseInt(ctx.PathParam("todo_id"))
		todo, ok := db[id]
		if !ok {
			return rivex.NewHTTPException(rivex.StatusNotFound, "Todo not found")
		}
		return ctx.SendJSON(todo)
	})

	// Create a new todo
	router.Post("/todos", func(ctx *rivex.Context) error {
		var input TodoCreate
		if err := ctx.JSON(&input); err != nil {
			return rivex.NewHTTPException(rivex.StatusBadRequest, "Invalid request body")
		}
		if input.Title == "" {
			return rivex.NewHTTPException(rivex.StatusUnprocessableEntity, "Title is required")
		}

		mu.Lock()
		todo := &Todo{
			ID:        nextID,
			Title:     input.Title,
			Completed: input.Completed,
		}
		db[nextID] = todo
		nextID++
		mu.Unlock()

		return ctx.SendJSONWithStatus(todo, rivex.StatusCreated)
	}, rivex.WithStatusCode(rivex.StatusCreated))

	// Update a todo
	router.Put("/todos/{todo_id}", func(ctx *rivex.Context) error {
		id := mustParseInt(ctx.PathParam("todo_id"))

		var input TodoCreate
		if err := ctx.JSON(&input); err != nil {
			return rivex.NewHTTPException(rivex.StatusBadRequest, "Invalid request body")
		}

		mu.Lock()
		defer mu.Unlock()

		if _, ok := db[id]; !ok {
			return rivex.NewHTTPException(rivex.StatusNotFound, "Todo not found")
		}

		db[id] = &Todo{
			ID:        id,
			Title:     input.Title,
			Completed: input.Completed,
		}
		return ctx.SendJSON(db[id])
	})

	// Delete a todo
	router.Delete("/todos/{todo_id}", func(ctx *rivex.Context) error {
		id := mustParseInt(ctx.PathParam("todo_id"))

		mu.Lock()
		defer mu.Unlock()

		if _, ok := db[id]; !ok {
			return rivex.NewHTTPException(rivex.StatusNotFound, "Todo not found")
		}

		delete(db, id)
		return ctx.SendJSONWithStatus(nil, rivex.StatusNoContent)
	})

	// Include the router
	app.IncludeRouter(router)

	// Health check at root level
	app.Get("/health", func(ctx *rivex.Context) error {
		return ctx.SendJSON(map[string]string{"status": "ok"})
	})

	app.Run("127.0.0.1", 8000)
}

func mustParseInt(s string) int {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return 0
		}
		n = n*10 + int(ch-'0')
	}
	return n
}
