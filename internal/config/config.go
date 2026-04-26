package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	AwsAccessKey       string
	AwsSecretAccessKey string
	AwsRegion          string
	CognitoUserPoolId  string
	CognitoAppClientID string
	CognitoIssuer      string
	CognitoJWKSURL     string
	Port               string
	PostgresHost       string
	PostgresPort       int
	PostgresUser       string
	PostgresPassword   string
	PostgresDB         string
	PostgresSSLMode    string
	DatabaseURL        string
	SMTPHost           string
	SMTPPort           int
	SMTPUsername       string
	SMTPPassword       string
	SMTPSender         string
	InviteURL          string
}

func LoadConfig(log *logrus.Logger) *Config {
	environment := os.Getenv("APP_ENV")
	if environment == "development" || environment == "" {
		err := godotenv.Load()
		if err != nil {
			log.WithError(err).Error("Error while loading configs")
			return nil
		}
	}

	smtpPort, _ := strconv.Atoi(os.Getenv("SMTP_PORT"))
	postgresPort, _ := strconv.Atoi(os.Getenv("POSTGRES_PORT"))

	return &Config{
		AwsAccessKey:       os.Getenv("AWS_ACCESS_KEY_ID"),
		AwsSecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
		AwsRegion:          os.Getenv("AWS_REGION"),
		CognitoUserPoolId:  os.Getenv("COGNITO_USER_POOL_ID"),
		CognitoAppClientID: os.Getenv("COGNITO_APP_CLIENT_ID"),
		CognitoIssuer:      os.Getenv("COGNITO_ISSUER"),
		CognitoJWKSURL:     os.Getenv("COGNITO_JWKS_URL"),
		Port:               os.Getenv("PORT"),
		PostgresHost:       os.Getenv("POSTGRES_HOST"),
		PostgresPort:       postgresPort,
		PostgresUser:       os.Getenv("POSTGRES_USER"),
		PostgresPassword:   os.Getenv("POSTGRES_PASSWORD"),
		PostgresDB:         os.Getenv("POSTGRES_DB"),
		PostgresSSLMode:    os.Getenv("POSTGRES_SSLMODE"),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		SMTPHost:           os.Getenv("SMTP_HOST"),
		SMTPPort:           smtpPort,
		SMTPUsername:       os.Getenv("SMTP_USERNAME"),
		SMTPPassword:       os.Getenv("SMTP_PASSWORD"),
		SMTPSender:         os.Getenv("SMTP_SENDER"),
		InviteURL:          os.Getenv("INVITE_URL"),
	}
}
