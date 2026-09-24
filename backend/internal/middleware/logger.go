package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

func ErrorLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		started := time.Now()
		c.Next()
		for _, item := range c.Errors {
			log.Printf("request failed method=%s path=%s status=%d duration=%s error=%v", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(started), item.Err)
		}
	}
}
