const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();
try {
  tg.setHeaderColor("#0a0a10");
  tg.setBackgroundColor("#0a0a10");
} catch (e) {}

const CATEGORY_EMOJI = { kino: "🎬", multfilm: "🧸", serial: "📺" };
const CATEGORY_LABEL = { kino: "Kino", multfilm: "Multfilm", serial: "Serial" };

const els = {
  searchInput: document.getElementById("searchInput"),
  searchCancel: document.getElementById("searchCancel"),
  searchSuggestions: document.getElementById("searchSuggestions"),
  homeContent: document.getElementById("homeContent"),
  tabs: document.querySelectorAll(".tab"),
  grid: document.getElementById("grid"),
  emptyState: document.getElementById("emptyState"),
  newCarousel: document.getElementById("newCarousel"),
  topCarousel: document.getElementById("topCarousel"),
  heroTrack: document.getElementById("heroTrack"),
  heroDots: document.getElementById("heroDots"),
  heroPrev: document.getElementById("heroPrev"),
  heroNext: document.getElementById("heroNext"),
  avatarRow: document.getElementById("avatarRow"),
  ratingGrid: document.getElementById("ratingGrid"),
  favGrid: document.getElementById("favGrid"),
  favEmpty: document.getElementById("favEmpty"),
  historyGrid: document.getElementById("historyGrid"),
  historyEmpty: document.getElementById("historyEmpty"),
  profileAvatar: document.getElementById("profileAvatar"),
  profileName: document.getElementById("profileName"),
  statHistory: document.getElementById("statHistory"),
  statFav: document.getElementById("statFav"),
  contactAdminBtn: document.getElementById("contactAdminBtn"),
  navBtns: document.querySelectorAll(".nav-btn"),
  views: document.querySelectorAll(".view"),
  modalOverlay: document.getElementById("modalOverlay"),
  modalClose: document.getElementById("modalClose"),
  modalFav: document.getElementById("modalFav"),
  modalPoster: document.getElementById("modalPoster"),
  modalTitle: document.getElementById("modalTitle"),
  modalDesc: document.getElementById("modalDesc"),
  watchBtn: document.getElementById("watchBtn"),
  episodeList: document.getElementById("episodeList"),
  watchStatus: document.getElementById("watchStatus"),
};

let state = {
  currentCategory: "",
  currentSearch: "",
  currentMovie: null,
  favoriteIds: new Set(),
  currentView: "home",
  adminUsername: "",
};

let debounceTimer = null;

// ---------- helpers ----------
function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str || "";
  return div.innerHTML;
}

async function postJSON(url, data) {
  const res = await fetch(url, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(data),
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.detail || `HTTP ${res.status}`);
  }
  return res.json();
}

// ---------- card building ----------
function buildCard(movie, index = 0) {
  const card = document.createElement("div");
  card.className = "card";
  card.style.animationDelay = `${Math.min(index, 10) * 40}ms`;

  const isFav = state.favoriteIds.has(movie.id);
  const posterInner = movie.poster_file_id
    ? `<img class="poster-img" src="/api/poster/${movie.id}" loading="lazy" />`
    : `<div class="poster-fallback">${CATEGORY_EMOJI[movie.category] || "🎬"}</div>`;

  const badgeText = movie.is_series && movie.episode_count > 0
    ? `📺 ${movie.episode_count} qism`
    : `${CATEGORY_EMOJI[movie.category] || ""} ${CATEGORY_LABEL[movie.category] || ""}`;

  card.innerHTML = `
    ${posterInner}
    <div class="card-badge">${badgeText}</div>
    <button class="card-heart" data-id="${movie.id}">${isFav ? "❤️" : "🤍"}</button>
    <div class="card-scrim">
      <div class="card-title">${escapeHtml(movie.title)}</div>
      <div class="card-meta">👁 ${movie.views}</div>
    </div>
  `;

  card.addEventListener("click", (e) => {
    if (e.target.closest(".card-heart")) return;
    tg.HapticFeedback?.impactOccurred("light");
    openModal(movie);
  });

  card.querySelector(".card-heart").addEventListener("click", async (e) => {
    e.stopPropagation();
    await handleToggleFavorite(movie.id, card.querySelector(".card-heart"));
  });

  return card;
}

function skeletonCards(container, count, small = false) {
  container.innerHTML = "";
  for (let i = 0; i < count; i++) {
    const s = document.createElement("div");
    s.className = small ? "skeleton-card-sm" : "skeleton-card";
    container.appendChild(s);
  }
}

