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

CATEGORIES = {
    "kino": "🎬 Kino",
    "multfilm": "🧸 Multfilm",
    "serial": "📺 Serial",
}
