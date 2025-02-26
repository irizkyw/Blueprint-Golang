package controllers

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Data    interface{} `json:"data,omitempty"`
	Error   *string     `json:"error,omitempty"`
}

type Controller struct{}

func (c *Controller) Success(ctx *fiber.Ctx, data interface{}, code int) error {
	response := Response{
		Success: true,
		Code:    code,
		Data:    data,
		Error:   nil,
	}
	return ctx.Status(code).JSON(response)
}

func (c *Controller) Error(ctx *fiber.Ctx, message string, code int) error {
	response := Response{
		Success: false,
		Code:    code,
		Data:    nil,
		Error:   &message,
	}
	return ctx.Status(code).JSON(response)
}
