package authapi

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		// Checking if the `Authorization` header is present
		authHeader := ctx.GetHeader("Authorization")
		if authHeader == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		// Extracting the token from the `Authorization` header
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header"})
			return
		}

		// Validating the token
		username, err := h.jwtManager.ValidateToken(parts[1])
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// Setting the username in the context
		ctx.Set("username", username)
		ctx.Next()
	}
}
