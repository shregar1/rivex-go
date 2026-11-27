package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/shregar1/rivex-go"
)

var (
	auditLog   []string
	auditMutex sync.Mutex
)

func appendAudit(line string) {
	auditMutex.Lock()
	defer auditMutex.Unlock()
	auditLog = append(auditLog, line)
}

func main() {
	app := rivex.New(rivex.WithTitle("Background Tasks Example"))

	app.Post("/signup/{email}", func(ctx *rivex.Context) error {
		email := ctx.PathParam("email")

		tasks := rivex.NewBackgroundTasks()
		tasks.AddTask(func() {
			time.Sleep(50 * time.Millisecond)
			appendAudit(fmt.Sprintf("sent welcome email to %s", email))
		})
		tasks.AddTask(func() {
			time.Sleep(20 * time.Millisecond)
			appendAudit(fmt.Sprintf("provisioned workspace for %s", email))
		})

		go tasks.RunAll()

		return ctx.SendJSONWithStatus(map[string]string{
			"status": "accepted",
			"email":  email,
		}, rivex.StatusAccepted)
	})

	app.Get("/audit", func(ctx *rivex.Context) error {
		auditMutex.Lock()
		defer auditMutex.Unlock()
		copyLog := make([]string, len(auditLog))
		copy(copyLog, auditLog)
		return ctx.SendJSON(map[string]interface{}{"log": copyLog})
	})

	app.Run("127.0.0.1", 8000)
}
