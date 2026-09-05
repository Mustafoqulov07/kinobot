package bot

import (
	"fmt"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kinobot/config"
	"kinobot/database"
)

func IsAdminUser(userID int64, cfg *config.Config) bool {
	if cfg.IsSuperAdmin(userID) {
		return true
	}
	isAdmin, _ := database.IsAdmin(userID)
	return isAdmin
}

func handleCancel(msg *tgbotapi.Message) {
	session := GetAdminSession(msg.From.ID)
	if session == nil {
		return
	}
	ClearAdminSession(msg.From.ID)

	if session.State == "episode_loop" {
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf(
			"❌ Qismlar qo'shish to'xtatildi.\nKino bazada saqlangan (kod: <b>%s</b>).\nKeyinchalik yangi qismlar qo'shish uchun <code>/addepisode %s</code> dan foydalanishingiz mumkin.",
			session.Code, session.Code,
		))
		reply.ParseMode = "HTML"
		_, _ = BotAPI.Send(reply)
	} else {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Yangi kino qo'shish jarayoni bekor qilindi.")
		_, _ = BotAPI.Send(reply)
	}
}

func CategoryKeyboard() tgbotapi.InlineKeyboardMarkup {
	var rows [][]tgbotapi.InlineKeyboardButton
	for k, label := range config.Categories {
		btn := tgbotapi.NewInlineKeyboardButtonData(label, "cat:"+k)
		rows = append(rows, []tgbotapi.InlineKeyboardButton{btn})
	}
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func FormatKeyboard() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📼 1 ta to'liq qism (Film)", "fmt:single"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🎞 Ko'p qismli (1, 2, 3... qismlar)", "fmt:multi"),
		),
	)
}

func handleAddMovieStart(msg *tgbotapi.Message, cfg *config.Config) {
	if !IsAdminUser(msg.From.ID, cfg) {
		return
	}
	SetAdminSession(msg.From.ID, &AdminSession{State: "title"})
	reply := tgbotapi.NewMessage(msg.Chat.ID, "🎬 Yangi kino/serial nomini yuboring:")
	_, _ = BotAPI.Send(reply)
}

func handleCategoryCallback(cb *tgbotapi.CallbackQuery) {
	session := GetAdminSession(cb.From.ID)
	if session == nil || session.State != "category" {
		ack := tgbotapi.NewCallback(cb.ID, "")
		_, _ = BotAPI.Request(ack)
		return
	}

	cat := strings.TrimPrefix(cb.Data, "cat:")
	session.Category = cat
	catLabel := config.Categories[cat]
	if catLabel == "" {
		catLabel = cat
	}

	if cat == "serial" {
		session.FormatType = "multi"
		session.State = "description"
		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, fmt.Sprintf("Kategoriya: %s ✅", catLabel))
		_, _ = BotAPI.Send(edit)

		nextMsg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Tavsif (qisqacha) yuboring. O'tkazib yuborish uchun /skip yozing:")
		_, _ = BotAPI.Send(nextMsg)
	} else {
		session.State = "format_type"
		edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, fmt.Sprintf("Kategoriya: %s ✅", catLabel))
		_, _ = BotAPI.Send(edit)

		nextMsg := tgbotapi.NewMessage(cb.Message.Chat.ID, fmt.Sprintf(
			"<b>%s</b> formatini tanlang:\n\n• <b>1 ta to'liq qism</b> — oddiy film\n• <b>Ko'p qismli</b> — bir nechta qismli kino (masalan: Forsaj 1, 2, 3...)",
			catLabel,
		))
		nextMsg.ParseMode = "HTML"
		nextMsg.ReplyMarkup = FormatKeyboard()
		_, _ = BotAPI.Send(nextMsg)
	}

	ack := tgbotapi.NewCallback(cb.ID, "")
	_, _ = BotAPI.Request(ack)
}

func handleFormatCallback(cb *tgbotapi.CallbackQuery) {
	session := GetAdminSession(cb.From.ID)
	if session == nil || session.State != "format_type" {
		ack := tgbotapi.NewCallback(cb.ID, "")
		_, _ = BotAPI.Request(ack)
		return
	}

	fmtType := strings.TrimPrefix(cb.Data, "fmt:")
	session.FormatType = fmtType
	session.State = "description"

	fmtText := "1 ta to'liq qism (Film) 📼"
	if fmtType == "multi" {
		fmtText = "Ko'p qismli (1, 2, 3...) 🎞"
	}
	edit := tgbotapi.NewEditMessageText(cb.Message.Chat.ID, cb.Message.MessageID, fmt.Sprintf("Format: %s ✅", fmtText))
	_, _ = BotAPI.Send(edit)

	nextMsg := tgbotapi.NewMessage(cb.Message.Chat.ID, "Tavsif (qisqacha) yuboring. O'tkazib yuborish uchun /skip yozing:")
	_, _ = BotAPI.Send(nextMsg)

	ack := tgbotapi.NewCallback(cb.ID, "")
	_, _ = BotAPI.Request(ack)
}

