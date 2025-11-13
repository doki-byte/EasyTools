package middleware

import (
	gin2 "EasyTools/app/controller/connect/ssh/gin"
)

func JWTAuth() gin2.HandlerFunc {
	return func(c *gin2.Context) {
		c.Next()
	}
}
