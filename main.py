import logging
from contextlib import asynccontextmanager

from aiogram.types import Update
from fastapi import FastAPI, Request, HTTPException
from fastapi.responses import StreamingResponse, FileResponse
from fastapi.staticfiles import StaticFiles

import database as db
from bot_instance import bot, dp
from config import WEBHOOK_PATH, WEBHOOK_URL, BASE_URL, ADMIN_USERNAME
from handlers import user, admin
from security import validate_init_data

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

dp.include_router(admin.router)
dp.include_router(user.router)


@asynccontextmanager
async def lifespan(app: FastAPI):
    await db.init_db()
    if BASE_URL:
        await bot.set_webhook(WEBHOOK_URL, drop_pending_updates=True)
        logger.info(f"Webhook o'rnatildi: {WEBHOOK_URL}")
    else:
        logger.warning("BASE_URL o'rnatilmagan — webhook faollashtirilmadi.")
    yield
    await db.close_db()
    await bot.session.close()


app = FastAPI(lifespan=lifespan)


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
    data = await request.json()
    update = Update.model_validate(data)
    await dp.feed_update(bot, update)
    return {"ok": True}


# ---------- Mini App meta ----------
@app.get("/api/meta")
async def api_meta():
    return {"admin_username": ADMIN_USERNAME}


# ---------- Kinolar ----------
@app.get("/api/movies")
async def api_movies(category: str | None = None, search: str | None = None,
                      sort: str = "new", limit: int | None = None):
    return await db.get_movies(category=category, search=search, sort=sort, limit=limit)


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

    file = await bot.get_file(movie["poster_file_id"])
    file_bytes = await bot.download_file(file.file_path)
    return StreamingResponse(file_bytes, media_type="image/jpeg")


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
