package main

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/joho/godotenv/autoload"
	"os"
)

func main() {
	app := fiber.New()

	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(healthcheck.New())

	app.Use(basicauth.New(basicauth.Config{
		Users: map[string]string{
			"user": os.Getenv("PASSWORD"),
		},
	}))

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}
