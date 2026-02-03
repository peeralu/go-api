package main

import (
	"os"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	serverPort = ":8000"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// logger สำหรับ Docker (stdout only)
func newLogger() (*zap.Logger, func(), error) {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	encoder := zapcore.NewJSONEncoder(encoderConfig)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zap.InfoLevel,
	)

	logger := zap.New(core)

	cleanup := func() {
		_ = logger.Sync()
	}

	return logger, cleanup, nil
}

func prometheusMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		timer := prometheus.NewTimer(
			httpRequestDuration.WithLabelValues(c.Method(), c.Path()),
		)
		defer timer.ObserveDuration()

		err := c.Next()

		httpRequestsTotal.WithLabelValues(
			c.Method(),
			c.Path(),
			string(rune(c.Response().StatusCode())),
		).Inc()

		return err
	}
}

func setupRoutes(app *fiber.App, logger *zap.Logger) {
	app.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))

	app.Get("/", func(c fiber.Ctx) error {
		logger.Info("Request received", zap.String("path", "/"))
		return c.SendString("Hello, World!")
	})
}

func main() {
	logger, cleanup, err := newLogger()
	if err != nil {
		panic(err)
	}
	defer cleanup()

	app := fiber.New()
	app.Use(prometheusMiddleware())
	setupRoutes(app, logger)

	logger.Info("Starting server", zap.String("port", serverPort))
	if err := app.Listen(serverPort); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
