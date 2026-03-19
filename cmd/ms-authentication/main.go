package main

import (
	"fmt"
	"nrs-authentication/internal/config"
	"nrs-authentication/internal/handlers"
	"nrs-authentication/internal/middleware"
	"nrs-authentication/internal/service"
	"os"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var log = logrus.New()

func init() {
	log.SetOutput(os.Stdout)
	log.SetFormatter(&logrus.JSONFormatter{})
	log.SetLevel(logrus.InfoLevel)
}

func main() {
	var appConfig = config.LoadConfig(log)

	var awsService = service.NewAwsService(appConfig, log)

	var authenticationHandler = handlers.NewAuthenticationHandler(&awsService)

	router := gin.New()

	router.Use(gin.Recovery())
	router.Use(middleware.Logger(log))
	router.Use(middleware.RateLimiter())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = []string{"http://localhost:5173", "https://morder-referral-production.up.railway.app"} // Specify allowed origins
	corsConfig.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}                                    // Specify allowed methods
	corsConfig.AllowHeaders = []string{"Origin", "Content-Type", "Authorization", "Accept"}                          // Specify allowed headers
	corsConfig.ExposeHeaders = []string{"Content-Length"}                                                            // Headers the browser should be able to access
	corsConfig.AllowCredentials = true                                                                               // Allow cookies/credentials to be sent cross-origin
	corsConfig.MaxAge = 12 * time.Hour                                                                               // Cache preflight requests

	router.Use(cors.New(corsConfig))

	authenticationGroup := router.Group("/api/v1/auth/me")
	{
		authenticationGroup.POST("/attach-role", authenticationHandler.AttachRole)
	}

	log.Info(fmt.Sprintf("Starting server on port %s", appConfig.Port))
	router.Run(fmt.Sprintf(":%s", appConfig.Port))

}
