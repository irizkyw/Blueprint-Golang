package controllers

import "github.com/gofiber/fiber/v2"

type Controller struct{}

func (c *Controller) Success(ctx *fiber.Ctx, data interface{}) error {
	return ctx.JSON(fiber.Map{
		"success": true,
		"data":    data,
	})
}

func (c *Controller) Error(ctx *fiber.Ctx, message string, code int) error {
	return ctx.Status(code).JSON(fiber.Map{
		"success": false,
		"error":   message,
	})
}
