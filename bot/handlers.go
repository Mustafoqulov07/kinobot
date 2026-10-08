package bot

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kinobot/config"
	"kinobot/database"
)

const (
	WelcomeText = `🎬 <b>KinoApp</b>ga xush kelibsiz!

Bu yerda minglab kino, multfilm va seriallarni bir joydan topib, to'g'ridan-to'g'ri shu yerda tomosha qilishingiz mumkin.

👇 Yozish maydoni yonidagi <b>Kinolar</b> tugmasi orqali ilovani oching!`

	BtnAdmin = "✉️ Admin bilan bog'lanish"
	BtnCodes = "🔑 Kino kodlari"

	EpisodeChunk = 20
)

var fourDigitRegex = regexp.MustCompile(`^\d{4}$`)

func MainKeyboard() tgbotapi.ReplyKeyboardMarkup {
	return tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton(BtnAdmin),
			tgbotapi.NewKeyboardButton(BtnCodes),
		),
	)
}

func handleMessage(msg *tgbotapi.Message, cfg *config.Config) {
	userID := msg.From.ID
	firstName := msg.From.FirstName
	username := msg.From.UserName

	// Track user
	_ = database.TrackUser(userID, firstName, username)

	// Admin state / FSM check first
	session := GetAdminSession(userID)
	if session != nil {
		if handleAdminSessionMessage(msg, session, cfg) {
			return
		}
	}

	text := strings.TrimSpace(msg.Text)

	// Admin commands
	if strings.HasPrefix(text, "/") {
		cmdParts := strings.Fields(text)
		cmd := strings.ToLower(cmdParts[0])

		switch cmd {
		case "/start":
			handleStart(msg, cfg)
			return
		case "/codes":
			handleCodes(msg)
			return
		case "/cancel":
			handleCancel(msg)
			return
		case "/addmovie":
			handleAddMovieStart(msg, cfg)
			return
		case "/addepisode":
			handleAddEpisodeStart(msg, cfg)
			return
		case "/delete":
			handleDeleteMovie(msg, cfg)
			return
		case "/stats":
			handleStats(msg, cfg)
			return
		case "/users":
			handleUsers(msg, cfg)
			return
		case "/admins":
			handleAdminsList(msg, cfg)
			return
		case "/addadmin":
			handleAddAdmin(msg, cfg)
			return
		case "/deladmin":
			handleDelAdmin(msg, cfg)
			return
		}
	}

	// Buttons
	if text == BtnAdmin {
		handleContactAdmin(msg, cfg)
		return
	}
	if text == BtnCodes {
		handleCodes(msg)
		return
	}
	if strings.ToLower(text) == "bekor qilish" {
		handleCancel(msg)
		return
	}

	// 4-digit code search
	if fourDigitRegex.MatchString(text) {
		handleCodeSearch(msg, text)
		return
	}

	// General text search
	if text != "" && !strings.HasPrefix(text, "/") {
		handleGeneralSearch(msg, text)
		return
	}
}

func handleStart(msg *tgbotapi.Message, cfg *config.Config) {
	UpdateUserCommands(msg.From.ID, cfg)

	photo := tgbotapi.NewPhoto(msg.Chat.ID, tgbotapi.FilePath("static/banner.jpg"))
	photo.Caption = WelcomeText
	photo.ParseMode = "HTML"
	photo.ReplyMarkup = MainKeyboard()
	_, _ = BotAPI.Send(photo)
}

func handleContactAdmin(msg *tgbotapi.Message, cfg *config.Config) {
	if cfg.AdminUsername == "" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Admin hozircha sozlanmagan.")
		_, _ = BotAPI.Send(reply)
		return
	}

	url := fmt.Sprintf("https://t.me/%s", cfg.AdminUsername)
	kb := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonURL("✉️ Yozish", url),
		),
	)
	reply := tgbotapi.NewMessage(msg.Chat.ID, "Savol yoki taklifingiz bo'lsa, admin bilan bog'laning 👇")
	reply.ReplyMarkup = kb
	_, _ = BotAPI.Send(reply)
}

