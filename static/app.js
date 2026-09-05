const tg = window.Telegram.WebApp;
tg.ready();
tg.expand();
try {
  tg.setHeaderColor("#06070a");
  tg.setBackgroundColor("#06070a");
} catch (e) {}

const CATEGORY_SVG = {
  kino: `<svg class="badge-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="2" width="20" height="20" rx="2.18"/><line x1="7" y1="2" x2="7" y2="22"/><line x1="17" y1="2" x2="17" y2="22"/><line x1="2" y1="12" x2="22" y2="12"/><line x1="2" y1="7" x2="7" y2="7"/><line x1="2" y1="17" x2="7" y2="17"/><line x1="17" y1="17" x2="22" y2="17"/><line x1="17" y1="7" x2="22" y2="7"/></svg>`,
  serial: `<svg class="badge-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><rect x="2" y="7" width="20" height="15" rx="2"/><polyline points="17 2 12 7 7 2"/></svg>`,
  multfilm: `<svg class="badge-icon-svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 2l2.4 7.4H22l-6 4.6 2.3 7-6.3-4.6-6.3 4.6 2.3-7-6-4.6h7.6z"/></svg>`,
};
const CATEGORY_LABEL = { kino: "Kino", multfilm: "Multfilm", serial: "Serial" };

const els = {
  headerSearchBtn: document.getElementById("headerSearchBtn"),
  homeSearchTrigger: document.getElementById("homeSearchTrigger"),
  bottomNavSearch: document.getElementById("bottomNavSearch"),
  searchModal: document.getElementById("searchModal"),
  searchModalBackBtn: document.getElementById("searchModalBackBtn"),
  searchModalInput: document.getElementById("searchModalInput"),
  searchModalClearBtn: document.getElementById("searchModalClearBtn"),
  searchDefaultState: document.getElementById("searchDefaultState"),
  searchSuggestions: document.getElementById("searchSuggestions"),
  searchTypeChips: document.querySelectorAll(".search-type-chip"),
  quickChips: document.querySelectorAll(".quick-chip"),
  heroBillboard: document.getElementById("heroBillboard"),
  heroBg: document.getElementById("heroBg"),
  heroTitle: document.getElementById("heroTitle"),
  heroCat: document.getElementById("heroCat"),
  heroCodeBadge: document.getElementById("heroCodeBadge"),
  heroDesc: document.getElementById("heroDesc"),
  heroWatchBtn: document.getElementById("heroWatchBtn"),
  heroDetailBtn: document.getElementById("heroDetailBtn"),
  modalBackdropBlur: document.getElementById("modalBackdropBlur"),
  sectionNew: document.getElementById("sectionNew"),
  gridTitle: document.getElementById("gridTitle"),
  tabs: document.querySelectorAll(".tab-chip"),
  grid: document.getElementById("grid"),
  emptyState: document.getElementById("emptyState"),
  newCarousel: document.getElementById("newCarousel"),
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
  navBtns: document.querySelectorAll(".nav-item"),
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
  searchType: "all",
  currentMovie: null,
  favoriteIds: new Set(),
  currentView: "home",
  adminUsername: "",
  hasNewMovies: false,
};

let debounceTimer = null;

// ---------- Helpers ----------
function escapeHtml(str) {
  const div = document.createElement("div");
  div.textContent = str || "";
  return div.innerHTML;
}

