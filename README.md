# 🎬 KinoApp — Telegram Kino/Serial Boti + Mini App

Kino, multfilm va (ko'p qismli) seriallarni to'liq Telegram ilovasi (Mini App) orqali
ko'rsatadigan bot. Admin videolarni botga yuboradi, foydalanuvchilar esa chiroyli
"kinoteatr" uslubidagi ilova orqali qidiradi, tomosha qiladi, sevimlilarga qo'shadi.

---

## ✨ Imkoniyatlar

**Foydalanuvchi uchun (Mini App):**
- 🏠 Bosh sahifa — "Yangi qo'shilganlar" va "Ommabop" karusellari + kategoriya bo'yicha filtr va qidiruv
- 🔥 Reyting — eng ko'p tomosha qilingan kino/seriallar
- ❤️ Sevimlilar — istalgan kino/serialni bir bosishda saqlab qo'yish
- 👤 Profil — ism, avatar, tomosha statistikasi, tomosha tarixi
- ✉️ Admin bilan bog'lanish — bir tugma orqali to'g'ridan-to'g'ri xabar yozish
- 📺 Serial qismlari — serial tanlanganda barcha qismlar ro'yxati chiqadi, istalganini tanlab tomosha qilish mumkin
- Kod orqali qidirish — chatga 4 xonali kodni yuborish kifoya

**Admin uchun (bot orqali):**
- `/addmovie` — kino/multfilm/serial qo'shish (bosqichma-bosqich)
- Serial tanlansa — barcha qismlarni **ketma-ket, bitta-bitta video yuborib** qo'shish (nom/tavsif/poster faqat bir marta so'raladi)
- `/addepisode <kod>` — mavjud serialga keyinroq yangi qismlar qo'shish (masalan, yangi fasl chiqqanda)
- `/delete <kod>` — kino yoki serialni butunlay o'chirish
- `/stats` — bot statistikasi (foydalanuvchilar soni, kino/multfilm/serial soni, jami ko'rishlar)
- `/users` — so'nggi 30 foydalanuvchi ro'yxati
- 🔑 Kino kodlari — oxirgi qo'shilgan kinolar va ularning kodlari ro'yxati

---

## 🏗 Arxitektura

- **Bot**: [aiogram 3](https://docs.aiogram.dev/) — Telegram bilan muloqot (webhook orqali)
- **Server**: [FastAPI](https://fastapi.tiangolo.com/) — Mini App'ga API va statik fayllarni xizmat qiladi, bot bilan bir xil processda ishlaydi
- **Ma'lumotlar bazasi**: [Turso](https://turso.tech) (bulutli SQLite) — Render'ning bepul serveri qayta ishga tushganda ham ma'lumotlar **hech qachon yo'qolmaydi**
- **Video saqlash**: videolarning o'zi Telegram serverlarida turadi (bot faqat `file_id` orqali ishlaydi) — shuning uchun bazada juda kam joy band bo'ladi
- **Frontend**: sof HTML/CSS/JavaScript (freymvorksiz), Telegram Web App SDK orqali

```
kinobot/
├── main.py              # FastAPI server: webhook, barcha API endpointlar
├── bot_instance.py       # aiogram Bot va Dispatcher
├── config.py             # Muhit o'zgaruvchilari (token, admin, baza)
├── database.py           # Turso bilan ishlash (kino, serial, qismlar, sevimlilar, tarix)
├── security.py           # Mini App so'rovlarini xavfsiz tekshirish (Telegram imzosi)
├── handlers/
│   ├── user.py            # /start, kod qidirish, admin bilan bog'lanish, kodlar ro'yxati
│   └── admin.py           # Kino/serial qo'shish, qism qo'shish, o'chirish
├── static/
│   ├── index.html          # Mini App tuzilishi
│   ├── style.css            # Kinoteatr uslubidagi dizayn (spotlight, animatsiyalar)
│   ├── app.js                # Mini App logikasi
│   └── banner.jpg             # /start xabaridagi banner rasm
├── requirements.txt
├── Procfile
└── render.yaml
```

---

## 🚀 O'rnatish va deploy qilish

### 1. Bot yaratish

1. Telegram'da **@BotFather** → `/newbot` → nom va username bering.
2. Berilgan **tokenni** saqlab qo'ying.

### 2. O'z Telegram ID raqamingizni bilish

**@userinfobot** ga `/start` yuboring — u sizga ID raqamingizni beradi.

### 3. Turso'da bepul baza yaratish

1. [turso.tech](https://turso.tech) → GitHub bilan kiring → **Create Database**.
2. Baza yaratilgach: **Database URL** (`libsql://...`) va yangi **Auth Token** (Authorization level: **Full Access**, Expires: **Never**) ni nusxalab oling.

### 4. Kodni GitHub'ga yuklash

Yangi repository yarating va shu papkadagi barcha fayllarni yuklang.

### 5. Render.com'da deploy qilish

1. [render.com](https://render.com) → **New +** → **Web Service** → repositoryingizni tanlang.
2. Sozlamalar:
   - **Runtime**: Python 3
   - **Build Command**: `pip install -r requirements.txt`
   - **Start Command**: `uvicorn main:app --host 0.0.0.0 --port $PORT`
   - **Plan**: Free
3. **Environment Variables** bo'limiga quyidagilarni qo'shing:

   | Key | Qiymat |
   |---|---|
   | `PYTHON_VERSION` | `3.11.9` |
   | `BOT_TOKEN` | BotFather'dan olingan token |
   | `ADMIN_IDS` | sizning Telegram ID (bir nechta bo'lsa vergul bilan) |
   | `ADMIN_USERNAME` | Telegram usernamengiz, `@` belgisisiz |
   | `BASE_URL` | dastlab bo'sh qoldiring |
   | `TURSO_DATABASE_URL` | Turso'dan olingan baza manzili |
   | `TURSO_AUTH_TOKEN` | Turso'dan olingan token |

4. **Create Web Service** — deploy tugagach, Render domeningizni beradi (masalan `https://kinobot-xxxx.onrender.com`).
5. Shu domenni nusxalab, `BASE_URL` qiymatiga qo'ying (oxirida `/` belgisisiz) → **Save Changes**.

### 6. Menu Button sozlash (Mini App'ni ochish tugmasi)

**@BotFather** → `/mybots` → botingiz → **Bot Settings** → **Menu Button** → **Configure menu button** → domeningizni yuboring → tugma nomini kiriting (masalan `Kinolar`).

### 7. Serverni "uxlab qolishdan" saqlash

[uptimerobot.com](https://uptimerobot.com) → **Add New Monitor** → HTTP(s) → URL: `https://sizning-domen.onrender.com/health` → interval: **5 daqiqa**.

---

## 🎬 Foydalanish

### Oddiy kino/multfilm qo'shish

`/addmovie` → nom → kategoriya (Kino/Multfilm) → tavsif → poster → video → tayyor, kod beriladi.

### Serial qo'shish

`/addmovie` → nom → kategoriya: **Serial** → tavsif → poster → so'ng bot qism-qism video so'raydi:

```
1-qism videosini yuboring.
✅ 1-qism qo'shildi. Yana yuboring yoki /done yozing.
```

Videolarni ketma-ket yuboraverasiz, har biri avtomatik keyingi qism bo'lib qo'shiladi.
Tugagach `/done`.

### Mavjud serialga yangi qism qo'shish

```
/addepisode 1234
```

(`1234` — serial kodi) — bot davom ettirib, yangi qismlarni qabul qiladi.

### O'chirish

```
/delete 1234
```

---

## 🔧 Muhim eslatmalar

- Render'ning **bepul** tarifi 15 daqiqa foydalanilmasa "uxlaydi" — shuning uchun UptimeRobot sozlash tavsiya etiladi.
- Ma'lumotlar bazasi **Turso**'da saqlanadi, shuning uchun kodni necha marta yangilasangiz ham (`git push`, qayta deploy) kino/serial ma'lumotlari **yo'qolmaydi**.
- Video fayllarning o'zi Telegram serverlarida qoladi — bazada faqat matn (`file_id`) saqlanadi, shuning uchun minglab kino qo'shsangiz ham joy muammosi bo'lmaydi.