func handleAdminSessionMessage(msg *tgbotapi.Message, session *AdminSession, cfg *config.Config) bool {
	text := strings.TrimSpace(msg.Text)

	if strings.ToLower(text) == "/cancel" || strings.ToLower(text) == "bekor qilish" {
		handleCancel(msg)
		return true
	}

	switch session.State {
	case "title":
		if text == "" {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Iltimos, kino nomini matn shaklida yuboring:")
			_, _ = BotAPI.Send(reply)
			return true
		}
		session.Title = text
		session.State = "category"
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Kategoriyani tanlang:")
		reply.ReplyMarkup = CategoryKeyboard()
		_, _ = BotAPI.Send(reply)
		return true

	case "description":
		if text == "/skip" {
			session.Description = ""
		} else {
			session.Description = text
		}
		session.State = "poster"
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Poster (rasm) yuboring. O'tkazib yuborish uchun /skip yozing:")
		_, _ = BotAPI.Send(reply)
		return true

	case "poster":
		var posterFileID string
		if text == "/skip" {
			posterFileID = ""
		} else if len(msg.Photo) > 0 {
			posterFileID = msg.Photo[len(msg.Photo)-1].FileID
		} else {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Iltimos, rasm yuboring yoki o'tkazib yuborish uchun /skip yozing:")
			_, _ = BotAPI.Send(reply)
			return true
		}

		session.PosterFileID = posterFileID

		if session.FormatType == "multi" {
			code, movieID, err := database.AddMultipartMovie(session.Title, session.Category, session.Description, posterFileID)
			if err != nil {
				reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Xatolik yuz berdi: %v", err))
				_, _ = BotAPI.Send(reply)
				ClearAdminSession(msg.From.ID)
				return true
			}
			session.MovieID = movieID
			session.Code = code
			session.EpisodesNum = 0
			session.State = "episode_loop"

			catLabel := config.Categories[session.Category]
			replyText := fmt.Sprintf(
				"✅ <b>%s</b> yaratildi!\n🎬 %s\n🔑 Kod: <b>%s</b>\n\nEndi qismlarni <b>birin-ketin video qilib</b> yuboravering — har bir video avtomatik keyingi qism (1-qism, 2-qism...) sifatida saqlanadi.\n\n<b>1-qism</b> videosini yuboring. Barcha qismlar tugagach — <b>/done</b> yozing.",
				catLabel, session.Title, code,
			)
			reply := tgbotapi.NewMessage(msg.Chat.ID, replyText)
			reply.ParseMode = "HTML"
			_, _ = BotAPI.Send(reply)
			return true
		}

		session.State = "video"
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Endi video faylni yuboring 🎥:")
		_, _ = BotAPI.Send(reply)
		return true

	case "video":
		if msg.Video == nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Iltimos, video fayl yuboring:")
			_, _ = BotAPI.Send(reply)
			return true
		}

		code, err := database.AddMovie(session.Title, session.Category, session.Description, session.PosterFileID, msg.Video.FileID)
		ClearAdminSession(msg.From.ID)
		if err != nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Xatolik: %v", err))
			_, _ = BotAPI.Send(reply)
			return true
		}

		replyText := fmt.Sprintf("✅ Kino qo'shildi!\n\n🎬 %s\nKod: <b>%s</b>", session.Title, code)
		reply := tgbotapi.NewMessage(msg.Chat.ID, replyText)
		reply.ParseMode = "HTML"
		_, _ = BotAPI.Send(reply)
		return true

	case "episode_loop":
		if text == "/done" {
			total := session.EpisodesNum
			title := session.Title
			code := session.Code
			ClearAdminSession(msg.From.ID)

			replyText := fmt.Sprintf("🎉 Tayyor! <b>%s</b> jami %d ta qism bilan muvaffaqiyatli saqlandi.\n🔑 Kod: <b>%s</b>", title, total, code)
			reply := tgbotapi.NewMessage(msg.Chat.ID, replyText)
			reply.ParseMode = "HTML"
			_, _ = BotAPI.Send(reply)
			return true
		}

		if msg.Video == nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, "Iltimos, navbatdagi qism videosini yuboring yoki yakunlash uchun /done yozing:")
			_, _ = BotAPI.Send(reply)
			return true
		}

		nextEp, err := database.AddEpisode(session.MovieID, msg.Video.FileID)
		if err != nil {
			reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Qism qo'shishda xatolik: %v", err))
			_, _ = BotAPI.Send(reply)
			return true
		}

		session.EpisodesNum = nextEp
		replyText := fmt.Sprintf("✅ <b>%d-qism</b> qo'shildi.\nKeyingi qism videosini yuboring yoki tugatish uchun /done yozing.", nextEp)
		reply := tgbotapi.NewMessage(msg.Chat.ID, replyText)
		reply.ParseMode = "HTML"
		_, _ = BotAPI.Send(reply)
		return true
	}

	return false
}

