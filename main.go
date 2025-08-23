package main

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

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
	Formula string   `json:"formula" validate:"required"`
	Options string   `json:"options" validate:"required"`
	Files   []string `json:"files"   validate:"required,min=1"`
	Timeout int      `json:"timeout" validate:"required,min=1"`
}

// Response body.
type Response struct {
	Output  string            `json:"output"`
	Timeout bool              `json:"timeout,omitempty"`
	Files   map[string]string `json:"files"`
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
		req := new(Request)
		// parse JSON request body into struct
		if err := c.BodyParser(req); err != nil {
			log.Error(err)
			return c.SendStatus(fiber.StatusBadRequest)
		}
		log.Info(req)
		// validate required fields using struct tags
		if err := validate.Struct(req); err != nil {
			log.Error(err)
			return c.SendStatus(fiber.StatusBadRequest)
		}

		// create temporary directory
		out, err := os.MkdirTemp(".", "out-")
		if err != nil {
			log.Error(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		defer os.RemoveAll(out)

		// write formula content to formula.txt file
		if err := os.WriteFile(filepath.Join(out, "formula.txt"), []byte(req.Formula), PERM); err != nil {
			log.Error(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}
		// write options content to options.json file
		if err := os.WriteFile(filepath.Join(out, "options.json"), []byte(req.Options), PERM); err != nil {
			log.Error(err)
			return c.SendStatus(fiber.StatusInternalServerError)
		}

		// create context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
		defer cancel()

		// set up command
		cmd := exec.CommandContext(ctx, "./prover", "--out", out)

		// execute prover and get combined output
		output, err := cmd.CombinedOutput()
		if err != nil && !errors.Is(ctx.Err(), context.DeadlineExceeded) {
			log.Error(err)
		}

		// initialize response
		response := Response{
			Output:  string(output),
			Timeout: errors.Is(ctx.Err(), context.DeadlineExceeded),
			Files:   make(map[string]string),
		}

		// read output files
		for _, filename := range req.Files {
			// read file content
			content, err := os.ReadFile(filepath.Join(out, filename))
			if err != nil {
				log.Error(err)
				continue
			}
			response.Files[filename] = string(content)
		}

		// return JSON response
		return c.JSON(response)
	})

	// set port
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	log.Fatal(app.Listen(":" + port))
}
