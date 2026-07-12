package pkg

import "github.com/gin-gonic/gin"

type Handler interface {
	SetupRoutes(router gin.IRouter)
}
