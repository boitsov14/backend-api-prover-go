package main

import (
	"github.com/go-playground/validator/v10"
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

// Request body.
type Request struct {
	Formula string `json:"formula" validate:"required"`
	Options string `json:"options" validate:"required"`
	Timeout int    `json:"timeout" validate:"required"`
}

func main() {
	app := fiber.New()

	validate := validator.New()

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

	app.Post("/", func(c *fiber.Ctx) error {
		// parse JSON request body into struct
		req := new(Request)
		if err := c.BodyParser(req); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}
		// validate required fields using struct tags
		if err := validate.Struct(req); err != nil {
			return c.SendStatus(fiber.StatusBadRequest)
		}

		return c.SendString("Hello, World!")
	})

	log.Fatal(app.Listen(":3000"))
}
