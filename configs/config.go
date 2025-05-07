package configs

import (
	"os"
	"strconv"
)

// Config represents the application configuration
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	JWT      JWTConfig
	Email    EmailConfig
	PGP      PGPConfig
	CBR      CBRConfig
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port int
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret string
	TTL    int // in hours
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUser     string
	SMTPPassword string
	SenderEmail  string
}

// PGPConfig holds PGP encryption configuration
type PGPConfig struct {
	PublicKey  string
	PrivateKey string
	Passphrase string
}

// CBRConfig holds Central Bank RF API configuration
type CBRConfig struct {
	APIURL string
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	port, err := strconv.Atoi(getEnv("SERVER_PORT", "8080"))
	if err != nil {
		return nil, err
	}

	dbPort, err := strconv.Atoi(getEnv("DB_PORT", "5432"))
	if err != nil {
		return nil, err
	}

	jwtTTL, err := strconv.Atoi(getEnv("JWT_TTL", "24"))
	if err != nil {
		return nil, err
	}

	smtpPort, err := strconv.Atoi(getEnv("SMTP_PORT", "587"))
	if err != nil {
		return nil, err
	}

	return &Config{
		Server: ServerConfig{
			Port: port,
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     dbPort,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			DBName:   getEnv("DB_NAME", "banking_service"),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "super_secret_key"),
			TTL:    jwtTTL,
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "smtp.example.com"),
			SMTPPort:     smtpPort,
			SMTPUser:     getEnv("SMTP_USER", "user"),
			SMTPPassword: getEnv("SMTP_PASSWORD", "password"),
			SenderEmail:  getEnv("SENDER_EMAIL", "no-reply@banking-service.com"),
		},
		PGP: PGPConfig{
			PublicKey:  getEnv("PGP_PUBLIC_KEY", ""),
			PrivateKey: getEnv("PGP_PRIVATE_KEY", ""),
			Passphrase: getEnv("PGP_PASSPHRASE", ""),
		},
		CBR: CBRConfig{
			APIURL: getEnv("CBR_API_URL", "https://www.cbr.ru/DailyInfoWebServ/DailyInfo.asmx"),
		},
	}, nil
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}