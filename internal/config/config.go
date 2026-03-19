package config

import (
	"os"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AwsAccessKey       string
	AwsSecretAccessKey string
	AwsRegion          string
	CognitoUserPoolId  string
	Port               string
}

func LoadConfig(log *logrus.Logger) *Config {
	if os.Getenv("APP_ENV") == "development"{
		err := godotenv.Load()
		if err != nil {
			log.WithError(err).Error("Error while loading configs")
			return nil
		}
	}
	

	return &Config{
		AwsAccessKey:       os.Getenv("AWS_ACCESS_KEY_ID"),
		AwsSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AwsRegion:          os.Getenv("AWS_REGION"),
		CognitoUserPoolId:  os.Getenv("COGNITO_USER_POOL_ID"),
		Port:               os.Getenv("PORT"),
	}
}
