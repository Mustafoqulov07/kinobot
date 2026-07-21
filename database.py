import asyncio
import random

import libsql

from config import TURSO_DATABASE_URL, TURSO_AUTH_TOKEN


def _raw_execute(sql: str, params: tuple = ()):
    """Har bir chaqiruvda yangi ulanish ochadi (thread-xavfsizlik uchun)."""
    conn = libsql.connect(database=TURSO_DATABASE_URL, auth_token=TURSO_AUTH_TOKEN)
    try:
        cur = conn.execute(sql, params)
        conn.commit()
        try:
            rows = cur.fetchall()
        except Exception:
            rows = []
        return rows
    finally:
        conn.close()


async def _execute(sql: str, params: tuple = ()):
    return await asyncio.to_thread(_raw_execute, sql, params)


CREATE_STATEMENTS = [
    """
    CREATE TABLE IF NOT EXISTS movies (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        code TEXT UNIQUE NOT NULL,
        title TEXT NOT NULL,
        category TEXT NOT NULL,
        description TEXT DEFAULT '',
        poster_file_id TEXT,
        video_file_id TEXT,
        views INTEGER DEFAULT 0,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS movie_episodes (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        movie_id INTEGER NOT NULL,
        episode_number INTEGER NOT NULL,
        video_file_id TEXT NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS favorites (
        user_id INTEGER NOT NULL,
        movie_id INTEGER NOT NULL,
        created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (user_id, movie_id)
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS history (
        user_id INTEGER NOT NULL,
        movie_id INTEGER NOT NULL,
        watched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        PRIMARY KEY (user_id, movie_id)
    )
    """,
    """
    CREATE TABLE IF NOT EXISTS bot_users (
        user_id INTEGER PRIMARY KEY,
        first_name TEXT,
        username TEXT,
        joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
        last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
    )
    """,
]

MOVIE_SELECT = """
    SELECT m.id, m.code, m.title, m.category, m.description, m.poster_file_id, m.views,
           (SELECT COUNT(*) FROM movie_episodes e WHERE e.movie_id = m.id) AS episode_count
    FROM movies m
"""


def _row_to_movie(r) -> dict:
    return {
        "id": r[0], "code": r[1], "title": r[2], "category": r[3],
        "description": r[4], "poster_file_id": r[5], "views": r[6],
        "episode_count": r[7], "is_series": r[3] == "serial",
    }


async def _table_columns(table: str) -> set:
    rows = await _execute(f"PRAGMA table_info({table})")
    return {r[1] for r in rows}


async def _ensure_movies_video_nullable():
    rows = await _execute("PRAGMA table_info(movies)")
    for r in rows:
        # r: (cid, name, type, notnull, dflt_value, pk)
        if r[1] == "video_file_id" and r[3] == 1:
            await _execute(
                """
                CREATE TABLE movies_new (
                    id INTEGER PRIMARY KEY,
                    code TEXT UNIQUE NOT NULL,
                    title TEXT NOT NULL,
                    category TEXT NOT NULL,
                    description TEXT DEFAULT '',
                    poster_file_id TEXT,
                    video_file_id TEXT,
                    views INTEGER DEFAULT 0,
                    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
                )
                """
            )
            await _execute(
                """
                INSERT INTO movies_new (id, code, title, category, description, poster_file_id, video_file_id, views, created_at)
                SELECT id, code, title, category, description, poster_file_id, video_file_id, views, created_at FROM movies
                """
            )
            await _execute("DROP TABLE movies")
            await _execute("ALTER TABLE movies_new RENAME TO movies")
            break


async def init_db():
    for stmt in CREATE_STATEMENTS:
        await _execute(stmt)

    # Eski/yarim yaratilgan bazalarda ustun yetishmasa — avtomatik qo'shib qo'yamiz
    try:
        movie_cols = await _table_columns("movies")
        if "views" not in movie_cols:
            await _execute("ALTER TABLE movies ADD COLUMN views INTEGER DEFAULT 0")

        await _ensure_movies_video_nullable()
    except Exception as e:
        print(f"Sxema tekshiruvida ogohlantirish: {e}")


