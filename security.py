import hashlib
import hmac
import json
import logging
from urllib.parse import parse_qsl

from config import BOT_TOKEN

logger = logging.getLogger(__name__)


def validate_init_data(init_data: str, max_age_seconds: int = 86400) -> dict | None:
    """
    Telegram Mini App'dan kelgan initData'ni tekshiradi.
    Muvaffaqiyatli bo'lsa foydalanuvchi ma'lumotini qaytaradi, aks holda None.
    Bu tekshiruv orqali faqat haqiqiy Telegram foydalanuvchilari videoni so'rashi mumkin.
    """
    if not init_data:
        logger.warning("VALIDATE FAIL: init_data bo'sh keldi")
        return None

    try:
        parsed = dict(parse_qsl(init_data, strict_parsing=True))
    except ValueError as e:
        logger.warning(f"VALIDATE FAIL: parse_qsl xatosi: {e} | raw={init_data[:200]}")
        return None

    received_hash = parsed.pop("hash", None)
    if not received_hash:
        logger.warning(f"VALIDATE FAIL: hash maydoni topilmadi | keys={list(parsed.keys())}")
        return None

    data_check_string = "\n".join(f"{k}={v}" for k, v in sorted(parsed.items()))

    if not BOT_TOKEN:
        logger.warning("VALIDATE FAIL: BOT_TOKEN bo'sh (config.py / environment tekshiring)")
        return None

    secret_key = hmac.new(b"WebAppData", BOT_TOKEN.encode(), hashlib.sha256).digest()
    calculated_hash = hmac.new(secret_key, data_check_string.encode(), hashlib.sha256).hexdigest()

    if not hmac.compare_digest(calculated_hash, received_hash):
        logger.warning(
            f"VALIDATE FAIL: hash mos kelmadi. calculated={calculated_hash} received={received_hash}"
        )
        return None

    user_raw = parsed.get("user")
    if not user_raw:
        logger.warning("VALIDATE FAIL: user maydoni topilmadi")
        return None

    try:
        user = json.loads(user_raw)
    except json.JSONDecodeError as e:
        logger.warning(f"VALIDATE FAIL: user JSON xato: {e}")
        return None

    logger.info(f"VALIDATE OK: user_id={user.get('id')}")
    return user
