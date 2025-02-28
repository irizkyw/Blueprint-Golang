package controllers

import (
	controllers "backends/internal/controllers/handler"
	"backends/internal/models"
	"backends/internal/storage/query"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type UserController struct {
	controllers.Controller
	DB *query.DBClient
}

func NewUserController(db *query.DBClient) *UserController {
	return &UserController{
		DB: db,
	}
}

func (uc *UserController) GetUsers(c *fiber.Ctx) error {
	var users []models.User
	if err := uc.DB.All("users", &users); err != nil {
		return uc.Error(c, "Internal Server error", fiber.StatusInternalServerError)
	}

	if len(users) == 0 {
		return uc.SuccessMessage(c, "Data is null!", fiber.StatusOK)
	}
	return uc.Success(c, users, fiber.StatusOK)
}

func (uc *UserController) GetUserByID(c *fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return uc.Error(c, "Invalid user ID", fiber.StatusBadRequest)
	}

	var user models.User
	if err := uc.DB.Find("users", id, &user); err != nil {
		return uc.Error(c, "User not found", fiber.StatusNotFound)
	}

	return uc.Success(c, fiber.Map{"message": "Data retrieved", "user": user}, fiber.StatusOK)
}

func (uc *UserController) CreateUser(c *fiber.Ctx) error {
	var user models.User

	if err := c.BodyParser(&user); err != nil {
		return uc.Error(c, "Invalid request", fiber.StatusBadRequest)
	}

	cols := []string{"name", "email"}
	val := []interface{}{user.Name, user.Email}

	if user.RoleId != 0 {
		if err := uc.DB.Find("roles", user.RoleId, &user.Role); err != nil {
			return uc.Error(c, "Invalid role id does not exist", fiber.StatusBadRequest)
		}

		cols = append(cols, "role_id")
		val = append(val, user.RoleId)
	}

	lastID, err := uc.DB.Create("users", cols, val)
	if err != nil {
		return uc.Error(c, "Failed to insert user", fiber.StatusInternalServerError)
	}

	user.Id = int(lastID)
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
