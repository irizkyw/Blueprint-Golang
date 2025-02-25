package controllers

import (
	"backends/internal/models"
	"backends/internal/storage"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type UserController struct {
	DB *storage.DBClient
}

func NewUserController(db *storage.DBClient) *UserController {
	return &UserController{DB: db}
}

func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := uc.DB.All("users", &users); err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Database error"})
	}

	return c.JSON(users)
}

func (uc *UserController) GetUserByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid user ID"})
	}

	var user models.User
	if err := uc.DB.Find("users", id, &user); err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "User not found"})
	}

	return c.JSON(user)
}

func (uc *UserController) CreateUser(c *fiber.Ctx) error {
	var user models.User

	if err := c.BodyParser(&user); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	lastID, err := uc.DB.Create("users", []string{"name", "email"}, []interface{}{user.Name, user.Email})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": "Failed to insert user"})
	}

	user.ID = int(lastID)
	return c.Status(201).JSON(fiber.Map{"message": "User created", "user": user})
}