// ---------- favorite toggle ----------
async function handleToggleFavorite(movieId, btnEl) {
  try {
    const result = await postJSON(`/api/favorite/${movieId}`, { initData: tg.initData });
    if (result.favorited) {
      state.favoriteIds.add(movieId);
      tg.HapticFeedback?.notificationOccurred("success");
    } else {
      state.favoriteIds.delete(movieId);
    }
    if (btnEl) btnEl.textContent = result.favorited ? "❤️" : "🤍";
    if (state.currentMovie && state.currentMovie.id === movieId) {
      els.modalFav.textContent = result.favorited ? "❤️" : "🤍";
    }
    if (state.currentView === "favorites") loadFavorites();
    if (state.currentView === "profile") loadProfile();
  } catch (e) {
    tg.HapticFeedback?.notificationOccurred("error");
  }
}

// ---------- Hero carousel ----------
let heroTimer = null;
let heroIndex = 0;
let heroMovies = [];

function renderHero(movies) {
  heroMovies = movies.slice(0, 6);
  els.heroTrack.innerHTML = "";
  els.heroDots.innerHTML = "";

  heroMovies.forEach((movie, i) => {
    const slide = document.createElement("div");
    slide.className = "hero-slide";
    if (movie.poster_file_id) {
      slide.style.backgroundImage = `url(/api/poster/${movie.id})`;
    }
    const badge = movie.is_series ? "SERIAL" : (CATEGORY_LABEL[movie.category] || "KINO").toUpperCase();
    slide.innerHTML = `
      <div class="hero-play">▶</div>
      <div class="hero-slide-info">
        <span class="hero-badge">${badge}</span>
        <div class="hero-slide-title">${escapeHtml(movie.title)}</div>
      </div>
    `;
    slide.addEventListener("click", () => openModal(movie));
    els.heroTrack.appendChild(slide);

    const dot = document.createElement("div");
    dot.className = "hero-dot" + (i === 0 ? " active" : "");
    els.heroDots.appendChild(dot);
  });

  heroIndex = 0;
  restartHeroTimer();
}

function goToHeroSlide(index) {
  if (heroMovies.length === 0) return;
  heroIndex = (index + heroMovies.length) % heroMovies.length;
  els.heroTrack.scrollTo({ left: heroIndex * els.heroTrack.clientWidth, behavior: "smooth" });
  [...els.heroDots.children].forEach((d, i) => d.classList.toggle("active", i === heroIndex));
}

function restartHeroTimer() {
  clearInterval(heroTimer);
  if (heroMovies.length <= 1) return;
  heroTimer = setInterval(() => goToHeroSlide(heroIndex + 1), 4500);
}

els.heroPrev.addEventListener("click", () => { goToHeroSlide(heroIndex - 1); restartHeroTimer(); });
els.heroNext.addEventListener("click", () => { goToHeroSlide(heroIndex + 1); restartHeroTimer(); });

els.heroTrack.addEventListener("scroll", () => {
  clearTimeout(els.heroTrack._scrollTimer);
  els.heroTrack._scrollTimer = setTimeout(() => {
    const idx = Math.round(els.heroTrack.scrollLeft / els.heroTrack.clientWidth);
    heroIndex = idx;
    [...els.heroDots.children].forEach((d, i) => d.classList.toggle("active", i === idx));
  }, 100);
});

// ---------- Avatar row ----------
function renderAvatarRow(movies) {
  els.avatarRow.innerHTML = "";
  movies.slice(0, 12).forEach((movie) => {
    const item = document.createElement("div");
    item.className = "avatar-item";
    const inner = movie.poster_file_id
      ? `<img src="/api/poster/${movie.id}" loading="lazy" />`
      : (CATEGORY_EMOJI[movie.category] || "🎬");
    item.innerHTML = `
      <div class="avatar-ring"><div class="avatar-inner">${inner}</div></div>
      <div class="avatar-label">${escapeHtml(movie.title)}</div>
    `;
    item.addEventListener("click", () => openModal(movie));
    els.avatarRow.appendChild(item);
  });
}

