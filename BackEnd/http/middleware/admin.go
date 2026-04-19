package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 要求 JWT 中 role = 1 才能访问
func AdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "forbidden",
			})
			return
		}

		role, ok := roleVal.(uint8)
		if !ok || role != 1 {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": 403,
				"msg":  "admin only",
			})
			return
		}

		c.Next()
	}
}
