package main

import (
	"os"

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
)

const (
	PERM = 0600
)

// Request body.
type Request struct {
	Formula string `json:"formula" validate:"required"`
	Options string `json:"options" validate:"required"`
	Timeout int    `json:"timeout" validate:"required"`
}

func main() {
	// create fiber instance
	app := fiber.New()

	// create validator instance
	validate := validator.New()

	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(logger.New())
	app.Use(compress.New())
	app.Use(healthcheck.New())

	// for basic authentication
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
		// write formula content to formula.txt file
		if err := os.WriteFile("formula.txt", []byte(req.Formula), PERM); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// write options content to options.json file
		if err := os.WriteFile("options.json", []byte(req.Options), PERM); err != nil {
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		return c.SendString("Hello, World!")
	})

	// set port
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(app.Listen(":" + port))
}
