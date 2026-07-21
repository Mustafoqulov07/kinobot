from aiogram import Router, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from aiogram.types import Message, InlineKeyboardMarkup, InlineKeyboardButton, CallbackQuery

from config import ADMIN_IDS, CATEGORIES
import database as db

router = Router()


def is_admin(user_id: int) -> bool:
    return user_id in ADMIN_IDS


class AddMovie(StatesGroup):
    title = State()
    category = State()
    description = State()
    poster = State()
    video = State()          # kino/multfilm uchun — bitta video
    episode_loop = State()   # serial uchun — ketma-ket qismlar


def category_keyboard() -> InlineKeyboardMarkup:
    buttons = [
        [InlineKeyboardButton(text=label, callback_data=f"cat:{key}")]
        for key, label in CATEGORIES.items()
    ]
    return InlineKeyboardMarkup(inline_keyboard=buttons)


@router.message(Command("addmovie"))
async def add_movie_start(message: Message, state: FSMContext):
    if not is_admin(message.from_user.id):
        return
    await state.set_state(AddMovie.title)
    await message.answer("🎬 Yangi kino/serial nomini yuboring:")


@router.message(AddMovie.title)
async def add_movie_title(message: Message, state: FSMContext):
    await state.update_data(title=message.text)
    await state.set_state(AddMovie.category)
    await message.answer("Kategoriyani tanlang:", reply_markup=category_keyboard())


@router.callback_query(AddMovie.category, F.data.startswith("cat:"))
async def add_movie_category(callback: CallbackQuery, state: FSMContext):
    category = callback.data.split(":")[1]
    await state.update_data(category=category)
    await state.set_state(AddMovie.description)
    await callback.message.edit_text(f"Kategoriya: {CATEGORIES[category]} ✅")
    await callback.message.answer("Tavsif (qisqacha) yuboring. O'tkazib yuborish uchun /skip yozing:")
    await callback.answer()


@router.message(AddMovie.description, Command("skip"))
async def add_movie_desc_skip(message: Message, state: FSMContext):
    await state.update_data(description="")
    await state.set_state(AddMovie.poster)
    await message.answer("Poster (rasm) yuboring. O'tkazib yuborish uchun /skip yozing:")


@router.message(AddMovie.description)
async def add_movie_desc(message: Message, state: FSMContext):
    await state.update_data(description=message.text)
    await state.set_state(AddMovie.poster)
    await message.answer("Poster (rasm) yuboring. O'tkazib yuborish uchun /skip yozing:")


async def _finish_poster_step(message: Message, state: FSMContext, poster_file_id: str | None):
    data = await state.get_data()
    await state.update_data(poster_file_id=poster_file_id)

    if data["category"] == "serial":
        # Serial: yangi yozuv yaratamiz va darhol qism qo'shish rejimiga o'tamiz
        code, movie_id = await db.add_series(data["title"], data.get("description", ""), poster_file_id)
        await state.update_data(movie_id=movie_id, code=code, episodes_added=0)
        await state.set_state(AddMovie.episode_loop)
        await message.answer(
            f"✅ Serial yaratildi!\n🎬 {data['title']}\nKod: <b>{code}</b>\n\n"
            "Endi qismlarni <b>birin-ketin</b> yuboraveringizmumkin — har bir video avtomatik "
            "keyingi qism sifatida qo'shiladi.\n\n"
            "1-qism videosini yuboring. Tugatgach — <b>/done</b> yozing."
        )
    else:
        await state.set_state(AddMovie.video)
        await message.answer("Endi video faylni yuboring 🎥:")


@router.message(AddMovie.poster, Command("skip"))
async def add_movie_poster_skip(message: Message, state: FSMContext):
    await _finish_poster_step(message, state, None)


@router.message(AddMovie.poster, F.photo)
async def add_movie_poster(message: Message, state: FSMContext):
    await _finish_poster_step(message, state, message.photo[-1].file_id)


