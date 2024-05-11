package main

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	r.Use(RequestLogger())

	r.POST("/api/auth/local", auth)
	r.POST("/api/l-4-lbs/updateservice", updateService)
	r.POST("/api/l-4-lbs/updateservice/delete", deleteService)
	r.POST("/api/updatecertificate", updateCertificate)
	r.POST("/api/deleteCertificates", deleteCertificates)

	if err := r.Run(":1337"); err != nil {
		panic(err)
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		t := time.Now()
		var buf bytes.Buffer
		tee := io.TeeReader(c.Request.Body, &buf)
		body, _ := io.ReadAll(tee)
		c.Request.Body = io.NopCloser(&buf)

		c.Next()

		latency := time.Since(t)

		fmt.Printf("%s %s %s %s\n%s\n",
			c.Request.Method,
			c.Request.RequestURI,
			c.Request.Proto,
			latency,
			string(body),
		)
	}
}

func auth(c *gin.Context) {
	c.JSON(200, gin.H{
		"jwt": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyIjp7InVzZXJuYW1lIjoiYWRtaW4iLCJwYXNzd29yZCI6IiQyYSQxMCR0b3J6Z0J",
	})
}

func updateService(c *gin.Context) {
	c.JSON(200, gin.H{
		"LBIPs": []string{"1.1.1.1"},
	})
}

func deleteService(c *gin.Context) {
	c.JSON(200, gin.H{
		"LBIPs": []string{},
	})
}

func updateCertificate(c *gin.Context) {
	c.JSON(200, gin.H{})
}

func deleteCertificates(c *gin.Context) {
	c.JSON(200, gin.H{})
}
