package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-yaml"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/log"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/gofiber/fiber/v2/middleware/healthcheck"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	_ "github.com/joho/godotenv/autoload"
)

// Request body.
type Request struct {
	Options map[string]any `json:"options" validate:"required"`
	Formula string         `json:"formula" validate:"required"`
	Timeout int            `json:"timeout" validate:"required,min=1,max=10"`
	Trace   bool           `json:"trace"`
}

// Response body.
type Response struct {
	Files  map[string]map[string]string `json:"files"`
	Result map[string]any               `json:"result"`
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

	// setup json logger
	l := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(l)

	// main API
	app.Post("/", prove)

	// init port
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// use localhost in dev environment
	host := ""
	if os.Getenv("ENV") == "dev" {
		host = "localhost"
	}

	// start server
	log.Info("Starting server on port: ", port)
	log.Fatal(app.Listen(host + ":" + port))
}

func prove(c *fiber.Ctx) error {
	log.Info("Request received")

	// ==============================
	// ==  Parse and Validate
	// ==============================

	// init request
	req := new(Request)

	// parse
	if err := c.BodyParser(req); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// validate
	validate := validator.New()
	if err := validate.Struct(req); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusBadRequest)
	}
	slog.Info("Request parsed", "request", req)

	// ==============================
	// ==  Temp directory and files
	// ==============================

	// tmp directory
	tmpPath, err := os.MkdirTemp(".", "tmp-")
	if err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	tmp := filepath.Base(tmpPath)
	log.Info("Created tmp directory: ", tmp)

	// cleanup
	defer func() {
		if err := os.RemoveAll(tmp); err != nil {
			log.Error(err)
		} else {
			log.Info("Cleaned up tmp directory: ", tmp)
		}
	}()

	// write formula to file
	if err := os.WriteFile(filepath.Join(tmp, "formula.txt"), []byte(req.Formula), 0400); err != nil {
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
	if err := os.WriteFile(filepath.Join(tmp, "options.json"), options, 0400); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// ==============================
	// ==  Execute prover
	// ==============================

	// context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(req.Timeout)*time.Second)
	defer cancel()

	// setup prover path
	prover := "prover"
	if req.Trace {
		prover += "-trace"
	}
	if runtime.GOOS == "windows" {
		prover += "-windows.exe"
	}
	prover = filepath.Join(".", "bin", prover)

	// execute prover
	log.Info("Proving..")
	cmd := exec.CommandContext(ctx, prover, "--out", tmp) // #nosec G204
	stdout, err := cmd.CombinedOutput()

	// check if timed out
	timeout := errors.Is(ctx.Err(), context.DeadlineExceeded)

	// log result
	switch {
	case timeout:
		log.Warn("Timeout")
	case err != nil:
		log.Error(err)
	default:
		log.Info("Done")
	}

	// ==============================
	// ==  Setup Result
	// ==============================

	// init response
	response := new(Response)

	// read result.yaml
	content, err := os.ReadFile(filepath.Join(tmp, "result.yaml")) // #nosec G304
	if err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	// parse YAML
	if err := yaml.Unmarshal(content, &response.Result); err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// add stdout if not empty
	if s := string(stdout); s != "" {
		response.Result["stdout"] = s
	}
	// add timeout if timed out
	if timeout {
		response.Result["timeout"] = true
	}

	// ==============================
	// ==  Setup Files
	// ==============================

	// init files
	response.Files = make(map[string]map[string]string)

	// read files from tmp directory
	files, err := os.ReadDir(tmp)
	if err != nil {
		log.Error(err)
		return c.SendStatus(fiber.StatusInternalServerError)
	}

	// process each file in tmp directory
	for _, f := range files {
		// get filename
		filename := f.Name()

		// skip input/result files
		switch filename {
		case "formula.txt", "options.json", "result.yaml":
			continue
		}

		// read file
		bytes, err := os.ReadFile(filepath.Join(tmp, filename)) // #nosec G304
		if err != nil {
			log.Error(err)
			// skip
			continue
		}

		// skip empty files
		content := string(bytes)
		if content == "" {
			continue
		}

		// split filename into base and extension
		base, ext, _ := strings.Cut(filename, ".")

		// check if extension map exists
		if _, ok := response.Files[ext]; !ok {
			response.Files[ext] = make(map[string]string)
		}

		// add to files
		response.Files[ext][base] = content
	}

	// return response
	return c.JSON(response)
}