func handleAddEpisodeStart(msg *tgbotapi.Message, cfg *config.Config) {
	if !IsAdminUser(msg.From.ID, cfg) {
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) != 2 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Foydalanish: /addepisode 1234 (1234 — kino yoki serial kodi)")
		_, _ = BotAPI.Send(reply)
		return
	}

	code := parts[1]
	movie, err := database.GetMovieByCode(code)
	if err != nil || movie == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Bunday kodli kino yoki serial topilmadi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	currentEps, _ := database.EnsureMovieHasEpisodes(movie.ID)

	SetAdminSession(msg.From.ID, &AdminSession{
		State:       "episode_loop",
		MovieID:     movie.ID,
		Code:        movie.Code,
		Title:       movie.Title,
		EpisodesNum: currentEps,
	})

	catEmoji := "🎬"
	if movie.Category == "serial" {
		catEmoji = "📺"
	} else if movie.Category == "multfilm" {
		catEmoji = "🧸"
	}

	replyText := fmt.Sprintf(
		"%s <b>%s</b> (hozir %d ta qism mavjud)\n\nYangi qism videosini yuboring — <b>%d-qism</b> sifatida qo'shiladi.\nTugatgach — /done yozing.",
		catEmoji, movie.Title, currentEps, currentEps+1,
	)
	reply := tgbotapi.NewMessage(msg.Chat.ID, replyText)
	reply.ParseMode = "HTML"
	_, _ = BotAPI.Send(reply)
}

func handleDeleteMovie(msg *tgbotapi.Message, cfg *config.Config) {
	if !IsAdminUser(msg.From.ID, cfg) {
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) != 2 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Foydalanish: /delete 1234")
		_, _ = BotAPI.Send(reply)
		return
	}

	code := parts[1]
	ok, err := database.DeleteMovie(code)
	if err != nil || !ok {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Bunday kod topilmadi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, "✅ O'chirildi.")
	_, _ = BotAPI.Send(reply)
}

func handleStats(msg *tgbotapi.Message, cfg *config.Config) {
	if !IsAdminUser(msg.From.ID, cfg) {
		return
	}

	s, err := database.GetStats()
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("Statistika olishda xatolik: %v", err))
		_, _ = BotAPI.Send(reply)
		return
	}

	text := fmt.Sprintf(
		"📊 <b>Bot statistikasi</b>\n\n👥 Foydalanuvchilar: <b>%d</b>\n\n🎬 Kino: <b>%d</b>\n🧸 Multfilm: <b>%d</b>\n📺 Serial: <b>%d</b> (jami %d qism)\n\n👁 Jami ko'rishlar: <b>%d</b>",
		s.Users, s.Kino, s.Multfilm, s.Serial, s.Episodes, s.Views,
	)
	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ParseMode = "HTML"
	_, _ = BotAPI.Send(reply)
}

func handleUsers(msg *tgbotapi.Message, cfg *config.Config) {
	if !IsAdminUser(msg.From.ID, cfg) {
		return
	}

	total, _ := database.GetUserCount()
	users, _ := database.GetRecentUsers(30)
	if len(users) == 0 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Hozircha foydalanuvchilar yo'q.")
		_, _ = BotAPI.Send(reply)
		return
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("👥 <b>Jami foydalanuvchilar: %d</b>\nSo'nggi %d tasi:\n", total, len(users)))
	for _, u := range users {
		name := "Noma'lum"
		if u.FirstName != nil && *u.FirstName != "" {
			name = *u.FirstName
		}
		uname := ""
		if u.Username != nil && *u.Username != "" {
			uname = fmt.Sprintf(" (@%s)", *u.Username)
		}
		lines = append(lines, fmt.Sprintf("• %s%s — <code>%d</code>", name, uname, u.UserID))
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, strings.Join(lines, "\n"))
	reply.ParseMode = "HTML"
	_, _ = BotAPI.Send(reply)
}