func handleCodes(msg *tgbotapi.Message) {
	movies, err := database.GetMovies(database.MovieFilter{Sort: "new", Limit: 30})
	if err != nil || len(movies) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Hozircha kinolar mavjud emas.")
		_, _ = BotAPI.Send(reply)
		return
	}

	var lines []string
	lines = append(lines, "🔑 <b>So'nggi qo'shilgan kinolar va kodlari:</b>\n")
	for _, m := range movies {
		catEmoji := "🎬"
		if m.Category == "serial" {
			catEmoji = "📺"
		} else if m.Category == "multfilm" {
			catEmoji = "🧸"
		}
		extra := ""
		if m.HasEpisodes && m.EpisodeCount > 1 {
			extra = fmt.Sprintf(" (%d qism)", m.EpisodeCount)
		}
		lines = append(lines, fmt.Sprintf("%s %s%s — <b>%s</b>", catEmoji, m.Title, extra, m.Code))
	}
	lines = append(lines, "\nKodni shu chatga yuborsangiz, video darhol keladi.")

	reply := tgbotapi.NewMessage(msg.Chat.ID, strings.Join(lines, "\n"))
	reply.ParseMode = "HTML"
	_, _ = BotAPI.Send(reply)
}

func buildEpisodeKeyboard(episodes []database.Episode, movieID int64, rangeIndex *int) tgbotapi.InlineKeyboardMarkup {
	if len(episodes) <= EpisodeChunk || rangeIndex != nil {
		subset := episodes
		if len(episodes) > EpisodeChunk && rangeIndex != nil {
			start := *rangeIndex * EpisodeChunk
			end := start + EpisodeChunk
			if end > len(episodes) {
				end = len(episodes)
			}
			subset = episodes[start:end]
		}

		var rows [][]tgbotapi.InlineKeyboardButton
		var curRow []tgbotapi.InlineKeyboardButton

		for _, ep := range subset {
			btn := tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("%d-qism", ep.EpisodeNumber),
				fmt.Sprintf("ep:%d", ep.ID),
			)
			curRow = append(curRow, btn)
			if len(curRow) == 4 {
				rows = append(rows, curRow)
				curRow = []tgbotapi.InlineKeyboardButton{}
			}
		}
		if len(curRow) > 0 {
			rows = append(rows, curRow)
		}

		if len(episodes) > EpisodeChunk {
			rows = append(rows, []tgbotapi.InlineKeyboardButton{
				tgbotapi.NewInlineKeyboardButtonData("⬅️ Orqaga", fmt.Sprintf("epback:%d", movieID)),
			})
		}
		return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	}

	rangeCount := (len(episodes) + EpisodeChunk - 1) / EpisodeChunk
	var rows [][]tgbotapi.InlineKeyboardButton
	var curRow []tgbotapi.InlineKeyboardButton

	for i := 0; i < rangeCount; i++ {
		startEp := episodes[i*EpisodeChunk].EpisodeNumber
		endIdx := (i + 1) * EpisodeChunk
		if endIdx > len(episodes) {
			endIdx = len(episodes)
		}
		endEp := episodes[endIdx-1].EpisodeNumber

		btn := tgbotapi.NewInlineKeyboardButtonData(
			fmt.Sprintf("%d-%d", startEp, endEp),
			fmt.Sprintf("eprange:%d:%d", movieID, i),
		)
		curRow = append(curRow, btn)
		if len(curRow) == 3 {
			rows = append(rows, curRow)
			curRow = []tgbotapi.InlineKeyboardButton{}
		}
	}
	if len(curRow) > 0 {
		rows = append(rows, curRow)
	}

	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func handleCodeSearch(msg *tgbotapi.Message, code string) {
	movie, err := database.GetMovieByCode(code)
	if err != nil || movie == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Bu kodga mos kino topilmadi. Kodni tekshirib qayta yuboring.")
		_, _ = BotAPI.Send(reply)
		return
	}

	if movie.HasEpisodes {
		episodes, _ := database.GetEpisodes(movie.ID)
		if len(episodes) > 0 {
			kb := buildEpisodeKeyboard(episodes, movie.ID, nil)
			hint := "Qismni tanlang:"
			if len(episodes) > EpisodeChunk {
				hint = "Guruhni tanlang:"
			}
			catEmoji := "🎬"
			if movie.Category == "serial" {
				catEmoji = "📺"
			} else if movie.Category == "multfilm" {
				catEmoji = "🧸"
			}
			caption := fmt.Sprintf("%s <b>%s</b>\n\n%s\n\n%s", catEmoji, movie.Title, movie.Description, hint)
			reply := tgbotapi.NewMessage(msg.Chat.ID, caption)
			reply.ParseMode = "HTML"
			reply.ReplyMarkup = kb
			_, _ = BotAPI.Send(reply)
			return
		}
	}

	_ = database.IncrementViews(movie.ID)
	videoID, err := database.GetMovieVideo(movie.ID)
	if err != nil || videoID == "" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Video fayl topilmadi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	caption := fmt.Sprintf("🎬 <b>%s</b>\n\n%s\n\nKod: %s", movie.Title, movie.Description, movie.Code)
	video := tgbotapi.NewVideo(msg.Chat.ID, tgbotapi.FileID(videoID))
	video.Caption = caption
	video.ParseMode = "HTML"
	_, _ = BotAPI.Send(video)
}

