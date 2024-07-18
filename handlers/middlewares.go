package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthenticateMiddleware(c *gin.Context) {
	authorization_header := c.Request.Header["Authorization"][0]
	if authorization_header == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Authorization missing in Header",
		})
		c.Abort()
		return
	}

	token_string := strings.Split(authorization_header, " ")[1]
	if token_string == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Bearer Token Authorization missing in Header",
		})
		c.Abort()
		return
	}

	_, err := verifyToken(token_string)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status_code": http.StatusBadRequest,
			"message":     "Token inválido",
		})
		c.Abort()
		return
	}

	c.Next()
}
