import logging
from contextlib import asynccontextmanager

from aiogram.types import Update, BotCommandScopeDefault, BotCommandScopeChat
from fastapi import FastAPI, Request, HTTPException
from fastapi.responses import StreamingResponse, FileResponse
from fastapi.staticfiles import StaticFiles

import database as db
from bot_instance import bot, dp
from config import (
    WEBHOOK_PATH, WEBHOOK_URL, BASE_URL, ADMIN_USERNAME, ADMIN_IDS,
    USER_COMMANDS, ADMIN_COMMANDS, SUPER_ADMIN_COMMANDS
)
from handlers import user, admin
from security import validate_init_data

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

dp.include_router(admin.router)
dp.include_router(user.router)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await db.init_db()

    # Dynamic command scopes setting
    try:
        await bot.set_my_commands(USER_COMMANDS, scope=BotCommandScopeDefault())
        
        # Super Adminlar
        for super_aid in ADMIN_IDS:
            try:
                await bot.set_my_commands(SUPER_ADMIN_COMMANDS, scope=BotCommandScopeChat(chat_id=super_aid))
            except Exception as ex:
                logger.warning(f"Failed to set super admin commands for {super_aid}: {ex}")

        # DB Oddiy Adminlar
        db_admins = await db.get_db_admins()
        for a in db_admins:
            aid = a["user_id"]
            if aid not in ADMIN_IDS:
                try:
                    await bot.set_my_commands(ADMIN_COMMANDS, scope=BotCommandScopeChat(chat_id=aid))
                except Exception as ex:
                    logger.warning(f"Failed to set admin commands for {aid}: {ex}")
    except Exception as e:
        logger.error(f"Error setting bot commands: {e}")

    if BASE_URL:
        await bot.set_webhook(WEBHOOK_URL, drop_pending_updates=True)
        logger.info(f"Webhook o'rnatildi: {WEBHOOK_URL}")
    else:
        logger.warning("BASE_URL o'rnatilmagan — webhook faollashtirilmadi.")
    yield
    await db.close_db()
    await bot.session.close()


app = FastAPI(lifespan=lifespan)


@app.middleware("http")
async def add_no_cache_header(request: Request, call_next):
    response = await call_next(request)
    response.headers["Cache-Control"] = "no-cache, no-store, must-revalidate"
    response.headers["Pragma"] = "no-cache"
    response.headers["Expires"] = "0"
    return response


async def require_user(init_data: str) -> dict:
    user_data = validate_init_data(init_data)
    if not user_data:
        raise HTTPException(403, "Tekshiruvdan o'tmadi")
    try:
        await db.track_user(user_data["id"], user_data.get("first_name"), user_data.get("username"))
    except Exception as e:
        logger.warning(f"track_user xato (e'tiborsiz qoldirildi): {e}")
    return user_data


# ---------- Telegram webhook ----------
@app.post(WEBHOOK_PATH)
async def telegram_webhook(request: Request):
    try:
        data = await request.json()
        update = Update.model_validate(data)
        await dp.feed_update(bot, update)
    except Exception as e:
        logger.error(f"Webhook update ishlanishida xatolik: {e}", exc_info=True)
    return {"ok": True}


# ---------- Mini App meta ----------
@app.get("/api/meta")
async def api_meta():
    return {"admin_username": ADMIN_USERNAME}


# ---------- Kinolar ----------
@app.get("/api/movies")
async def api_movies(category: str | None = None, search: str | None = None,
                      search_type: str | None = None, sort: str = "new", limit: int | None = None):
    return await db.get_movies(category=category, search=search, search_type=search_type, sort=sort, limit=limit)


@app.get("/api/movie/{movie_id}")
async def api_movie(movie_id: int):
    movie = await db.get_movie_by_id(movie_id)
    if not movie:
        raise HTTPException(404, "Topilmadi")
    return movie


