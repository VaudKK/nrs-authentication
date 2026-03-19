package main

import (
	"context"
	"fmt"
	"nrs-authentication/internal/config"
	"nrs-authentication/internal/handlers"
	"nrs-authentication/internal/middleware"
	"nrs-authentication/internal/service"
	"os"
	"time"

	awsConfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "nrs-authentication/docs"
)

var log = logrus.New()

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
}

// @title nrs-authentication
// @version 1.0
// @description Handles authentication utilities
// @host localhost:8080
// @basePath /api/v1/auth/me
func main() {
	var appConfig = config.LoadConfig(log)

	ctx := context.TODO()
	cfg, err := awsConfig.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatal(err)
	}

	var awsService = service.NewAwsService(appConfig, log, cfg)

	var authenticationHandler = handlers.NewAuthenticationHandler(&awsService)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.Logger(log))
	router.Use(middleware.RateLimiter())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:5173", "https://nrs-authentication-production.up.railway.app", "https://morder-referral-production.up.railway.app"} // Specify allowed origins
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}                                                                                            // Specify allowed methods
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept"}                                                                                  // Specify allowed headers
	corsConfig.ExposeHeaders = []string{"Content-Length"}                                                                                                                    // Headers the browser should be able to access
	corsConfig.AllowCredentials = true                                                                                                                                       // Allow cookies/credentials to be sent cross-origin
	corsConfig.MaxAge = 12 * time.Hour                                                                                                                                       // Cache preflight requests

	router.Use(cors.New(corsConfig))

	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	authenticationGroup := router.Group("/api/v1/auth/me")
	{
		authenticationGroup.POST("/attach-role", authenticationHandler.AttachRole)
		authenticationGroup.GET("/get-facility-users", authenticationHandler.GetFacilityUsers)
	}

	log.Info(fmt.Sprintf("Starting server on port %s", appConfig.Port))
	router.Run(fmt.Sprintf(":%s", appConfig.Port))

}
