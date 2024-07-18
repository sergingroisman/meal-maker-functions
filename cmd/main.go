package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sergingroisman/meal-maker-functions/cmd/config"
	"github.com/sergingroisman/meal-maker-functions/database"
	"github.com/sergingroisman/meal-maker-functions/handlers"
)

func initRoutes(router *gin.Engine) {
	ctx := context.Background()

	err := config.Init()
	if err != nil {
		log.Fatalf("Falha ao obter as variáveis de ambiente internas: %s", err)
	}

	MONGODB_URL := config.Env.MongoDB.URL
	MONGODB_DATABASE := config.Env.MongoDB.Database

	_, database, err := database.GetConnection(ctx, database.MongodbConfig{
		ConnectionURL: MONGODB_URL,
		Database:      MONGODB_DATABASE,
	})
	if err != nil {
		log.Fatalf("Não foi possível estabelecer uma conexão com o MongoDB, %s\n", err)
		return
	}

	h := handlers.NewHandlers(ctx, database)

	router.Use(cors.New(cors.Config{
		AllowMethods:     []string{"GET", "POST", "OPTIONS", "PUT"},
		AllowHeaders:     []string{"Origin", "Authorization", "Content-Type", "Content-Length", "User-Agent", "Host", "Referrer"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		AllowAllOrigins:  false,
		AllowOriginFunc:  func(origin string) bool { return true },
		MaxAge:           12 * time.Hour,
	}))

	api := router.Group("/api")
	{
		api.GET("/get-users", handlers.AuthenticateMiddleware, h.GetUsers)
		api.GET("/get-user/:phone_number", handlers.AuthenticateMiddleware, h.GetUserByPhoneNumber)
		api.POST("/update-user-password/:phone_number", handlers.AuthenticateMiddleware, h.UpdatePassword)
		api.POST("/sign-up", h.SignUp)
		api.POST("/sign-in", h.SignIn)
		api.GET("/get-restaurant/:partner_id", h.GetRestaurantByPartnerId)
		api.GET("/health-check", func(c *gin.Context) {
			c.Header("Content-Type", "application/json")
			c.JSON(http.StatusOK, gin.H{
				"status_code": http.StatusOK,
				"message":     "Application is healthy",
			})
		})
	}

}

func getAddr() string {
	listenAddr := ":8080"
	if val, ok := os.LookupEnv("FUNCTIONS_CUSTOMHANDLER_PORT"); ok {
		listenAddr = ":" + val
	}
	return listenAddr
}

func main() {
	listenAddr := getAddr()
	router := gin.Default()
	initRoutes(router)
	log.Printf("Custom handlers server is running on http://127.0.0.1%s", listenAddr)
	log.Fatal(router.Run(listenAddr))
}
