package main

import (
	"backends/config"
	"backends/internal/routes"
	"backends/internal/storage"
	"backends/pkg/shutdown"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
)

func buildServer(env config.EnvStructs) (*fiber.App, func(), error) {
	db, cleanupDB, err := storage.ConnectMySQL(env.MYSQL_HOST, env.MYSQL_PORT, env.MYSQL_USER, env.MYSQL_PASSWORD, env.MYSQL_DB, 10*time.Second)
	if err != nil {
		return nil, nil, err
	}

	app := fiber.New()

	useAuth, _ := strconv.ParseBool(os.Getenv("USE_AUTH")) //.env | USE_AUTH=true/false

	routes.SetupRoutes(app, db, useAuth)

	cleanup := func() {
		fmt.Println("Closing database connection...")
		cleanupDB()
	}

	return app, cleanup, nil
}
func run(env config.EnvStructs) (func(), error) {
	app, cleanup, err := buildServer(env)
	if err != nil {
		return nil, err
	}

	go func() {
		app.Listen("0.0.0.0:" + env.PORT)
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

	// Tunggu sinyal shutdown (Ctrl+C)
	shutdown.WaitForShutdown()
}