@app.get("/api/poster/{movie_id}")
async def api_poster(movie_id: int):
    movie = await db.get_movie_by_id(movie_id)
    if not movie or not movie["poster_file_id"]:
        raise HTTPException(404, "Poster yo'q")

    import os
    import asyncio

    file_id = movie["poster_file_id"]
    cache_dir = "static/posters"
    os.makedirs(cache_dir, exist_ok=True)
    cache_path = os.path.join(cache_dir, f"{file_id}.jpg")

    if os.path.exists(cache_path):
        return FileResponse(cache_path, media_type="image/jpeg")

    try:
        file = await bot.get_file(file_id)
        file_bytes = await bot.download_file(file.file_path)

        def save_file(path, data_bytes):
            with open(path, "wb") as f:
                f.write(data_bytes.read())

        await asyncio.to_thread(save_file, cache_path, file_bytes)
        return FileResponse(cache_path, media_type="image/jpeg")
    except Exception as e:
        logger.error(f"Poster caching error for movie {movie_id}: {e}")
        try:
            if 'file_bytes' in locals():
                file_bytes.seek(0)
                return StreamingResponse(file_bytes, media_type="image/jpeg")
        except Exception:
            pass
        raise HTTPException(500, f"Poster yuklashda xatolik: {e}")


@app.post("/api/watch/{movie_id}")
async def api_watch(movie_id: int, request: Request):
    body = await request.json()
    init_data = body.get("initData", "")
    logger.info(f"WATCH so'rovi: movie_id={movie_id} init_data_len={len(init_data)}")

    user_data = await require_user(init_data)

    movie = await db.get_movie_by_id(movie_id)
    if not movie:
        raise HTTPException(404, "Topilmadi")
    if movie["is_series"]:
        raise HTTPException(400, "Bu serial — qism tanlang")

    video_id = await db.get_movie_video(movie_id)
    chat_id = user_data["id"]
    caption = f"🎬 <b>{movie['title']}</b>\n\n{movie['description'] or ''}"
    try:
        await bot.send_video(chat_id, video_id, caption=caption)
    except Exception as e:
        logger.error(f"send_video xatosi: {e}")
        raise HTTPException(500, f"Video yuborishda xato: {e}")

    await db.increment_views(movie_id)
    await db.add_history(chat_id, movie_id)

    return {"ok": True}


# ---------- Serial qismlari ----------
@app.get("/api/episodes/{movie_id}")
async def api_episodes(movie_id: int):
    try:
        return await db.get_episodes(movie_id)
    except Exception as e:
        logger.error(f"episodes olishda xato: movie_id={movie_id} err={e}")
        raise HTTPException(500, f"Qismlarni olishda xato: {e}")


@app.post("/api/watch-episode/{episode_id}")
async def api_watch_episode(episode_id: int, request: Request):
    body = await request.json()
    user_data = await require_user(body.get("initData", ""))

    episode = await db.get_episode_by_id(episode_id)
    if not episode:
        raise HTTPException(404, "Topilmadi")

    chat_id = user_data["id"]
    caption = f"📺 <b>{episode['title']}</b> — {episode['episode_number']}-qism"
    try:
        await bot.send_video(chat_id, episode["video_file_id"], caption=caption)
    except Exception as e:
        logger.error(f"send_video xatosi: {e}")
        raise HTTPException(500, f"Video yuborishda xato: {e}")

    await db.increment_views(episode["movie_id"])
    await db.add_history(chat_id, episode["movie_id"])

    return {"ok": True}


# ---------- Sevimlilar ----------
@app.post("/api/favorite/{movie_id}")
async def api_toggle_favorite(movie_id: int, request: Request):
    body = await request.json()
    user_data = await require_user(body.get("initData", ""))

    movie = await db.get_movie_by_id(movie_id)
    if not movie:
        raise HTTPException(404, "Topilmadi")

    favorited = await db.toggle_favorite(user_data["id"], movie_id)
    return {"favorited": favorited}


@app.post("/api/favorites")
async def api_list_favorites(request: Request):
    body = await request.json()
    user_data = await require_user(body.get("initData", ""))
    ids = await db.get_favorite_ids(user_data["id"])
    movies = await db.get_favorite_movies(user_data["id"])
    return {"ids": ids, "movies": movies}


# ---------- Tarix / Profil ----------
@app.post("/api/history")
async def api_history(request: Request):
    body = await request.json()
    user_data = await require_user(body.get("initData", ""))
    movies = await db.get_history_movies(user_data["id"])
    count = await db.get_history_count(user_data["id"])
    return {"movies": movies, "count": count}


# ---------- Mini App static fayllar ----------
app.mount("/static", StaticFiles(directory="static"), name="static")


@app.get("/")
async def index():
    return FileResponse("static/index.html")


@app.api_route("/health", methods=["GET", "HEAD"])
async def health():
    return {"status": "ok"}
