package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthenticateMiddleware(c *gin.Context) {
	authorizationHeader := c.Request.Header["Authorization"][0]
	if authorizationHeader == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Authorization missing in Header",
		})
		c.Abort()
		return
	}

	tokenString := strings.Split(authorizationHeader, " ")[1]
	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Bearer Token Authorization missing in Header",
		})
		c.Abort()
		return
	}

	_, err := verifyToken(tokenString)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    http.StatusBadRequest,
			"message": "Token inválido",
		})
		c.Abort()
		return
	}

	c.Next()
}
