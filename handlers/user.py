from aiogram import Router, F
from aiogram.filters import CommandStart
from aiogram.types import (
    Message, FSInputFile, KeyboardButton, ReplyKeyboardMarkup,
    InlineKeyboardMarkup, InlineKeyboardButton, CallbackQuery,
)

from config import ADMIN_USERNAME, CATEGORIES
import database as db

router = Router()

WELCOME_TEXT = (
    "🎬 <b>KinoApp</b>ga xush kelibsiz!\n\n"
    "Bu yerda minglab kino, multfilm va seriallarni bir joydan topib, "
    "to'g'ridan-to'g'ri shu yerda tomosha qilishingiz mumkin.\n\n"
    "👆 Yozish maydoni yonidagi <b>Kinolar</b> tugmasi orqali ilovani oching!"
)

BTN_ADMIN = "✉️ Admin bilan bog'lanish"
BTN_CODES = "🔑 Kino kodlari"


def main_keyboard() -> ReplyKeyboardMarkup:
    return ReplyKeyboardMarkup(
        keyboard=[[KeyboardButton(text=BTN_ADMIN), KeyboardButton(text=BTN_CODES)]],
        resize_keyboard=True,
    )


@router.message(CommandStart())
async def start_handler(message: Message):
    try:
        await db.track_user(message.from_user.id, message.from_user.first_name, message.from_user.username)
    except Exception:
        pass
    banner = FSInputFile("static/banner.jpg")
    await message.answer_photo(
        banner,
        caption=WELCOME_TEXT,
        reply_markup=main_keyboard(),
    )


@router.message(F.text == BTN_ADMIN)
async def contact_admin_handler(message: Message):
    if not ADMIN_USERNAME:
        await message.answer("Admin hozircha sozlanmagan.")
        return
    kb = InlineKeyboardMarkup(
        inline_keyboard=[[InlineKeyboardButton(text="✉️ Yozish", url=f"https://t.me/{ADMIN_USERNAME}")]]
    )
    await message.answer("Savol yoki taklifingiz bo'lsa, admin bilan bog'laning 👇", reply_markup=kb)


@router.message(F.text == BTN_CODES)
async def codes_list_handler(message: Message):
    movies = await db.get_movies(sort="new", limit=30)
    if not movies:
        await message.answer("Hozircha kinolar mavjud emas.")
        return

    lines = ["🔑 <b>So'nggi qo'shilgan kinolar va kodlari:</b>\n"]
    for m in movies:
        emoji = CATEGORIES.get(m["category"], "🎬").split()[0]
        extra = f" ({m['episode_count']} qism)" if m["is_series"] else ""
        lines.append(f"{emoji} {m['title']}{extra} — <b>{m['code']}</b>")
    lines.append("\nKodni shu chatga yuborsangiz, video darhol keladi.")
    await message.answer("\n".join(lines))


EPISODE_CHUNK = 20  # bitta "sahifa"da nechta qism ko'rsatilishi


def build_episode_keyboard(episodes: list, movie_id: int, range_index: int | None = None) -> InlineKeyboardMarkup:
    """
    Qismlar soni ko'p bo'lsa (masalan 100+), avval "1-20", "21-40" kabi
    guruhlarni ko'rsatadi. Guruh tanlanganda o'sha oraliqdagi qismlarni chiqaradi.
    """
    if len(episodes) <= EPISODE_CHUNK or range_index is not None:
        subset = episodes
        if len(episodes) > EPISODE_CHUNK and range_index is not None:
            start = range_index * EPISODE_CHUNK
            subset = episodes[start:start + EPISODE_CHUNK]

        buttons = [
            InlineKeyboardButton(text=f"{ep['episode_number']}-qism", callback_data=f"ep:{ep['id']}")
            for ep in subset
        ]
        rows = [buttons[i:i + 4] for i in range(0, len(buttons), 4)]
        if len(episodes) > EPISODE_CHUNK:
            rows.append([InlineKeyboardButton(text="⬅️ Orqaga", callback_data=f"epback:{movie_id}")])
        return InlineKeyboardMarkup(inline_keyboard=rows)

    range_count = (len(episodes) + EPISODE_CHUNK - 1) // EPISODE_CHUNK
    range_buttons = []
    for i in range(range_count):
        start_ep = episodes[i * EPISODE_CHUNK]["episode_number"]
        end_idx = min((i + 1) * EPISODE_CHUNK, len(episodes)) - 1
        end_ep = episodes[end_idx]["episode_number"]
        range_buttons.append(
            InlineKeyboardButton(text=f"{start_ep}-{end_ep}", callback_data=f"eprange:{movie_id}:{i}")
        )
    rows = [range_buttons[i:i + 3] for i in range(0, len(range_buttons), 3)]
    return InlineKeyboardMarkup(inline_keyboard=rows)


@router.message(F.text.regexp(r"^\d{4}$"))
async def code_search_handler(message: Message):
    movie = await db.get_movie_by_code(message.text.strip())
    if not movie:
        await message.answer("❌ Bu kodga mos kino topilmadi. Kodni tekshirib qayta yuboring.")
        return

    if movie["is_series"]:
        episodes = await db.get_episodes(movie["id"])
        if not episodes:
            await message.answer("Bu serialga hali qismlar qo'shilmagan.")
            return
        kb = build_episode_keyboard(episodes, movie["id"])
        hint = "Guruhni tanlang:" if len(episodes) > EPISODE_CHUNK else "Qismni tanlang:"
        caption = f"📺 <b>{movie['title']}</b>\n\n{movie['description'] or ''}\n\n{hint}"
        await message.answer(caption, reply_markup=kb)
        return

    await db.increment_views(movie["id"])
    video_id = await db.get_movie_video(movie["id"])
    caption = f"🎬 <b>{movie['title']}</b>\n\n{movie['description'] or ''}\n\nKod: {movie['code']}"
    await message.answer_video(video_id, caption=caption)


@router.callback_query(F.data.startswith("eprange:"))
async def episode_range_callback(callback: CallbackQuery):
    _, movie_id_str, idx_str = callback.data.split(":")
    movie_id, idx = int(movie_id_str), int(idx_str)
    episodes = await db.get_episodes(movie_id)
    kb = build_episode_keyboard(episodes, movie_id, range_index=idx)
    await callback.message.edit_reply_markup(reply_markup=kb)
    await callback.answer()


@router.callback_query(F.data.startswith("epback:"))
async def episode_back_callback(callback: CallbackQuery):
    movie_id = int(callback.data.split(":")[1])
    episodes = await db.get_episodes(movie_id)
    kb = build_episode_keyboard(episodes, movie_id, range_index=None)
    await callback.message.edit_reply_markup(reply_markup=kb)
    await callback.answer()


@router.callback_query(F.data.startswith("ep:"))
async def episode_callback_handler(callback: CallbackQuery):
    episode_id = int(callback.data.split(":")[1])
    episode = await db.get_episode_by_id(episode_id)
    if not episode:
        await callback.answer("Topilmadi", show_alert=True)
        return

    await db.increment_views(episode["movie_id"])
    caption = f"📺 <b>{episode['title']}</b> — {episode['episode_number']}-qism"
    await callback.message.answer_video(episode["video_file_id"], caption=caption)
    await callback.answer()
