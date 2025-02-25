package controllers

import (
	controllers "backends/internal/controllers/handler"
	"backends/internal/models"
	"backends/internal/storage"
	"strconv"

	"github.com/gofiber/fiber/v2"
)

type UserController struct {
	controllers.Controller
	DB *storage.DBClient
}

func NewUserController(db *storage.DBClient) *UserController {
	return &UserController{
		DB: db,
	}
}

func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := uc.DB.All("users", &users); err != nil {
		return uc.Error(c, "Database error", 500)
	}

	return uc.Success(c, users, fiber.StatusOK)
}

func (uc *UserController) GetUserByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return uc.Error(c, "Invalid user ID", 400)
	}

	var user models.User
	if err := uc.DB.Find("users", id, &user); err != nil {
		return uc.Error(c, "User not found", 404)
	}

	return uc.Success(c, fiber.Map{"message": "Data retrieved"}, fiber.StatusOK)
}

func (uc *UserController) CreateUser(c *fiber.Ctx) error {
	var user models.User

	if err := c.BodyParser(&user); err != nil {
		return uc.Error(c, "Invalid request", 400)
	}

	lastID, err := uc.DB.Create("users", []string{"name", "email"}, []interface{}{user.Name, user.Email})
	if err != nil {
		return uc.Error(c, "Failed to insert user", 500)
	}

	user.ID = int(lastID)
	return uc.Success(c, fiber.Map{"message": "User created", "user": user}, fiber.StatusOK)
}
