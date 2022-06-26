package main

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/otterize/otternose/mapper/pkg/resolvers"
	"net/http"
)

func main() {
	e := echo.New()
	e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.Use(middleware.Logger())
	e.Use(middleware.RemoveTrailingSlash())
	resolver := resolvers.NewResolver()
	resolver.Register(e)
	err := e.Start("0.0.0.0:8080")
	if err != nil {
		panic(err)
	}
}
