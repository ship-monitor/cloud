package auth

import "github.com/gin-gonic/gin"

// Deprecated: Use WithAuthenticationRequired
func (m *Middleware) WithAuthentication(ctx *gin.Context) {
	m.WithAuthenticationRequired(ctx)
}

// Deprecated: Use WithMiddleware
func (m *Middleware) WithMiddlewareOnly(ctx *gin.Context) {
	m.WithMiddleware(ctx)
}
