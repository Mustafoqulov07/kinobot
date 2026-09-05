package main

import (
	"log"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kinobot/bot"
	"kinobot/config"
	"kinobot/database"
	"kinobot/server"
)

func main() {
	log.Println("Kinobot Go server ishga tushirilmoqda...")

	// 1. Sozlamalarni yuklash
	cfg := config.LoadConfig()

	// 2. Ma'lumotlar bazasini ishga tushirish
	if err := database.InitDB(cfg.TursoDatabaseURL, cfg.TursoAuthToken); err != nil {
		log.Fatalf("Ma'lumotlar bazasiga ulanishda xato: %v", err)
	}
	defer database.CloseDB()

	// 3. Telegram Botni ishga tushirish
	b, err := bot.InitBot(cfg)
	if err != nil {
		log.Printf("OGOHLANTIRISH: Telegram Bot ishga tushmadi (token noto'g'ri bo'lishi mumkin): %v", err)
	} else if cfg.WebhookURL == "" {
		// Webhook sozlanmagan bo'lsa (masalan lokal ishga tushirganda), polling rejimida ishlaydi
		go func() {
			log.Println("Telegram updates polling rejimida qabul qilinmoqda...")
			u := tgbotapi.NewUpdate(0)
			u.Timeout = 60
			updates := b.GetUpdatesChan(u)
			for update := range updates {
				go bot.HandleUpdate(update, cfg)
			}
		}()
	}

	// 4. HTTP Serverni ishga tushirish
	srv := server.NewServer(cfg)
	if err := srv.Start(); err != nil {
		log.Fatalf("Server xatosi: %v", err)
	}
}
