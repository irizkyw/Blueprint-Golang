package middlewares

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
)

func LoggerMiddleware(c *fiber.Ctx) error {
	start := time.Now()
	err := c.Next()
	duration := time.Since(start)

	fmt.Printf("[%s] %s %s %d %s\n",
		time.Now().Format("2025-01-02 15:04:05"),
		c.Method(),
		c.Path(),
		c.Response().StatusCode(),
		duration,
	)

	return err
}

func AuthMiddleware(c *fiber.Ctx) error {
	token := c.Get("Authorization")
	if token == "" {
		return c.Status(401).JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}
	return c.Next()
}