async def generate_unique_code() -> str:
    while True:
        code = str(random.randint(1000, 9999))
        rows = await _execute("SELECT id FROM movies WHERE code = ?", (code,))
        if not rows:
            return code


# ---------- Kino / Multfilm (bitta video) ----------
async def add_movie(title, category, description, poster_file_id, video_file_id) -> str:
    code = await generate_unique_code()
    await _execute(
        """INSERT INTO movies (code, title, category, description, poster_file_id, video_file_id)
           VALUES (?, ?, ?, ?, ?, ?)""",
        (code, title, category, description or "", poster_file_id, video_file_id),
    )
    return code


# ---------- Serial (ko'p qismli) ----------
async def add_series(title, description, poster_file_id) -> tuple[str, int]:
    code = await generate_unique_code()
    await _execute(
        """INSERT INTO movies (code, title, category, description, poster_file_id, video_file_id)
           VALUES (?, ?, 'serial', ?, ?, NULL)""",
        (code, title, description or "", poster_file_id),
    )
    rows = await _execute("SELECT id FROM movies WHERE code = ?", (code,))
    return code, rows[0][0]


async def add_episode(movie_id: int, video_file_id: str) -> int:
    rows = await _execute(
        "SELECT COALESCE(MAX(episode_number), 0) FROM movie_episodes WHERE movie_id = ?", (movie_id,)
    )
    next_num = (rows[0][0] or 0) + 1
    await _execute(
        "INSERT INTO movie_episodes (movie_id, episode_number, video_file_id) VALUES (?, ?, ?)",
        (movie_id, next_num, video_file_id),
    )
    return next_num


async def get_episodes(movie_id: int):
    rows = await _execute(
        "SELECT id, episode_number FROM movie_episodes WHERE movie_id = ? ORDER BY episode_number",
        (movie_id,),
    )
    return [{"id": r[0], "episode_number": r[1]} for r in rows]


async def get_episode_by_id(episode_id: int):
    rows = await _execute(
        """SELECT e.id, e.episode_number, e.video_file_id, e.movie_id, m.title
           FROM movie_episodes e JOIN movies m ON m.id = e.movie_id
           WHERE e.id = ?""",
        (episode_id,),
    )
    if not rows:
        return None
    r = rows[0]
    return {"id": r[0], "episode_number": r[1], "video_file_id": r[2], "movie_id": r[3], "title": r[4]}


# ---------- Umumiy ----------
async def get_movies(category: str | None = None, search: str | None = None,
                      sort: str = "new", limit: int | None = None):
    query = MOVIE_SELECT + " WHERE 1=1"
    params = []
    if category:
        query += " AND m.category = ?"
        params.append(category)
    if search:
        query += " AND m.title LIKE ?"
        params.append(f"%{search}%")

    query += " ORDER BY m.views DESC, m.id DESC" if sort == "top" else " ORDER BY m.id DESC"

    if limit:
        query += " LIMIT ?"
        params.append(limit)

    rows = await _execute(query, tuple(params))
    return [_row_to_movie(r) for r in rows]


async def get_movie_by_id(movie_id: int):
    rows = await _execute(MOVIE_SELECT + " WHERE m.id = ?", (movie_id,))
    if not rows:
        return None
    return _row_to_movie(rows[0])


async def get_movie_by_code(code: str):
    rows = await _execute(MOVIE_SELECT + " WHERE m.code = ?", (code,))
    if not rows:
        return None
    return _row_to_movie(rows[0])


async def get_movie_video(movie_id: int) -> str | None:
    rows = await _execute("SELECT video_file_id FROM movies WHERE id = ?", (movie_id,))
    return rows[0][0] if rows else None


async def increment_views(movie_id: int):
    await _execute("UPDATE movies SET views = views + 1 WHERE id = ?", (movie_id,))


