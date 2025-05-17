package middleware

import (
	"EasyTools/app/connect/ssh/gin"
)

func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
	}
}
