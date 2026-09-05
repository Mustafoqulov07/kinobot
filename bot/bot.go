package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kinobot/config"
	"kinobot/database"
)

var (
	BotAPI *tgbotapi.BotAPI
	once   sync.Once
)

type AdminSession struct {
	State        string // title, category, format_type, description, poster, video, episode_loop
	Title        string
	Category     string
	FormatType   string // single, multi
	Description  string
	PosterFileID string
	MovieID      int64
	Code         string
	EpisodesNum  int
}

var (
	sessionsLock sync.RWMutex
	adminState   = make(map[int64]*AdminSession)
)

func GetAdminSession(userID int64) *AdminSession {
	sessionsLock.RLock()
	defer sessionsLock.RUnlock()
	return adminState[userID]
}

func SetAdminSession(userID int64, s *AdminSession) {
	sessionsLock.Lock()
	defer sessionsLock.Unlock()
	if s == nil {
		delete(adminState, userID)
	} else {
		adminState[userID] = s
	}
}

func ClearAdminSession(userID int64) {
	SetAdminSession(userID, nil)
}

func InitBot(cfg *config.Config) (*tgbotapi.BotAPI, error) {
	if cfg.BotToken == "" {
		return nil, fmt.Errorf("BOT_TOKEN ko'rsatilmagan")
	}

	var err error
	BotAPI, err = tgbotapi.NewBotAPI(cfg.BotToken)
	if err != nil {
		return nil, fmt.Errorf("tgbotapi.NewBotAPI xatosi: %w", err)
	}

	log.Printf("Telegram Bot avtorizatsiya qilindi: @%s", BotAPI.Self.UserName)

	// Set my commands
	setupCommands(cfg)

	// Webhook sozlash
	if cfg.WebhookURL != "" {
		wh, err := tgbotapi.NewWebhook(cfg.WebhookURL)
		if err == nil {
			wh.DropPendingUpdates = true
			_, err = BotAPI.Request(wh)
			if err != nil {
				log.Printf("Webhook o'rnatishda xatolik: %v", err)
			} else {
				log.Printf("Telegram Webhook o'rnatildi: %s", cfg.WebhookURL)
			}
		}
	} else {
		log.Println("BASE_URL yo'q — polling rejimida ishlaydi.")
	}

	return BotAPI, nil
}

func setupCommands(cfg *config.Config) {
	// Standard user commands
	userCmds := []tgbotapi.BotCommand{
		{Command: "start", Description: "Botni ishga tushirish va Mini App"},
		{Command: "codes", Description: "🔑 Kinolar va ularning kodlari"},
	}
	userConfig := tgbotapi.NewSetMyCommands(userCmds...)
	_, _ = BotAPI.Request(userConfig)

	// Admin commands
	adminCmds := []tgbotapi.BotCommand{
		{Command: "start", Description: "Botni ishga tushirish va Mini App"},
		{Command: "addmovie", Description: "🎬 Yangi kino/serial/multfilm qo'shish"},
		{Command: "addepisode", Description: "📺 Serialga yangi qism qo'shish"},
		{Command: "delete", Description: "❌ Kinoni o'chirish"},
		{Command: "stats", Description: "📊 Bot statistikasi"},
		{Command: "users", Description: "👥 Foydalanuvchilar ro'yxati"},
		{Command: "codes", Description: "🔑 Kinolar va ularning kodlari"},
	}

	// Super Admin commands
	superAdminCmds := []tgbotapi.BotCommand{
		{Command: "start", Description: "Botni ishga tushirish va Mini App"},
		{Command: "addmovie", Description: "🎬 Yangi kino/serial/multfilm qo'shish"},
		{Command: "addepisode", Description: "📺 Serialga yangi qism qo'shish"},
		{Command: "delete", Description: "❌ Kinoni o'chirish"},
		{Command: "stats", Description: "📊 Bot statistikasi"},
		{Command: "users", Description: "👥 Foydalanuvchilar ro'yxati"},
		{Command: "admins", Description: "👑 Adminlar ro'yxati"},
		{Command: "addadmin", Description: "⭐ Yangi admin qo'shish"},
		{Command: "deladmin", Description: "🗑 Adminni olib tashlash"},
		{Command: "codes", Description: "🔑 Kinolar va ularning kodlari"},
	}

	// Super admins scope
	for _, aid := range cfg.AdminIDs {
		setCommandsForChat(aid, superAdminCmds)
	}

	// DB admins scope
	dbAdmins, err := database.GetDBAdmins()
	if err == nil {
		for _, a := range dbAdmins {
			if !cfg.IsSuperAdmin(a.UserID) {
				setCommandsForChat(a.UserID, adminCmds)
			}
		}
	}
}