function formatViews(num) {
  if (!num) return "0";
  if (num >= 1000000) return (num / 1000000).toFixed(1) + "M";
  if (num >= 1000) return (num / 1000).toFixed(1) + "k";
  return num;
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

// ---------- Movie Card Builder ----------
function buildCard(movie, index = 0, rank = null) {
  const card = document.createElement("div");
  card.className = "movie-card";
  card.style.animationDelay = `${Math.min(index, 10) * 45}ms`;

  const isFav = state.favoriteIds.has(movie.id);
  const catSvg = CATEGORY_SVG[movie.category] || CATEGORY_SVG.kino;
  const posterInner = movie.poster_file_id
    ? `<img class="poster-image" src="/api/poster/${movie.id}" loading="lazy" alt="${escapeHtml(movie.title)}" />`
    : `<div class="poster-fallback-vector">${catSvg}</div>`;

  const hasEps = movie.has_episodes || movie.is_series || (movie.episode_count > 1);
  const badgeLabel = hasEps && movie.episode_count > 0
    ? `${movie.episode_count} qism`
    : (CATEGORY_LABEL[movie.category] || "Kino");
  const badgeText = `${catSvg} <span>${badgeLabel}</span>`;

  const rankHtml = rank ? `<div class="cinema-rank-number">${rank}</div>` : "";

  card.innerHTML = `
    ${rankHtml}
    ${posterInner}
    <div class="movie-card-badge">${badgeText}</div>
    <button class="movie-card-fav-btn" data-id="${movie.id}">${isFav ? "❤️" : "🤍"}</button>
    <div class="movie-card-scrim">
      <div class="movie-card-title">${escapeHtml(movie.title)}</div>
      <div class="movie-card-stats">
        <span>👁 ${formatViews(movie.views)}</span>
        <span style="color:var(--gold);font-weight:700">🔑 ${movie.code}</span>
      </div>
    </div>
  `;

  card.addEventListener("click", (e) => {
    if (e.target.closest(".movie-card-fav-btn")) return;
    tg.HapticFeedback?.impactOccurred("light");
    openModal(movie);
  });

  card.querySelector(".movie-card-fav-btn").addEventListener("click", async (e) => {
    e.stopPropagation();
    await handleToggleFavorite(movie.id, card.querySelector(".movie-card-fav-btn"));
  });

  return card;
}

function skeletonCards(container, count) {
  container.innerHTML = "";
  for (let i = 0; i < count; i++) {
    const s = document.createElement("div");
    s.className = "skeleton-box";
    container.appendChild(s);
  }
}

// ---------- Favorite Toggle ----------
async function handleToggleFavorite(movieId, btnEl) {
  try {
    const result = await postJSON(`/api/favorite/${movieId}`, { initData: tg.initData });
    if (result.favorited) {
      state.favoriteIds.add(movieId);
      tg.HapticFeedback?.notificationOccurred("success");
    } else {
      state.favoriteIds.delete(movieId);
    }
    const favButtons = document.querySelectorAll(`.movie-card-fav-btn[data-id="${movieId}"]`);
    favButtons.forEach(btn => {
      btn.textContent = result.favorited ? "❤️" : "🤍";
    });
    if (state.currentMovie && state.currentMovie.id === movieId) {
      els.modalFav.textContent = result.favorited ? "❤️" : "🤍";
    }
    if (state.currentView === "favorites") loadFavorites();
    if (state.currentView === "profile") loadProfile();
  } catch (e) {
    tg.HapticFeedback?.notificationOccurred("error");
  }
}

// ---------- Home Views Loader ----------
async function loadHomeCarousels() {
  if (els.newCarousel) skeletonCards(els.newCarousel, 5);

  try {
    // 7 kun ichida qo'shilgan kinolarni olamiz
    const res = await fetch("/api/movies?sort=new&limit=10&days_limit=7");
    const newMovies = await res.json();

    state.hasNewMovies = newMovies.length > 0;
    if (els.newCarousel) {
      els.newCarousel.innerHTML = "";
      if (state.hasNewMovies) {
        if (els.sectionNew) els.sectionNew.classList.toggle("hidden", !(!state.currentCategory));
        newMovies.forEach((m, i) => els.newCarousel.appendChild(buildCard(m, i)));
      } else {
        if (els.sectionNew) els.sectionNew.classList.add("hidden");
      }
    }
  } catch (e) {
    console.error("Yangi kinolarni yuklashda xato:", e);
    state.hasNewMovies = false;
    if (els.sectionNew) els.sectionNew.classList.add("hidden");
  }
}

function setupHeroBillboard(movies) {
  if (!els.heroBillboard) return;
  if (state.currentCategory || !movies || movies.length === 0) {
    els.heroBillboard.classList.add("hidden");
    return;
  }

  // Eng sara kino (posteri bor birinchisi yoki eng ko'p ko'rilgani)
  const featured = movies.find((m) => m.poster_file_id) || movies[0];
  if (!featured) {
    els.heroBillboard.classList.add("hidden");
    return;
  }

  if (featured.poster_file_id) {
    els.heroBg.style.backgroundImage = `url('/api/poster/${featured.id}')`;
  } else {
    els.heroBg.style.backgroundImage = "none";
  }

  els.heroTitle.textContent = featured.title;
  els.heroCat.textContent = CATEGORY_LABEL[featured.category] || "🎬 Kino";
  els.heroCodeBadge.textContent = `🔑 Kod: ${featured.code}`;
  els.heroDesc.textContent = featured.description || "Yuqori sifatli va unutilmas kino kartinasi. Hoziroq tomosha qiling!";

  els.heroWatchBtn.onclick = (e) => {
    e.stopPropagation();
    openModal(featured);
  };
  els.heroDetailBtn.onclick = (e) => {
    e.stopPropagation();
    openModal(featured);
  };
  els.heroBillboard.onclick = () => openModal(featured);

  els.heroBillboard.classList.remove("hidden");
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

  setupHeroBillboard(movies);

  movies.forEach((m, i) => els.grid.appendChild(buildCard(m, i)));
}

// ---------- Rating Page ----------
async function loadRating() {
  skeletonCards(els.ratingGrid, 8);
  const movies = await fetch("/api/movies?sort=top&limit=30").then((r) => r.json());
  els.ratingGrid.innerHTML = "";
  movies.forEach((m, i) => {
    const rank = i + 1;
    els.ratingGrid.appendChild(buildCard(m, i, rank));
  });
}

// ---------- Favorites Page ----------
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

// ---------- Profile & History Page ----------
async function loadProfile() {
  const u = tg.initDataUnsafe?.user;
  if (u) {
    const name = [u.first_name, u.last_name].filter(Boolean).join(" ") || u.username || "Foydalanuvchi";
    els.profileName.textContent = name;
    els.profileAvatar.textContent = (u.first_name || "👤").slice(0, 1).toUpperCase();
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
      btn.textContent = `${start}-${end} qismlar`;
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

// ---------- Ultra Glass Detail Modal ----------
async function openModal(movie) {
  state.currentMovie = movie;
  els.modalTitle.textContent = movie.title;
  els.modalDesc.textContent = movie.description || "";
  const metaRow = document.getElementById("modalMetaRow");
  if (metaRow) {
    metaRow.innerHTML = `
      <span class="meta-tag rating-tag">⭐ 9.${(movie.id % 9) + 1}</span>
      <span class="meta-tag quality-tag">4K Ultra HD</span>
      <span class="meta-tag code-tag" style="background:var(--gold-soft);color:var(--gold);border:1px solid var(--glass-border-gold);font-weight:700">🔑 Kod: ${movie.code}</span>
    `;
  }
  const catSvg = CATEGORY_SVG[movie.category] || CATEGORY_SVG.kino;
  els.modalPoster.innerHTML = movie.poster_file_id
    ? `<img src="/api/poster/${movie.id}" alt="" />`
    : `<div class="poster-fallback-vector">${catSvg}</div>`;
  if (els.modalBackdropBlur) {
    els.modalBackdropBlur.style.backgroundImage = movie.poster_file_id
      ? `url('/api/poster/${movie.id}')`
      : "none";
  }
  els.modalFav.textContent = state.favoriteIds.has(movie.id) ? "❤️" : "🤍";
  els.watchStatus.textContent = "";
  els.modalOverlay.classList.add("open");

  const hasEpisodes = movie.has_episodes || movie.is_series || (movie.episode_count > 0);

  if (hasEpisodes) {
    els.watchBtn.classList.add("hidden");
    els.episodeList.classList.remove("hidden");
    els.episodeList.innerHTML = `<p class="watch-status-msg">Qismlar yuklanmoqda...</p>`;
    try {
      const res = await fetch(`/api/episodes/${movie.id}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const episodes = await res.json();

      if (episodes.length === 0) {
        els.episodeList.innerHTML = `<p class="watch-status-msg">Hali qismlar qo'shilmagan</p>`;
        return;
      }
      renderEpisodeStage(episodes, null);
    } catch (e) {
      console.error("Qismlarni yuklashda xato:", e);
      els.episodeList.innerHTML = `<p class="watch-status-msg">❌ Qismlarni yuklab bo'lmadi: ${e.message}<br><button id="retryEpisodes" class="episode-btn" style="margin-top:8px;width:100%">🔄 Qayta urinish</button></p>`;
      const retryBtn = document.getElementById("retryEpisodes");
      if (retryBtn) retryBtn.addEventListener("click", () => openModal(movie));
    }
  } else {
    els.watchBtn.classList.remove("hidden");
    els.episodeList.classList.add("hidden");
    els.watchBtn.disabled = false;
    els.watchBtn.innerHTML = `<span class="play-symbol">▶</span><span class="btn-label">Hoziroq Tomosha Qilish</span>`;
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
  els.watchBtn.innerHTML = `<span class="btn-label">Yuborilmoqda...</span>`;

  try {
    await postJSON(`/api/watch/${state.currentMovie.id}`, { initData: tg.initData });
    els.watchStatus.textContent = "✅ Video Telegram chatingizga yuborildi!";
    els.watchBtn.innerHTML = `<span class="btn-label">✅ Yuborildi</span>`;
    tg.HapticFeedback?.notificationOccurred("success");
  } catch (e) {
    els.watchStatus.textContent = `❌ Xatolik: ${e.message}`;
    els.watchBtn.disabled = false;
    els.watchBtn.innerHTML = `<span class="play-symbol">▶</span><span class="btn-label">Qayta Urinish</span>`;
  }
});

// ---------- Tabs & Filter ----------
const CATEGORY_TITLE = {
  "": "Katalog",
  kino: "Kinolar",
  multfilm: "Multfilmlar",
  serial: "Seriallar",
};

function updateCategorySections() {
  const isAll = !state.currentCategory;
  if (els.sectionNew) {
    els.sectionNew.classList.toggle("hidden", !(isAll && state.hasNewMovies));
  }
  if (els.gridTitle) els.gridTitle.textContent = CATEGORY_TITLE[state.currentCategory] || "Katalog";
}

els.tabs.forEach((tab) => {
  tab.addEventListener("click", () => {
    els.tabs.forEach((t) => t.classList.remove("active"));
    tab.classList.add("active");
    state.currentCategory = tab.dataset.cat;
    updateCategorySections();
    loadHomeGrid();
  });
});

// ---------- Alohida Qidiruv Oynasi (Dedicated Search Modal) ----------
function openSearchModal(initialQuery = "") {
  if (!els.searchModal) return;
  els.searchModal.classList.remove("hidden");
  document.body.style.overflow = "hidden";

  try {
    if (tg && tg.BackButton) {
      tg.BackButton.show();
      tg.BackButton.onClick(closeSearchModal);
    }
  } catch (e) {}

  if (initialQuery) {
    els.searchModalInput.value = initialQuery;
    if (els.searchModalClearBtn) els.searchModalClearBtn.classList.remove("hidden");
    triggerSearch();
  } else {
    if (!els.searchModalInput.value.trim()) {
      if (els.searchDefaultState) els.searchDefaultState.classList.remove("hidden");
      if (els.searchSuggestions) {
        els.searchSuggestions.classList.add("hidden");
        els.searchSuggestions.innerHTML = "";
      }
    }
  }

  setTimeout(() => {
    if (els.searchModalInput) els.searchModalInput.focus();
  }, 80);
}

function closeSearchModal() {
  if (!els.searchModal) return;
  els.searchModal.classList.add("hidden");
  document.body.style.overflow = "";
  if (els.searchModalInput) els.searchModalInput.blur();

  try {
    if (tg && tg.BackButton) {
      tg.BackButton.hide();
      tg.BackButton.offClick(closeSearchModal);
    }
  } catch (e) {}

  // Pastki navigatsiyada qidiruv faol qolib ketmasligi uchun joriy ko'rinishga moslash
  if (els.navBtns) {
    els.navBtns.forEach((b) => b.classList.toggle("active", b.dataset.view === state.currentView));
  }
}

async function triggerSearch() {
  if (!els.searchModalInput) return;
  const query = els.searchModalInput.value.trim();

  if (els.searchModalClearBtn) {
    els.searchModalClearBtn.classList.toggle("hidden", query === "");
  }

  if (!query) {
    if (els.searchDefaultState) els.searchDefaultState.classList.remove("hidden");
    if (els.searchSuggestions) {
      els.searchSuggestions.classList.add("hidden");
      els.searchSuggestions.innerHTML = "";
    }
    return;
  }

  if (els.searchDefaultState) els.searchDefaultState.classList.add("hidden");
  if (els.searchSuggestions) {
    els.searchSuggestions.classList.remove("hidden");
    els.searchSuggestions.innerHTML = `<p class="watch-status-msg" style="padding:24px 0">🔍 Qidirilmoqda...</p>`;
  }

  try {
    const params = new URLSearchParams({ search: query });
    if (state.searchType && state.searchType !== "all") {
      params.set("search_type", state.searchType);
    }
    const res = await fetch(`/api/movies?${params.toString()}`);
    const movies = await res.json();
    renderSuggestions(movies, query);
  } catch (e) {
    if (els.searchSuggestions) {
      els.searchSuggestions.innerHTML = `<p class="watch-status-msg" style="padding:24px 0">❌ Qidirishda xatolik yuz berdi</p>`;
    }
  }
}

function renderSuggestions(movies, query) {
  if (!els.searchSuggestions) return;
  els.searchSuggestions.innerHTML = "";

  if (!query) {
    if (els.searchDefaultState) els.searchDefaultState.classList.remove("hidden");
    els.searchSuggestions.classList.add("hidden");
    return;
  }

  if (!movies || movies.length === 0) {
    els.searchSuggestions.innerHTML = `
      <div style="text-align:center; padding:40px 16px;">
        <div style="font-size:44px; margin-bottom:12px;">🔍</div>
        <h4 style="font-size:16px; font-weight:700; color:var(--text-main); margin-bottom:6px;">Hech narsa topilmadi</h4>
        <p style="font-size:13px; color:var(--text-sub); line-height:1.5; max-width:280px; margin:0 auto;">
          "<b>${escapeHtml(query)}</b>" bo'yicha ma'lumot topilmadi. Kino nomi yoki 4 xonali kodini tekshirib qayta kiriting.
        </p>
      </div>
    `;
    return;
  }

  movies.forEach((movie) => {
    const item = document.createElement("div");
    item.className = "search-item-card";
    const catSvg = CATEGORY_SVG[movie.category] || CATEGORY_SVG.kino;
    const poster = movie.poster_file_id
      ? `<img src="/api/poster/${movie.id}" alt="" loading="lazy" />`
      : `<div class="poster-fallback-vector" style="font-size:18px">${catSvg}</div>`;

    const catLabel = CATEGORY_LABEL[movie.category] || "Kino";
    const hasEps = movie.has_episodes || movie.is_series || (movie.episode_count > 1);
    const seriesExtra = hasEps && movie.episode_count > 0 ? ` (${movie.episode_count} qism)` : "";

    item.innerHTML = `
      <div class="search-item-poster">${poster}</div>
      <div class="search-item-info">
        <div class="search-item-title">${escapeHtml(movie.title)}</div>
        <div class="search-item-meta">
          <span>${catLabel}${seriesExtra}</span>
          <span>·</span>
          <span>👁 ${formatViews(movie.views)}</span>
          <span class="search-item-code-badge">🔑 ${movie.code}</span>
        </div>
      </div>
    `;

    item.addEventListener("click", () => {
      closeSearchModal();
      openModal(movie);
    });

    els.searchSuggestions.appendChild(item);
  });
}

// Qidiruv oynasini ochish uchun hodisalar
if (els.homeSearchTrigger) {
  els.homeSearchTrigger.addEventListener("click", () => openSearchModal());
}

if (els.headerSearchBtn) {
  els.headerSearchBtn.addEventListener("click", () => openSearchModal());
}

if (els.bottomNavSearch) {
  els.bottomNavSearch.addEventListener("click", (e) => {
    e.preventDefault();
    openSearchModal();
  });
}

if (els.searchModalBackBtn) {
  els.searchModalBackBtn.addEventListener("click", closeSearchModal);
}

if (els.searchModalInput) {
  els.searchModalInput.addEventListener("input", () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(triggerSearch, 220);
  });

  els.searchModalInput.addEventListener("keydown", (e) => {
    if (e.key === "Enter") {
      clearTimeout(debounceTimer);
      triggerSearch();
    }
  });
}

if (els.searchModalClearBtn) {
  els.searchModalClearBtn.addEventListener("click", () => {
    els.searchModalInput.value = "";
    els.searchModalClearBtn.classList.add("hidden");
    triggerSearch();
    els.searchModalInput.focus();
  });
}

if (els.searchTypeChips) {
  els.searchTypeChips.forEach((chip) => {
    chip.addEventListener("click", () => {
      els.searchTypeChips.forEach((c) => c.classList.remove("active"));
      chip.classList.add("active");
      state.searchType = chip.dataset.type;
      triggerSearch();
    });
  });
}

if (els.quickChips) {
  els.quickChips.forEach((chip) => {
    chip.addEventListener("click", () => {
      const q = chip.dataset.query;
      els.searchModalInput.value = q;
      if (els.searchModalClearBtn) els.searchModalClearBtn.classList.remove("hidden");
      triggerSearch();
    });
  });
}

// ---------- View Switcher ----------
function switchView(viewName) {
  state.currentView = viewName;
  els.navBtns.forEach((b) => b.classList.toggle("active", b.dataset.view === viewName));
  els.views.forEach((v) => {
    if (v.id === `view-${viewName}`) {
      v.classList.remove("hidden");
      v.classList.add("active-view");
    } else {
      v.classList.add("hidden");
      v.classList.remove("active-view");
    }
  });

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

// ---------- Admin Meta & Support ----------
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

// ---------- Application Initialization ----------
(async function init() {
  await loadMeta();
  try {
    const favData = await postJSON("/api/favorites", { initData: tg.initData });
    state.favoriteIds = new Set(favData.ids);
  } catch (e) {}
  loadHomeCarousels();
  updateCategorySections();
  loadHomeGrid();
})();