// ---------- Bosh sahifa ----------
async function loadHomeCarousels() {
  skeletonCards(els.newCarousel, 5, true);
  skeletonCards(els.topCarousel, 5, true);

  const [newMovies, topMovies] = await Promise.all([
    fetch("/api/movies?sort=new&limit=10").then((r) => r.json()),
    fetch("/api/movies?sort=top&limit=10").then((r) => r.json()),
  ]);

  els.newCarousel.innerHTML = "";
  newMovies.forEach((m, i) => els.newCarousel.appendChild(buildCard(m, i)));

  els.topCarousel.innerHTML = "";
  topMovies.forEach((m, i) => els.topCarousel.appendChild(buildCard(m, i)));

  renderHero(topMovies.length ? topMovies : newMovies);
  renderAvatarRow(newMovies);
}

async function loadHomeGrid() {
  skeletonCards(els.grid, 6);
  els.emptyState.classList.add("hidden");

  const params = new URLSearchParams();
  if (state.currentCategory) params.set("category", state.currentCategory);
  if (state.currentSearch) params.set("search", state.currentSearch);

  const res = await fetch(`/api/movies?${params.toString()}`);
  const movies = await res.json();

  els.grid.innerHTML = "";
  els.emptyState.classList.toggle("hidden", movies.length !== 0);
  movies.forEach((m, i) => els.grid.appendChild(buildCard(m, i)));
}

// ---------- Reyting ----------
async function loadRating() {
  skeletonCards(els.ratingGrid, 8);
  const movies = await fetch("/api/movies?sort=top&limit=30").then((r) => r.json());
  els.ratingGrid.innerHTML = "";
  movies.forEach((m, i) => els.ratingGrid.appendChild(buildCard(m, i)));
}

// ---------- Sevimlilar ----------
async function loadFavorites() {
  skeletonCards(els.favGrid, 4);
  els.favEmpty.classList.add("hidden");
  try {
    const data = await postJSON("/api/favorites", { initData: tg.initData });
    state.favoriteIds = new Set(data.ids);
    els.favGrid.innerHTML = "";
    els.favEmpty.classList.toggle("hidden", data.movies.length !== 0);
    data.movies.forEach((m, i) => els.favGrid.appendChild(buildCard(m, i)));
  } catch (e) {
    els.favGrid.innerHTML = "";
    els.favEmpty.classList.remove("hidden");
  }
}

// ---------- Profil ----------
async function loadProfile() {
  const u = tg.initDataUnsafe?.user;
  if (u) {
    const name = [u.first_name, u.last_name].filter(Boolean).join(" ") || u.username || "Foydalanuvchi";
    els.profileName.textContent = name;
    els.profileAvatar.textContent = (u.first_name || "🙂").slice(0, 1).toUpperCase();
  }

  skeletonCards(els.historyGrid, 4);
  els.historyEmpty.classList.add("hidden");

  try {
    const [historyData, favData] = await Promise.all([
      postJSON("/api/history", { initData: tg.initData }),
      postJSON("/api/favorites", { initData: tg.initData }),
    ]);
    els.statHistory.textContent = historyData.count;
    els.statFav.textContent = favData.movies.length;
    state.favoriteIds = new Set(favData.ids);

    els.historyGrid.innerHTML = "";
    els.historyEmpty.classList.toggle("hidden", historyData.movies.length !== 0);
    historyData.movies.forEach((m, i) => els.historyGrid.appendChild(buildCard(m, i)));
  } catch (e) {
    els.historyGrid.innerHTML = "";
    els.historyEmpty.classList.remove("hidden");
  }
}

const EPISODE_CHUNK = 20;

function renderEpisodeStage(episodes, rangeIndex) {
  els.episodeList.innerHTML = "";

  let list = episodes;
  const showRanges = episodes.length > EPISODE_CHUNK && rangeIndex === null;

  if (showRanges) {
    const rangeCount = Math.ceil(episodes.length / EPISODE_CHUNK);
    for (let i = 0; i < rangeCount; i++) {
      const start = episodes[i * EPISODE_CHUNK].episode_number;
      const endIdx = Math.min((i + 1) * EPISODE_CHUNK, episodes.length) - 1;
      const end = episodes[endIdx].episode_number;
      const btn = document.createElement("button");
      btn.className = "episode-btn episode-range";
      btn.textContent = `${start}-${end}`;
      btn.addEventListener("click", () => renderEpisodeStage(episodes, i));
      els.episodeList.appendChild(btn);
    }
    return;
  }

  if (episodes.length > EPISODE_CHUNK && rangeIndex !== null) {
    list = episodes.slice(rangeIndex * EPISODE_CHUNK, (rangeIndex + 1) * EPISODE_CHUNK);
    const backBtn = document.createElement("button");
    backBtn.className = "episode-btn episode-back";
    backBtn.textContent = "⬅️ Orqaga";
    backBtn.addEventListener("click", () => renderEpisodeStage(episodes, null));
    els.episodeList.appendChild(backBtn);
  }

  list.forEach((ep) => {
    const btn = document.createElement("button");
    btn.className = "episode-btn";
    btn.textContent = `${ep.episode_number}-qism`;
    btn.addEventListener("click", () => sendEpisode(ep.id, btn));
    els.episodeList.appendChild(btn);
  });
}

