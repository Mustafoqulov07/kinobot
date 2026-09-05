# 🎬 KinoApp — Telegram Kino/Serial Boti + Mini App (Go / Golang)

Kino, multfilm va (ko'p qismli) seriallarni to'liq Telegram ilovasi (Mini App) orqali ko'rsatadigan yuqori unumdor bot.
Backend to'liq **Go (Golang)** tiliga ko'chirilgan bo'lib, minimal RAM sarfi (15-25 MB) va chaqqon ishlash tezligini ta'minlaydi.

---

## ✨ Imkoniyatlar

**Foydalanuvchi uchun (Mini App):**
- 🏠 Bosh sahifa — "Yangi qo'shilganlar" va "Ommabop" karusellari + kategoriya bo'yicha filtr va aqlli qidiruv
- 🔍 Qidiruv — Kino nomi va 4 xonali kodi bo'yicha tezkor, moslik darajasiga ko'ra tartiblangan qidiruv
- 🔥 Reyting — Eng ko'p tomosha qilingan kino va seriallar
- ❤️ Sevimlilar — Istalgan kinoni bir bosishda saqlab qo'yish
- 👤 Profil — Ism, avatar, tomosha statistikasi va tomosha tarixi
- ✉️ Admin bilan bog'lanish — Bir tugma orqali to'g'ridan-to'g'ri bog'lanish
- 📺 Ko'p qismli kino va seriallar — Qismlar ro'yxati chiqadi, istalgan qismni tanlab tomosha qilish mumkin
- 🔑 4 xonali kod orqali qidirish — Chatga kodni yuborish orqali darhol video olish

**Admin uchun (Telegram bot orqali):**
- `/addmovie` — Kino, multfilm yoki serial qo'shish (1 ta to'liq film yoki 1, 2, 3... ko'p qismli)
- Ko'p qismli tanlansa — Barcha qismlarni ketma-ket, bitta-bitta video yuborib kiritish (`/done` bilan tugatish)
- `/addepisode <kod>` — Mavjud kino yoki serialga yangi qismlarni qo'shish
- `/addadmin <user_id yoki @username>` — Yangi adminga huquq berish
- `/deladmin <user_id yoki @username>` — Adminlik huquqini olib tashlash
- `/admins` — Barcha adminlar ro'yxati
- `/delete <kod>` — Kino yoki serialni bazadan to'liq o'chirish
- `/stats` — Bot statistikasi (foydalanuvchilar, kinolar, seriallar, jami ko'rishlar)
- `/users` — So'nggi 30 foydalanuvchi ro'yxati
- `/codes` — Oxirgi qo'shilgan kinolar va ularning kodlari

---

## 🏗 Arxitektura

- **Til va Runtime**: Go (Golang)
- **Telegram Bot**: `github.com/go-telegram-bot-api/telegram-bot-api/v5`
- **HTTP Web Server**: Go standart kutubxonasi (`net/http.ServeMux` - Go 1.22+)
- **Ma'lumotlar bazasi**: [Turso](https://turso.tech) (`github.com/tursodatabase/libsql-client-go`) + lokal rejimda sof Go SQLite
- **Xavfsizlik**: Mini App so'rovlarini HMAC-SHA256 Telegram imzosi orqali to'liq tekshirish
- **Frontend**: Sof HTML/CSS/JavaScript (Vanilla JS) + Telegram Web App SDK

```
kinobot/
├── main.go               # Server kirish nuqtasi (Entrypoint)
├── config/
│   └── config.go         # Muhit o'zgaruvchilari (Token, Turso, Adminlar)
├── database/
│   └── database.go       # Turso DB operatsiyalari, qidiruv va statistika
├── security/
│   └── security.go       # Telegram Mini App initData tekshiruvi (HMAC-SHA256)
├── bot/
│   ├── bot.go            # Botni sozlash, webhook va xabarlar
│   ├── handlers.go       # Foydalanuvchi buyruqlari, qidiruv, callbacklar
│   └── admin.go          # Admin interaktiv holatlari (FSM), kino/serial qo'shish
├── server/
│   └── server.go         # REST API endpointlari va statik fayllar
├── static/               # Mini App interfeysi (HTML, CSS, JS, banner)
├── render.yaml           # Render Go konfiguratsiyasi
├── Procfile              # Render start komandasi
└── go.mod / go.sum       # Go modullari
```

---

## 🚀 Render.com'da sozlash

1. [render.com](https://render.com) boshqaruv paneliga kiring.
2. Web Service sozlamalarida quyidagilarni o'rnating:
   - **Environment / Runtime**: `Go`
   - **Build Command**: `go build -o kinobot .`
   - **Start Command**: `./kinobot`
3. **Environment Variables**:
   - `BOT_TOKEN`
   - `ADMIN_IDS`
   - `ADMIN_USERNAME`
   - `BASE_URL` (masalan: `https://kinobot-xxx.onrender.com`)
   - `TURSO_DATABASE_URL`
   - `TURSO_AUTH_TOKEN`

### UptimeRobot sozlash (Server uxlamasligi uchun):
- Monitor Type: `HTTP(s)`
- URL: `https://sizning-domen.onrender.com/api/ping` (yoki `/health`)
- Interval: **5-10 daqiqa**
