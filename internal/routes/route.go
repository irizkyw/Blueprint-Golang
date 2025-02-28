package routes

import (
	"backends/internal/controllers"
	"backends/internal/storage/query"
	"database/sql"

	"github.com/gofiber/fiber/v2"
)

func SetupRoutes(app *fiber.App, db *sql.DB) {
	dbClient := query.NewDBClient(db)
	//userRepo := repository.NewUserRepository(db)
	userController := controllers.NewUserController(dbClient)

	api := app.Group("/api")

	api.Get("/users", userController.GetUsers)
	api.Get("/users/:id", userController.GetUserByID)
	api.Post("/users", userController.CreateUser)
	api.Post("/user/upload", userController.UploadImage)

	app.Use(func(c *fiber.Ctx) error {
		return c.Status(fiber.StatusNotFound).JSON(
			fiber.Map{
				"error":   true,
				"message": "404 Not Found",
			})
	})

}