// ---------- Modal ----------
async function openModal(movie) {
  state.currentMovie = movie;
  els.modalTitle.textContent = movie.title;
  els.modalDesc.textContent = movie.description || "";
  els.modalPoster.innerHTML = movie.poster_file_id
    ? `<img src="/api/poster/${movie.id}" />`
    : (CATEGORY_EMOJI[movie.category] || "🎬");
  els.modalFav.textContent = state.favoriteIds.has(movie.id) ? "❤️" : "🤍";
  els.watchStatus.textContent = "";
  els.modalOverlay.classList.add("open");

  if (movie.is_series) {
    els.watchBtn.classList.add("hidden");
    els.episodeList.classList.remove("hidden");
    els.episodeList.innerHTML = `<p class="watch-status">Yuklanmoqda...</p>`;
    try {
      const res = await fetch(`/api/episodes/${movie.id}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const episodes = await res.json();

      if (episodes.length === 0) {
        els.episodeList.innerHTML = `<p class="watch-status">Hali qismlar qo'shilmagan</p>`;
        return;
      }
      renderEpisodeStage(episodes, null);
    } catch (e) {
      console.error("Qismlarni yuklashda xato:", e);
      els.episodeList.innerHTML = `<p class="watch-status">❌ Qismlarni yuklab bo'lmadi: ${e.message}<br><button id="retryEpisodes" class="episode-btn" style="margin-top:8px;width:100%">🔄 Qayta urinish</button></p>`;
      const retryBtn = document.getElementById("retryEpisodes");
      if (retryBtn) retryBtn.addEventListener("click", () => openModal(movie));
    }
  } else {
    els.watchBtn.classList.remove("hidden");
    els.episodeList.classList.add("hidden");
    els.watchBtn.disabled = false;
    els.watchBtn.textContent = "▶️ Tomosha qilish";
  }
}

async function sendEpisode(episodeId, btnEl) {
  const original = btnEl.textContent;
  btnEl.disabled = true;
  btnEl.textContent = "...";
  try {
    await postJSON(`/api/watch-episode/${episodeId}`, { initData: tg.initData });
    btnEl.textContent = "✅";
    btnEl.classList.add("sent");
    els.watchStatus.textContent = "✅ Video Telegram chatingizga yuborildi!";
    tg.HapticFeedback?.notificationOccurred("success");
  } catch (e) {
    btnEl.textContent = original;
    els.watchStatus.textContent = `❌ Xatolik: ${e.message}`;
  } finally {
    btnEl.disabled = false;
  }
}

els.modalClose.addEventListener("click", () => els.modalOverlay.classList.remove("open"));
els.modalOverlay.addEventListener("click", (e) => {
  if (e.target === els.modalOverlay) els.modalOverlay.classList.remove("open");
});

els.modalFav.addEventListener("click", async () => {
  if (!state.currentMovie) return;
  await handleToggleFavorite(state.currentMovie.id, null);
  els.modalFav.textContent = state.favoriteIds.has(state.currentMovie.id) ? "❤️" : "🤍";
});

els.watchBtn.addEventListener("click", async () => {
  if (!state.currentMovie) return;
  els.watchBtn.disabled = true;
  els.watchBtn.textContent = "Yuborilmoqda...";

  try {
    await postJSON(`/api/watch/${state.currentMovie.id}`, { initData: tg.initData });
    els.watchStatus.textContent = "✅ Video Telegram chatingizga yuborildi!";
    els.watchBtn.textContent = "✅ Yuborildi";
    tg.HapticFeedback?.notificationOccurred("success");
  } catch (e) {
    els.watchStatus.textContent = `❌ Xatolik: ${e.message}`;
    els.watchBtn.disabled = false;
    els.watchBtn.textContent = "▶️ Qayta urinish";
  }
});