@router.message(AddMovie.video, F.video)
async def add_movie_video(message: Message, state: FSMContext):
    data = await state.get_data()
    code = await db.add_movie(
        title=data["title"],
        category=data["category"],
        description=data.get("description", ""),
        poster_file_id=data.get("poster_file_id"),
        video_file_id=message.video.file_id,
    )
    await state.clear()
    await message.answer(f"✅ Kino qo'shildi!\n\n🎬 {data['title']}\nKod: <b>{code}</b>")


# ---------- Serial: qismlarni ketma-ket qo'shish ----------
@router.message(AddMovie.episode_loop, F.video)
async def add_episode_loop(message: Message, state: FSMContext):
    data = await state.get_data()
    ep_num = await db.add_episode(data["movie_id"], message.video.file_id)
    await state.update_data(episodes_added=ep_num)
    await message.answer(
        f"✅ <b>{ep_num}-qism</b> qo'shildi.\n"
        "Yana video yuboring yoki tugatish uchun /done yozing."
    )


@router.message(AddMovie.episode_loop, Command("done"))
async def add_episode_done(message: Message, state: FSMContext):
    data = await state.get_data()
    total = data.get("episodes_added", 0)
    await state.clear()
    await message.answer(
        f"🎉 Tayyor! <b>{data['title']}</b> seriali {total} ta qism bilan saqlandi.\n"
        f"Kod: <b>{data['code']}</b>"
    )


# ---------- Mavjud serialga yangi qismlar qo'shish ----------
@router.message(Command("addepisode"))
async def add_episode_to_existing_start(message: Message, state: FSMContext):
    if not is_admin(message.from_user.id):
        return
    parts = message.text.split(maxsplit=1)
    if len(parts) != 2:
        await message.answer("Foydalanish: /addepisode 1234 (1234 — serial kodi)")
        return

    movie = await db.get_movie_by_code(parts[1].strip())
    if not movie or movie["category"] != "serial":
        await message.answer("❌ Bunday kodli serial topilmadi.")
        return

    await state.update_data(
        movie_id=movie["id"], code=movie["code"], title=movie["title"],
        episodes_added=movie["episode_count"],
    )
    await state.set_state(AddMovie.episode_loop)
    await message.answer(
        f"🎬 <b>{movie['title']}</b> (hozir {movie['episode_count']} qism bor)\n\n"
        f"Yangi qism videosini yuboring — {movie['episode_count'] + 1}-qism sifatida qo'shiladi.\n"
        "Tugatgach — /done yozing."
    )


# ---------- Statistika ----------
@router.message(Command("stats"))
async def stats_handler(message: Message):
    if not is_admin(message.from_user.id):
        return
    s = await db.get_stats()
    text = (
        "📊 <b>Bot statistikasi</b>\n\n"
        f"👥 Foydalanuvchilar: <b>{s['users']}</b>\n\n"
        f"🎬 Kino: <b>{s['kino']}</b>\n"
        f"🧸 Multfilm: <b>{s['multfilm']}</b>\n"
        f"📺 Serial: <b>{s['serial']}</b> (jami {s['episodes']} qism)\n\n"
        f"👁 Jami ko'rishlar: <b>{s['views']}</b>"
    )
    await message.answer(text)


@router.message(Command("users"))
async def users_handler(message: Message):
    if not is_admin(message.from_user.id):
        return
    total = await db.get_user_count()
    users = await db.get_recent_users(limit=30)
    if not users:
        await message.answer("Hozircha foydalanuvchilar yo'q.")
        return

    lines = [f"👥 <b>Jami foydalanuvchilar: {total}</b>\nSo'nggi {len(users)} tasi:\n"]
    for u in users:
        name = u["first_name"] or "Noma'lum"
        uname = f" (@{u['username']})" if u["username"] else ""
        lines.append(f"• {name}{uname} — <code>{u['user_id']}</code>")
    await message.answer("\n".join(lines))


# ---------- O'chirish ----------
@router.message(Command("delete"))
async def delete_movie_handler(message: Message):
    if not is_admin(message.from_user.id):
        return
    parts = message.text.split()
    if len(parts) != 2:
        await message.answer("Foydalanish: /delete 1234")
        return
    ok = await db.delete_movie(parts[1])
    await message.answer("✅ O'chirildi." if ok else "❌ Bunday kod topilmadi.")
