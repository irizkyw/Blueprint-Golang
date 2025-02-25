package routes

import (
	"backends/internal/controllers"
	"backends/internal/storage"
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, db *sql.DB, useAuth bool) {
	dbClient := storage.NewDBClient(db)
	//userRepo := repository.NewUserRepository(db)
	userController := controllers.NewUserController(dbClient)

	api := app.Group("/api")

	api.Get("/users", userController.GetUsers)
	api.Get("/users/:id", userController.GetUserByID)
	api.Post("/users", userController.CreateUser)
}