// ---------- Tabs & Search (Bosh sahifa) ----------
els.tabs.forEach((tab) => {
  tab.addEventListener("click", () => {
    els.tabs.forEach((t) => t.classList.remove("active"));
    tab.classList.add("active");
    state.currentCategory = tab.dataset.cat;
    loadHomeGrid();
  });
});

// ---------- Qidiruv rejimi ----------
function enterSearchMode() {
  els.homeContent.classList.add("hidden");
  els.searchSuggestions.classList.remove("hidden");
  els.searchCancel.classList.remove("hidden");
  clearInterval(heroTimer);
}

function exitSearchMode() {
  els.searchInput.value = "";
  els.searchInput.blur();
  els.searchSuggestions.classList.add("hidden");
  els.searchSuggestions.innerHTML = "";
  els.searchCancel.classList.add("hidden");
  els.homeContent.classList.remove("hidden");
  restartHeroTimer();
}

function renderSuggestions(movies, query) {
  els.searchSuggestions.innerHTML = "";
  if (!query) {
    els.searchSuggestions.innerHTML = `<p class="watch-status">Kino yoki serial nomini yozing...</p>`;
    return;
  }
  if (movies.length === 0) {
    els.searchSuggestions.innerHTML = `<p class="watch-status">🔍 "${escapeHtml(query)}" bo'yicha hech narsa topilmadi</p>`;
    return;
  }
  movies.forEach((movie) => {
    const item = document.createElement("div");
    item.className = "search-suggestion-item";
    const poster = movie.poster_file_id
      ? `<img src="/api/poster/${movie.id}" />`
      : (CATEGORY_EMOJI[movie.category] || "🎬");
    const meta = movie.is_series && movie.episode_count > 0
      ? `📺 ${movie.episode_count} qism`
      : (CATEGORY_LABEL[movie.category] || "");
    item.innerHTML = `
      <div class="suggestion-poster">${poster}</div>
      <div class="suggestion-info">
        <div class="suggestion-title">${escapeHtml(movie.title)}</div>
        <div class="suggestion-meta">${meta} · 👁 ${movie.views}</div>
      </div>
    `;
    item.addEventListener("click", () => {
      exitSearchMode();
      openModal(movie);
    });
    els.searchSuggestions.appendChild(item);
  });
}

els.searchInput.addEventListener("focus", enterSearchMode);

els.searchInput.addEventListener("input", () => {
  clearTimeout(debounceTimer);
  const query = els.searchInput.value.trim();
  debounceTimer = setTimeout(async () => {
    if (!query) {
      renderSuggestions([], "");
      return;
    }
    els.searchSuggestions.innerHTML = `<p class="watch-status">Qidirilmoqda...</p>`;
    try {
      const res = await fetch(`/api/movies?search=${encodeURIComponent(query)}`);
      const movies = await res.json();
      renderSuggestions(movies, query);
    } catch (e) {
      els.searchSuggestions.innerHTML = `<p class="watch-status">❌ Xatolik yuz berdi</p>`;
    }
  }, 300);
});

els.searchCancel.addEventListener("click", exitSearchMode);

// ---------- Bottom nav ----------
function switchView(viewName) {
  state.currentView = viewName;
  els.navBtns.forEach((b) => b.classList.toggle("active", b.dataset.view === viewName));
  els.views.forEach((v) => v.classList.toggle("hidden", v.id !== `view-${viewName}`));

  if (viewName === "rating") loadRating();
  if (viewName === "favorites") loadFavorites();
  if (viewName === "profile") loadProfile();
}

els.navBtns.forEach((btn) => {
  btn.addEventListener("click", () => {
    tg.HapticFeedback?.impactOccurred("light");
    switchView(btn.dataset.view);
  });
});

// ---------- Admin bilan bog'lanish ----------
async function loadMeta() {
  try {
    const meta = await fetch("/api/meta").then((r) => r.json());
    state.adminUsername = meta.admin_username || "";
  } catch (e) {}
}

els.contactAdminBtn.addEventListener("click", () => {
  if (state.adminUsername) {
    tg.openTelegramLink(`https://t.me/${state.adminUsername}`);
  } else {
    els.contactAdminBtn.textContent = "Admin username sozlanmagan";
  }
});

// ---------- Init ----------
(async function init() {
  await loadMeta();
  try {
    const favData = await postJSON("/api/favorites", { initData: tg.initData });
    state.favoriteIds = new Set(favData.ids);
  } catch (e) {}
  loadHomeCarousels();
  loadHomeGrid();
})();
