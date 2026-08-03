import os

# Render'da "Environment" bo'limiga qo'shiladigan sozlamalar
BOT_TOKEN = os.getenv("BOT_TOKEN", "")

# Admin(lar) Telegram ID raqami(lari). Bir nechta bo'lsa vergul bilan ajrating: "12345,67890"
ADMIN_IDS = [int(x) for x in os.getenv("ADMIN_IDS", "").split(",") if x.strip().isdigit()]

# Render avtomatik beradigan domen, masalan: https://kinobot.onrender.com
BASE_URL = os.getenv("BASE_URL", "").rstrip("/")

WEBHOOK_PATH = f"/webhook/{BOT_TOKEN}"
WEBHOOK_URL = f"{BASE_URL}{WEBHOOK_PATH}"

# Admin bilan bog'lanish tugmasi uchun (@ belgisisiz, masalan: "mustafoqulov")
ADMIN_USERNAME = os.getenv("ADMIN_USERNAME", "")

# Turso (bulutli SQLite) - ma'lumotlar doimiy saqlanishi uchun
TURSO_DATABASE_URL = os.getenv("TURSO_DATABASE_URL", "")
TURSO_AUTH_TOKEN = os.getenv("TURSO_AUTH_TOKEN", "")

from aiogram.types import BotCommand

CATEGORIES = {
    "kino": "🎬 Kino",
    "multfilm": "🧸 Multfilm",
    "serial": "📺 Serial",
}

USER_COMMANDS = [
    BotCommand(command="start", description="Botni ishga tushirish va Mini App"),
    BotCommand(command="codes", description="🔑 Kinolar va ularning kodlari"),
]

# Oddiy adminlar uchun komandalar (Faqat kino boshqaruvi)
ADMIN_COMMANDS = [
    BotCommand(command="start", description="Botni ishga tushirish va Mini App"),
    BotCommand(command="addmovie", description="🎬 Yangi kino/serial/multfilm qo'shish"),
    BotCommand(command="addepisode", description="📺 Serialga yangi qism qo'shish"),
    BotCommand(command="delete", description="❌ Kinoni o'chirish"),
    BotCommand(command="stats", description="📊 Bot statistikasi"),
    BotCommand(command="users", description="👥 Foydalanuvchilar ro'yxati"),
    BotCommand(command="codes", description="🔑 Kinolar va ularning kodlari"),
]

# Asosiy (Super) Adminlar uchun komandalar (Barcha imkoniyatlar)
SUPER_ADMIN_COMMANDS = [
    BotCommand(command="start", description="Botni ishga tushirish va Mini App"),
    BotCommand(command="addmovie", description="🎬 Yangi kino/serial/multfilm qo'shish"),
    BotCommand(command="addepisode", description="📺 Serialga yangi qism qo'shish"),
    BotCommand(command="delete", description="❌ Kinoni o'chirish"),
    BotCommand(command="stats", description="📊 Bot statistikasi"),
    BotCommand(command="users", description="👥 Foydalanuvchilar ro'yxati"),
    BotCommand(command="admins", description="👑 Adminlar ro'yxati"),
    BotCommand(command="addadmin", description="⭐ Yangi admin qo'shish"),
    BotCommand(command="deladmin", description="🗑 Adminni olib tashlash"),
    BotCommand(command="codes", description="🔑 Kinolar va ularning kodlari"),
]

