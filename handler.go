package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/sergingroisman/meal-maker-functions/database"
	"github.com/sergingroisman/meal-maker-functions/handlers"
)

func initRoutes(router *gin.Engine) {
	ctx := context.Background()

	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %s", err)
	}

	MONGODB_URL := os.Getenv("MONGODB_URL")
	MONGODB_DATABASE := os.Getenv("MONGODB_DATABASE")

	_, database, err := database.GetConnection(ctx, database.MongodbConfig{
		ConnectionURL: MONGODB_URL,
		Database:      MONGODB_DATABASE,
	})
	if err != nil {
		log.Fatalf("Não foi possível estabelecer uma conexão com o MongoDB, %s\n", err)
		return
	}

	h := handlers.NewHandlers(ctx, database)

	router.GET("/api/get-users", handlers.AuthenticateMiddleware, h.GetUsers)
	router.GET("/api/get-user/:phoneNumber", handlers.AuthenticateMiddleware, h.GetUserByPhoneNumber)
	router.POST("/api/sign-up", h.SignUp)
	router.POST("/api/sign-in", h.SignIn)

	router.GET("/api/get-menu/", handlers.AuthenticateMiddleware, h.GetUserByPhoneNumber)
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
