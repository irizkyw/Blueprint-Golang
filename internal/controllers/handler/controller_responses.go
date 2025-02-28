package controllers

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Error   *string     `json:"error,omitempty"`
	Message *string     `json:"message,omitempty"`
}

type Controller struct{}

func (c *Controller) Success(ctx *fiber.Ctx, data interface{}, code int) error {
	response := Response{
		Success: true,
		Code:    code,
		Data:    data,
	}
	return ctx.Status(code).JSON(response)
}

func (c *Controller) SuccessMessage(ctx *fiber.Ctx, message string, code int) error {
	response := Response{
		Success: true,
		Code:    code,
		Message: &message,
	}
	return ctx.Status(code).JSON(response)
}

func (c *Controller) Error(ctx *fiber.Ctx, message string, code int) error {
	response := Response{
		Success: false,
		Code:    code,
		Error:   &message,
	}
	return ctx.Status(code).JSON(response)
}

// BadRequest response status 400
func (c *Controller) BadRequest(ctx *fiber.Ctx, message string) error {
	return c.Error(ctx, message, fiber.StatusBadRequest)
}

// Unauthorized response status 401
func (c *Controller) Unauthorized(ctx *fiber.Ctx, message string) error {
	return c.Error(ctx, message, fiber.StatusUnauthorized)
}

// Forbidden response status 403
func (c *Controller) Forbidden(ctx *fiber.Ctx, message string) error {
	return c.Error(ctx, message, fiber.StatusForbidden)
}

// NotFound response status 404
func (c *Controller) NotFound(ctx *fiber.Ctx, message string) error {
	return c.Error(ctx, message, fiber.StatusNotFound)
}

// InternalServerError response status 500
func (c *Controller) InternalServerError(ctx *fiber.Ctx, message string) error {
	return c.Error(ctx, message, fiber.StatusInternalServerError)
}
