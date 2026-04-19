package middleware

import (
	"net/http"
	"strings"

	"IslaMemory/BackEnd/internal/auth"
	"IslaMemory/BackEnd/internal/user"

	"github.com/gin-gonic/gin"
)

func JWTAuth(jwtSvc *auth.JWTService, userSvc *user.Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "missing authorization header",
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "invalid authorization header",
			})
			return
		}

		tokenString := parts[1]

		claims, err := jwtSvc.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "invalid token",
			})
			return
		}

		if claims.IssuedAt == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "invalid token issued_at",
			})
			return
		}

		ok, err := userSvc.IsTokenValidAfterPasswordChange(
			c.Request.Context(),
			claims.UserID,
			claims.IssuedAt.Time,
		)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "user check failed",
			})
			return
		}
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"code": 401,
				"msg":  "token expired by password change",
			})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}
