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
	// file permission.
	PERM = 0600
)

// Request body.
type Request struct {
	Formula string `json:"formula" validate:"required"`
	Options string `json:"options" validate:"required"`
	Timeout int    `json:"timeout" validate:"required,min=1"`
}

// Response body.
type Response struct {
	Files   map[string]string `json:"files"`
	Output  string            `json:"output"`
	Timeout bool              `json:"timeout"`
}

func main() {
	// fiber instance
	app := fiber.New(fiber.Config{
		// disable startup message
		DisableStartupMessage: true,
	})

	// recover from panics
	app.Use(recover.New())
	// security
	app.Use(helmet.New())
	// logging
	app.Use(logger.New())
	// compression
	app.Use(compress.New())
	// healthcheck at /livez
	app.Use(healthcheck.New())

	// basic authentication
	app.Use(basicauth.New(basicauth.Config{
		Users: map[string]string{
			"user": os.Getenv("PASSWORD"),
		},
	}))

	// main API
	app.Post("/", prove)

	// initialize port
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// start server
	log.Info("Starting server on port:", port)
	log.Fatal(app.Listen(":" + port))
}

func prove(c *fiber.Ctx) error {
	log.Info("Received request")

	// initialize request
	req := new(Request)

	// parse
	if err := c.BodyParser(req); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}
	log.Info(req)

	// validate
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}
	log.Info("Validation passed")

	// temporary directory
	out, err := os.MkdirTemp(".", "out-")
	if err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	// cleanup
	defer func() {
		if err := os.RemoveAll(out); err != nil {
			log.Error(err)
		} else {
			log.Info("Cleaned up temporary directory:", out)
		}
	}()
	log.Info("Created temporary directory:", out)

	// write formula to file
	if err := os.WriteFile(filepath.Join(out, "formula.txt"), []byte(req.Formula), PERM); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	log.Info("Wrote formula to file")

	// write options to file
	if err := os.WriteFile(filepath.Join(out, "options.json"), []byte(req.Options), PERM); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	log.Info("Wrote options to file")

	// context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// execute prover
	cmd := exec.CommandContext(ctx, "./prover", "--out", out) // #nosec G204
	output, err := cmd.CombinedOutput()
	// check if timed out
	timeout := errors.Is(ctx.Err(), context.DeadlineExceeded)
	switch {
	case timeout:
		log.Warn("Timeout")
	case err != nil:
		log.Error(err)
	default:
		log.Info("Completed successfully")
	}

	// initialize response
	response := Response{
		Output:  string(output),
		Timeout: timeout,
		Files:   make(map[string]string),
	}

	// read all files from output directory
	files, err := os.ReadDir(out)
	if err != nil {
		log.Error(err)
		// return response without files
		return c.JSON(response)
	}
	log.Info("Found", len(files), "files in output directory")

	// process each file in output directory
	for _, file := range files {
		filename := file.Name()

		// read file
		content, err := os.ReadFile(filepath.Join(out, filename)) // #nosec G304
		if err != nil {
			log.Error(err)
			// skip and continue
			continue
		}

		// add content to response
		response.Files[filename] = string(content)
		log.Info("Added file:", filename)
	}

	// return response
	return c.JSON(response)
}
