package main

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
)

const (
	// file permission.
	PERM = 0600
)

// Request body.
type Request struct {
	Options map[string]any `json:"options" validate:"required"`
	Formula string         `json:"formula" validate:"required"`
	Timeout int            `json:"timeout" validate:"required,min=1"`
}

// Response body.
type Response struct {
	Files  map[string]string `json:"files"`
	Result map[string]any    `json:"result"`
}

func main() {
	// fiber instance
	app := fiber.New(fiber.Config{
		// disable startup message
		DisableStartupMessage: true,
	})

	// add middlewares
	app.Use(recover.New())     // recover from panics
	app.Use(helmet.New())      // security
	app.Use(logger.New())      // logging
	app.Use(compress.New())    // compression
	app.Use(healthcheck.New()) // healthcheck at /livez

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
	log.Info("Request received")

	// initialize request
	req := new(Request)

	// parse
	if err := c.BodyParser(req); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}
	// TODO: 2025/09/10 google cloud run上でどう見えるか確認する
	log.Info(req)

	// validate
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}
	log.Info("Validation passed")

	// temporary directory
	tmp, err := os.MkdirTemp(".", "tmp-")
	if err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	// cleanup
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			log.Error(err)
		} else {
			log.Info("Cleaned up tmp directory: ", tmp)
		}
	}()
	log.Info("Created tmp directory: ", tmp)

	// write formula to file
	if err := os.WriteFile(filepath.Join(tmp, "formula.txt"), []byte(req.Formula), PERM); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// convert options to JSON string
	options, err := json.MarshalIndent(req.Options, "", "  ")
	if err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	// write options to file
	if err := os.WriteFile(filepath.Join(tmp, "options.json"), options, PERM); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	log.Info("Wrote input files")

	// context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// execute prover
	prover := "./prover"
	if runtime.GOOS == "windows" {
		prover = "./prover.exe"
	}
	log.Info("Proving..")
	cmd := exec.CommandContext(ctx, prover, "--out", tmp) // #nosec G204
	stdout, err := cmd.CombinedOutput()
	// check if timed out
	timeout := errors.Is(ctx.Err(), context.DeadlineExceeded)
	switch {
	case timeout:
		log.Warn("Timeout")
	case err != nil:
		log.Error(err)
	default:
		log.Info("Done")
	}

	// remove input files
	if err := os.Remove(filepath.Join(tmp, "formula.txt")); err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	if err := os.Remove(filepath.Join(tmp, "options.json")); err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	log.Info("Removed input files")

	// initialize response
	response := Response{
		Files: make(map[string]string),
	}

	// parse result.log
	k := koanf.New(".")
	if err := k.Load(file.Provider(filepath.Join(tmp, "result.log")), yaml.Parser()); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	response.Result = k.All()
	log.Info("Parsed result.log")

	// add stdout and timeout to result
	response.Result["stdout"] = string(stdout)
	response.Result["timeout"] = timeout

	// remove result.log
	if err := os.Remove(filepath.Join(tmp, "result.log")); err != nil {
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	log.Info("Removed result.log")

	// read all files from output directory
	files, err := os.ReadDir(tmp)
	if err != nil {
		log.Error(err)
		// return response without files
		return c.JSON(response)
	}
	log.Info("Found ", len(files), " files in tmp directory")

	// process each file in tmp directory
	for _, f := range files {
		// get filename
		filename := f.Name()

		// read file
		content, err := os.ReadFile(filepath.Join(tmp, filename)) // #nosec G304
		if err != nil {
			log.Error(err)
			// skip and continue
			continue
		}

		// add content to response
		response.Files[filename] = string(content)
	}
	log.Info("Added all files")

	// return response
	return c.JSON(response)
}
