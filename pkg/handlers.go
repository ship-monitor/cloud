package pkg

import "github.com/labstack/echo/v5"

type Handler interface {
	SetupRoutes(router *echo.Group)
}
