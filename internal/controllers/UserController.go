package controllers

import (
	controllers "backends/internal/controllers/handler"
	"backends/internal/models"
	"backends/internal/storage"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
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
		return uc.Error(c, "Internal Server error", 500)
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

func (uc *UserController) UploadImage(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		fmt.Println("FormFile error:", err)
		return uc.Error(c, "File is required", fiber.StatusBadRequest)
	}

	upload_dir, _ := filepath.Abs("./uploads")

	if _, err := os.Stat(upload_dir); os.IsNotExist(err) {
		if err := os.Mkdir(upload_dir, os.ModePerm); err != nil {
			return uc.Error(c, "Failed to create upload directory", fiber.StatusInternalServerError)
		}
	}

	fileExt := filepath.Ext(file.Filename)
	generate_name := uuid.New().String() + fileExt
	filePath := filepath.Join(upload_dir, generate_name)

	if err := c.SaveFile(file, filePath); err != nil {
		return uc.Error(c, "Failed to save file", fiber.StatusInternalServerError)
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return uc.Error(c, "File save failed", fiber.StatusInternalServerError)
	}

	return uc.Success(c, fiber.Map{"message": "File uploaded successfully", "file": generate_name}, fiber.StatusOK)
}