func handleGeneralSearch(msg *tgbotapi.Message, query string) {
	movies, err := database.GetMovies(database.MovieFilter{Search: query, Limit: 10})
	if err != nil || len(movies) == 0 {
		text := fmt.Sprintf("❌ <b>'%s'</b> bo'yicha hech narsa topilmadi.\n\n💡 Qidirish uchun kino nomini yoki 4 xonali kodini yuboring.", query)
		reply := tgbotapi.NewMessage(msg.Chat.ID, text)
		reply.ParseMode = "HTML"
		_, _ = BotAPI.Send(reply)
		return
	}

	if len(movies) == 1 {
		m := movies[0]
		if m.HasEpisodes {
			episodes, _ := database.GetEpisodes(m.ID)
			if len(episodes) > 0 {
				kb := buildEpisodeKeyboard(episodes, m.ID, nil)
				catEmoji := "🎬"
				if m.Category == "serial" {
					catEmoji = "📺"
				} else if m.Category == "multfilm" {
					catEmoji = "🧸"
				}
				hint := "Qismni tanlang:"
				if len(episodes) > EpisodeChunk {
					hint = "Guruhni tanlang:"
				}
				caption := fmt.Sprintf("%s <b>%s</b>\n\n%s\n\n%s", catEmoji, m.Title, m.Description, hint)
				reply := tgbotapi.NewMessage(msg.Chat.ID, caption)
				reply.ParseMode = "HTML"
				reply.ReplyMarkup = kb
				_, _ = BotAPI.Send(reply)
				return
			}
		}

		_ = database.IncrementViews(m.ID)
		videoID, _ := database.GetMovieVideo(m.ID)
		if videoID != "" {
			caption := fmt.Sprintf("🎬 <b>%s</b>\n\n%s\n\n🔑 Kod: <b>%s</b>", m.Title, m.Description, m.Code)
			video := tgbotapi.NewVideo(msg.Chat.ID, tgbotapi.FileID(videoID))
			video.Caption = caption
			video.ParseMode = "HTML"
			_, _ = BotAPI.Send(video)
			return
		}
	}

	// Multiple movies
	var lines []string
	lines = append(lines, fmt.Sprintf("🔍 <b>'%s' bo'yicha topilganlar:</b>\n", query))

	var rows [][]tgbotapi.InlineKeyboardButton
	for _, m := range movies {
		catEmoji := "🎬"
		if m.Category == "serial" {
			catEmoji = "📺"
		} else if m.Category == "multfilm" {
			catEmoji = "🧸"
		}
		extra := ""
		if m.HasEpisodes && m.EpisodeCount > 1 {
			extra = fmt.Sprintf(" (%d qism)", m.EpisodeCount)
		}
		lines = append(lines, fmt.Sprintf("%s <b>%s</b>%s — 🔑 <code>%s</code>", catEmoji, m.Title, extra, m.Code))

		btnText := fmt.Sprintf("▶ %s (%s)", m.Title, m.Code)
		btn := tgbotapi.NewInlineKeyboardButtonData(btnText, fmt.Sprintf("movie:%d", m.ID))
		rows = append(rows, []tgbotapi.InlineKeyboardButton{btn})
	}
	lines = append(lines, "\nKo'rish uchun kerakli kino tugmasini bosing:")

	reply := tgbotapi.NewMessage(msg.Chat.ID, strings.Join(lines, "\n"))
	reply.ParseMode = "HTML"
	reply.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
	_, _ = BotAPI.Send(reply)
}