async def delete_movie(code: str) -> bool:
    rows = await _execute("SELECT id FROM movies WHERE code = ?", (code,))
    if not rows:
        return False
    movie_id = rows[0][0]
    await _execute("DELETE FROM movies WHERE code = ?", (code,))
    await _execute("DELETE FROM movie_episodes WHERE movie_id = ?", (movie_id,))
    await _execute("DELETE FROM favorites WHERE movie_id = ?", (movie_id,))
    await _execute("DELETE FROM history WHERE movie_id = ?", (movie_id,))
    return True


# ---------- Sevimlilar ----------
async def toggle_favorite(user_id: int, movie_id: int) -> bool:
    rows = await _execute(
        "SELECT 1 FROM favorites WHERE user_id = ? AND movie_id = ?", (user_id, movie_id)
    )
    if rows:
        await _execute(
            "DELETE FROM favorites WHERE user_id = ? AND movie_id = ?", (user_id, movie_id)
        )
        return False
    await _execute(
        "INSERT INTO favorites (user_id, movie_id) VALUES (?, ?)", (user_id, movie_id)
    )
    return True


async def get_favorite_ids(user_id: int) -> list[int]:
    rows = await _execute("SELECT movie_id FROM favorites WHERE user_id = ?", (user_id,))
    return [r[0] for r in rows]


async def get_favorite_movies(user_id: int):
    rows = await _execute(
        MOVIE_SELECT + " JOIN favorites f ON m.id = f.movie_id WHERE f.user_id = ? ORDER BY f.created_at DESC",
        (user_id,),
    )
    return [_row_to_movie(r) for r in rows]


# ---------- Tarix ----------
async def add_history(user_id: int, movie_id: int):
    await _execute(
        """INSERT INTO history (user_id, movie_id, watched_at) VALUES (?, ?, CURRENT_TIMESTAMP)
           ON CONFLICT(user_id, movie_id) DO UPDATE SET watched_at = CURRENT_TIMESTAMP""",
        (user_id, movie_id),
    )


async def get_history_movies(user_id: int, limit: int = 30):
    rows = await _execute(
        MOVIE_SELECT + """ JOIN history h ON m.id = h.movie_id
            WHERE h.user_id = ? ORDER BY h.watched_at DESC LIMIT ?""",
        (user_id, limit),
    )
    return [_row_to_movie(r) for r in rows]


async def get_history_count(user_id: int) -> int:
    rows = await _execute("SELECT COUNT(*) FROM history WHERE user_id = ?", (user_id,))
    return rows[0][0] if rows else 0


async def close_db():
    pass


# ---------- Foydalanuvchilar / statistika ----------
async def track_user(user_id: int, first_name: str | None, username: str | None):
    await _execute(
        """INSERT INTO bot_users (user_id, first_name, username, joined_at, last_seen)
           VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
           ON CONFLICT(user_id) DO UPDATE SET
             first_name = excluded.first_name,
             username = excluded.username,
             last_seen = CURRENT_TIMESTAMP""",
        (user_id, first_name, username),
    )


async def get_user_count() -> int:
    rows = await _execute("SELECT COUNT(*) FROM bot_users")
    return rows[0][0] if rows else 0


async def get_recent_users(limit: int = 30):
    rows = await _execute(
        "SELECT user_id, first_name, username, joined_at FROM bot_users ORDER BY joined_at DESC LIMIT ?",
        (limit,),
    )
    return [{"user_id": r[0], "first_name": r[1], "username": r[2], "joined_at": r[3]} for r in rows]


async def get_stats() -> dict:
    total_users = await get_user_count()
    rows = await _execute("SELECT category, COUNT(*) FROM movies GROUP BY category")
    cat_counts = {r[0]: r[1] for r in rows}
    ep_rows = await _execute("SELECT COUNT(*) FROM movie_episodes")
    total_episodes = ep_rows[0][0] if ep_rows else 0
    view_rows = await _execute("SELECT COALESCE(SUM(views), 0) FROM movies")
    total_views = view_rows[0][0] if view_rows else 0
    return {
        "users": total_users,
        "kino": cat_counts.get("kino", 0),
        "multfilm": cat_counts.get("multfilm", 0),
        "serial": cat_counts.get("serial", 0),
        "episodes": total_episodes,
        "views": total_views,
    }