func setCommandsForChat(chatID int64, cmds []tgbotapi.BotCommand) {
	payload := map[string]interface{}{
		"commands": cmds,
		"scope": map[string]interface{}{
			"type":    "chat",
			"chat_id": chatID,
		},
	}
	jsonData, _ := json.Marshal(payload)
	_, _ = BotAPI.MakeRequest("setMyCommands", tgbotapi.Params{"commands": string(jsonData)})
}

func UpdateUserCommands(userID int64, cfg *config.Config) {
	if cfg.IsSuperAdmin(userID) {
		superAdminCmds := []tgbotapi.BotCommand{
			{Command: "start", Description: "Botni ishga tushirish va Mini App"},
			{Command: "addmovie", Description: "🎬 Yangi kino/serial/multfilm qo'shish"},
			{Command: "addepisode", Description: "📺 Serialga yangi qism qo'shish"},
			{Command: "delete", Description: "❌ Kinoni o'chirish"},
			{Command: "stats", Description: "📊 Bot statistikasi"},
			{Command: "users", Description: "👥 Foydalanuvchilar ro'yxati"},
			{Command: "admins", Description: "👑 Adminlar ro'yxati"},
			{Command: "addadmin", Description: "⭐ Yangi admin qo'shish"},
			{Command: "deladmin", Description: "🗑 Adminni olib tashlash"},
			{Command: "codes", Description: "🔑 Kinolar va ularning kodlari"},
		}
		setCommandsForChat(userID, superAdminCmds)
		return
	}

	isAdmin, _ := database.IsAdmin(userID)
	if isAdmin {
		adminCmds := []tgbotapi.BotCommand{
			{Command: "start", Description: "Botni ishga tushirish va Mini App"},
			{Command: "addmovie", Description: "🎬 Yangi kino/serial/multfilm qo'shish"},
			{Command: "addepisode", Description: "📺 Serialga yangi qism qo'shish"},
			{Command: "delete", Description: "❌ Kinoni o'chirish"},
			{Command: "stats", Description: "📊 Bot statistikasi"},
			{Command: "users", Description: "👥 Foydalanuvchilar ro'yxati"},
			{Command: "codes", Description: "🔑 Kinolar va ularning kodlari"},
		}
		setCommandsForChat(userID, adminCmds)
		return
	}

	userCmds := []tgbotapi.BotCommand{
		{Command: "start", Description: "Botni ishga tushirish va Mini App"},
		{Command: "codes", Description: "🔑 Kinolar va ularning kodlari"},
	}
	setCommandsForChat(userID, userCmds)
}

// DownloadTelegramPoster saves poster image to static/posters/{fileID}.jpg
func DownloadTelegramPoster(fileID string, cfg *config.Config) (string, error) {
	cacheDir := filepath.Join("static", "posters")
	_ = os.MkdirAll(cacheDir, 0755)
	cachePath := filepath.Join(cacheDir, fileID+".jpg")

	if _, err := os.Stat(cachePath); err == nil {
		return cachePath, nil
	}

	fileDirectURL, err := BotAPI.GetFileDirectURL(fileID)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(fileDirectURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("telegram fayl yuklashda xato status: %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(cachePath, data, 0644)
	return cachePath, err
}

func SendVideoToUser(chatID int64, fileID string, caption string) error {
	msg := tgbotapi.NewVideo(chatID, tgbotapi.FileID(fileID))
	msg.Caption = caption
	msg.ParseMode = "HTML"
	_, err := BotAPI.Send(msg)
	return err
}

func HandleUpdate(update tgbotapi.Update, cfg *config.Config) {
	if update.Message != nil {
		handleMessage(update.Message, cfg)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(update.CallbackQuery, cfg)
	}
}
