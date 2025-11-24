package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Kaamos-Comms/server/internal/middleware"
	"github.com/Kaamos-Comms/server/internal/signaling"
	"github.com/labstack/echo/v4"
	echomiddleware "github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

var defaultPort = "8080"

type App struct {
	e               *echo.Echo
	signalingServer *signaling.Server
	port            string
}

func Initialize() *App {
	app := &App{
		e:               echo.New(),
		signalingServer: signaling.NewServer(),
		port:            getPort(),
	}

	app.e.HideBanner = true
	app.e.HidePort = false

	app.e.Use(echomiddleware.Logger())
	app.e.Use(echomiddleware.Recover())
	app.e.Use(echomiddleware.CORS())
	app.e.Use(echomiddleware.RequestID())

	// 🟢 No rate limiting
	app.e.GET("/health", healthHandler)

	// 🔴 5 req/min
	strictLimiter := middleware.NewIPRateLimiter(rate.Every(time.Minute/5), 1)
	strictProtected := app.e.Group("")
	strictProtected.Use(strictLimiter.Middleware())
	strictProtected.POST("/api/rooms/create", func(c echo.Context) error {
		// TODO: Implement room creation handler
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// 🟡 10 req/min
	lightLimiter := middleware.NewIPRateLimiter(rate.Every(time.Minute/10), 2)
	lightProtected := app.e.Group("")
	lightProtected.Use(lightLimiter.Middleware())
	lightProtected.GET("/api/rooms/:room_id", func(c echo.Context) error {
		// TODO: Implement room info handler
		return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// 🔴 3 req/min
	wsLimiter := middleware.NewIPRateLimiter(rate.Every(time.Minute/3), 1)
	app.e.GET("/ws/:room_id", wsLimiter.Middleware()(echo.WrapHandler(http.HandlerFunc(app.signalingServer.HandleWebSocket))))

	return app
}

func (a *App) Start() {
	go func() {
		log.Printf("Starting server on port %s", a.port)
		if err := a.e.Start(":" + a.port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()
}

func (a *App) Shutdown(ctx context.Context) error {
	a.signalingServer.Shutdown()
	return a.e.Shutdown(ctx)
}

func getPort() string {
	if port := os.Getenv("PORT"); port != "" {
		return port
	}
	return defaultPort
}