func handleAdminsList(msg *tgbotapi.Message, cfg *config.Config) {
	if !IsAdminUser(msg.From.ID, cfg) {
		return
	}

	dbAdmins, _ := database.GetDBAdmins()
	var lines []string
	lines = append(lines, "👑 <b>Bot adminlari ro'yxati</b>\n")
	lines = append(lines, "📌 <b>Asosiy adminlar (env/config):</b>")
	for _, aid := range cfg.AdminIDs {
		lines = append(lines, fmt.Sprintf("• <code>%d</code>", aid))
	}
	lines = append(lines, "\n⭐ <b>Qo'shilgan adminlar (DB):</b>")
	if len(dbAdmins) == 0 {
		lines = append(lines, "<i>Hozircha qo'shimcha adminlar yo'q.</i>")
	} else {
		for _, a := range dbAdmins {
			name := "Noma'lum"
			if a.FirstName != nil && *a.FirstName != "" {
				name = *a.FirstName
			}
			uname := ""
			if a.Username != nil && *a.Username != "" {
				uname = fmt.Sprintf(" (@%s)", *a.Username)
			}
			lines = append(lines, fmt.Sprintf("• %s%s — <code>%d</code>", name, uname, a.UserID))
		}
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, strings.Join(lines, "\n"))
	reply.ParseMode = "HTML"
	_, _ = BotAPI.Send(reply)
}

func handleAddAdmin(msg *tgbotapi.Message, cfg *config.Config) {
	if !cfg.IsSuperAdmin(msg.From.ID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Bu komandani faqat asosiy (Super Admin) bera oladi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) != 2 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Foydalanish: /addadmin <user_id yoki @username>\n\nMasalan: /addadmin 123456789 yoki /addadmin @username")
		_, _ = BotAPI.Send(reply)
		return
	}

	target := parts[1]
	userInfo, err := database.GetUserByUsernameOrID(target)
	if err != nil || userInfo == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Foydalanuvchi topilmadi. Foydalanuvchi avval botga /start bosgan bo'lishi kerak.")
		_, _ = BotAPI.Send(reply)
		return
	}

	targetID := userInfo.UserID
	if IsAdminUser(targetID, cfg) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "ℹ️ Bu foydalanuvchi allaqachon admin.")
		_, _ = BotAPI.Send(reply)
		return
	}

	err = database.AddAdmin(targetID, msg.From.ID)
	if err != nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("❌ Admin qo'shishda bazada xatolik yuz berdi: %v", err))
		_, _ = BotAPI.Send(reply)
		return
	}

	UpdateUserCommands(targetID, cfg)

	nameStr := fmt.Sprintf("<code>%d</code>", targetID)
	if userInfo.FirstName != nil && *userInfo.FirstName != "" {
		nameStr = fmt.Sprintf("<b>%s</b>", *userInfo.FirstName)
	}

	reply := tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ %s muvaffaqiyatli admin etib tayinlandi!", nameStr))
	reply.ParseMode = "HTML"
	_, _ = BotAPI.Send(reply)
}

func handleDelAdmin(msg *tgbotapi.Message, cfg *config.Config) {
	if !cfg.IsSuperAdmin(msg.From.ID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Bu komandani faqat asosiy (Super Admin) bera oladi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	parts := strings.Fields(msg.Text)
	if len(parts) != 2 {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "Foydalanish: /deladmin <user_id yoki @username>")
		_, _ = BotAPI.Send(reply)
		return
	}

	target := parts[1]
	userInfo, err := database.GetUserByUsernameOrID(target)
	if err != nil || userInfo == nil {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Foydalanuvchi topilmadi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	targetID := userInfo.UserID
	if cfg.IsSuperAdmin(targetID) {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Asosiy adminni o'chirib bo'lmaydi.")
		_, _ = BotAPI.Send(reply)
		return
	}

	ok, err := database.RemoveAdmin(targetID)
	if err != nil || !ok {
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Bu foydalanuvchi bazadagi adminlar ro'yxatida yo'q.")
		_, _ = BotAPI.Send(reply)
		return
	}

	UpdateUserCommands(targetID, cfg)
	reply := tgbotapi.NewMessage(msg.Chat.ID, "✅ Adminlik huquqi olib tashlandi.")
	_, _ = BotAPI.Send(reply)
}
