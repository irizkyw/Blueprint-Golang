package main

import (
	"backends/config"
	"backends/internal/routes"
	database "backends/internal/storage/databases"
	"backends/pkg/shutdown"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
)

func buildServer(env config.EnvStructs) (*fiber.App, func(), error) {
	db, cleanupDB, err := database.ConnectDatabase(
		database.MySQL,
		env.DB_HOST, env.DB_PORT, env.DB_USER, env.DB_PASSWORD, env.DB_DATABASE,
		10*time.Second,
	)

	if err != nil {
		fmt.Printf("Warning: Failed connection to database: %v\n", err)
		db = nil
	}

	app := fiber.New()

	if db != nil {
		routes.SetupRoutes(app, db.GetSQLDB())
	} else {
		fmt.Println("Database not available, only error handler displayed.")
		app.Use(func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"code":  fiber.ErrInternalServerError,
				"error": "Internal Server error",
			})
		})
	}

	cleanup := func() {
		if cleanupDB != nil {
			fmt.Println("Closing database connection...")
			cleanupDB()
		}
	}

	return app, cleanup, nil
}

func run(env config.EnvStructs) (func(), error) {
	app, cleanup, err := buildServer(env)
	if err != nil {
		return nil, err
	}

	go func() {
		if err := app.Listen(env.APP_URL + ":" + env.PORT); err != nil {
			fmt.Printf("Error starting server: %v\n", err)
		}
	}()

	return func() {
		cleanup()
		app.Shutdown()
	}, nil
}

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	env, err := config.LoadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		exitCode = 1
		return
	}

	cleanup, err := run(env)
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		exitCode = 1
		return
	}
	defer cleanup()

	// Tunggu shutdown (Ctrl+C)
	shutdown.WaitForShutdown()
}