func handleCallbackQuery(cb *tgbotapi.CallbackQuery, cfg *config.Config) {
	data := cb.Data

	if strings.HasPrefix(data, "cat:") {
		handleCategoryCallback(cb)
		return
	}
	if strings.HasPrefix(data, "fmt:") {
		handleFormatCallback(cb)
		return
	}

	if strings.HasPrefix(data, "eprange:") {
		parts := strings.Split(data, ":")
		if len(parts) == 3 {
			movieID, _ := strconv.ParseInt(parts[1], 10, 64)
			rangeIdx, _ := strconv.Atoi(parts[2])
			episodes, _ := database.GetEpisodes(movieID)
			kb := buildEpisodeKeyboard(episodes, movieID, &rangeIdx)
			edit := tgbotapi.NewEditMessageReplyMarkup(cb.Message.Chat.ID, cb.Message.MessageID, kb)
			_, _ = BotAPI.Send(edit)
		}
		ack := tgbotapi.NewCallback(cb.ID, "")
		_, _ = BotAPI.Request(ack)
		return
	}

	if strings.HasPrefix(data, "epback:") {
		parts := strings.Split(data, ":")
		if len(parts) == 2 {
			movieID, _ := strconv.ParseInt(parts[1], 10, 64)
			episodes, _ := database.GetEpisodes(movieID)
			kb := buildEpisodeKeyboard(episodes, movieID, nil)
			edit := tgbotapi.NewEditMessageReplyMarkup(cb.Message.Chat.ID, cb.Message.MessageID, kb)
			_, _ = BotAPI.Send(edit)
		}
		ack := tgbotapi.NewCallback(cb.ID, "")
		_, _ = BotAPI.Request(ack)
		return
	}

	if strings.HasPrefix(data, "ep:") {
		parts := strings.Split(data, ":")
		if len(parts) == 2 {
			epID, _ := strconv.ParseInt(parts[1], 10, 64)
			episode, err := database.GetEpisodeByID(epID)
			if err == nil && episode != nil {
				_ = database.IncrementViews(episode.MovieID)
				catEmoji := "🎬"
				if episode.Category == "serial" {
					catEmoji = "📺"
				} else if episode.Category == "multfilm" {
					catEmoji = "🧸"
				}
				caption := fmt.Sprintf("%s <b>%s</b> — %d-qism", catEmoji, episode.Title, episode.EpisodeNumber)
				video := tgbotapi.NewVideo(cb.Message.Chat.ID, tgbotapi.FileID(episode.VideoFileID))
				video.Caption = caption
				video.ParseMode = "HTML"
				_, _ = BotAPI.Send(video)
			}
		}
		ack := tgbotapi.NewCallback(cb.ID, "")
		_, _ = BotAPI.Request(ack)
		return
	}

	if strings.HasPrefix(data, "movie:") {
		parts := strings.Split(data, ":")
		if len(parts) == 2 {
			movieID, _ := strconv.ParseInt(parts[1], 10, 64)
			movie, err := database.GetMovieByID(movieID)
			if err == nil && movie != nil {
				if movie.HasEpisodes {
					episodes, _ := database.GetEpisodes(movie.ID)
					if len(episodes) > 0 {
						kb := buildEpisodeKeyboard(episodes, movie.ID, nil)
						hint := "Qismni tanlang:"
						if len(episodes) > EpisodeChunk {
							hint = "Guruhni tanlang:"
						}
						catEmoji := "🎬"
						if movie.Category == "serial" {
							catEmoji = "📺"
						} else if movie.Category == "multfilm" {
							catEmoji = "🧸"
						}
						caption := fmt.Sprintf("%s <b>%s</b>\n\n%s\n\n%s", catEmoji, movie.Title, movie.Description, hint)
						reply := tgbotapi.NewMessage(cb.Message.Chat.ID, caption)
						reply.ParseMode = "HTML"
						reply.ReplyMarkup = kb
						_, _ = BotAPI.Send(reply)
						ack := tgbotapi.NewCallback(cb.ID, "")
						_, _ = BotAPI.Request(ack)
						return
					}
				}

				_ = database.IncrementViews(movie.ID)
				videoID, _ := database.GetMovieVideo(movie.ID)
				if videoID != "" {
					caption := fmt.Sprintf("🎬 <b>%s</b>\n\n%s\n\n🔑 Kod: <b>%s</b>", movie.Title, movie.Description, movie.Code)
					video := tgbotapi.NewVideo(cb.Message.Chat.ID, tgbotapi.FileID(videoID))
					video.Caption = caption
					video.ParseMode = "HTML"
					_, _ = BotAPI.Send(video)
				}
			}
		}
		ack := tgbotapi.NewCallback(cb.ID, "")
		_, _ = BotAPI.Request(ack)
		return
	}
}
