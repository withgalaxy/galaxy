package api

import (
	"time"

	"github.com/withgalaxy/galaxy/pkg/endpoints"
)

func GET(ctx *endpoints.Context) error {
	return ctx.JSON(200, map[string]interface{}{
		"message":    "Hello from Galaxy SSR!",
		"time":       time.Now().Format(time.RFC3339),
		"path":       ctx.Request.URL.Path,
		"timestamp":  ctx.Locals["timestamp"],
		"serverName": ctx.Locals["serverName"],
	})
}

func POST(ctx *endpoints.Context) error {
	var body map[string]interface{}
	if err := ctx.BindJSON(&body); err != nil {
		return ctx.JSON(400, map[string]string{"error": "Invalid JSON"})
	}

	return ctx.JSON(200, map[string]interface{}{
		"received": body,
		"message":  "POST received",
	})
}
