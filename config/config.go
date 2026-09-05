package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	BotToken         string
	AdminIDs         []int64
	AdminUsername    string
	BaseURL          string
	WebhookPath      string
	WebhookURL       string
	TursoDatabaseURL string
	TursoAuthToken   string
	Port             string
}

var AppConfig *Config

var Categories = map[string]string{
	"kino":     "🎬 Kino",
	"multfilm": "🧸 Multfilm",
	"serial":   "📺 Serial",
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	botToken := os.Getenv("BOT_TOKEN")
	baseURL := strings.TrimRight(os.Getenv("BASE_URL"), "/")
	webhookPath := ""
	webhookURL := ""
	if botToken != "" {
		webhookPath = "/webhook/" + botToken
		if baseURL != "" {
			webhookURL = baseURL + webhookPath
		}
	}

	rawAdminIDs := os.Getenv("ADMIN_IDS")
	var adminIDs []int64
	for _, s := range strings.Split(rawAdminIDs, ",") {
		s = strings.TrimSpace(s)
		if id, err := strconv.ParseInt(s, 10, 64); err == nil {
			adminIDs = append(adminIDs, id)
		}
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	AppConfig = &Config{
		BotToken:         botToken,
		AdminIDs:         adminIDs,
		AdminUsername:    strings.TrimPrefix(strings.TrimSpace(os.Getenv("ADMIN_USERNAME")), "@"),
		BaseURL:          baseURL,
		WebhookPath:      webhookPath,
		WebhookURL:       webhookURL,
		TursoDatabaseURL: os.Getenv("TURSO_DATABASE_URL"),
		TursoAuthToken:   os.Getenv("TURSO_AUTH_TOKEN"),
		Port:             port,
	}

	return AppConfig
}

func (c *Config) IsSuperAdmin(userID int64) bool {
	for _, id := range c.AdminIDs {
		if id == userID {
			return true
		}
	}
	return false
}
