package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"kinobot/bot"
	"kinobot/config"
	"kinobot/database"
	"kinobot/security"
)

type Server struct {
	cfg *config.Config
	mux *http.ServeMux
}

func NewServer(cfg *config.Config) *Server {
	s := &Server{
		cfg: cfg,
		mux: http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Static files & root
	fileServer := http.FileServer(http.Dir("static"))
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", fileServer))
	s.mux.HandleFunc("GET /{$}", s.handleIndex)

	// Health & Ping (UptimeRobot)
	s.mux.HandleFunc("GET /health", s.handleHealth)
	s.mux.HandleFunc("HEAD /health", s.handleHealth)
	s.mux.HandleFunc("GET /api/ping", s.handlePing)

	// Telegram Webhook
	if s.cfg.WebhookPath != "" {
		s.mux.HandleFunc("POST "+s.cfg.WebhookPath, s.handleWebhook)
	}

	// APIs
	s.mux.HandleFunc("GET /api/meta", s.handleMeta)
	s.mux.HandleFunc("GET /api/movies", s.handleMovies)
	s.mux.HandleFunc("GET /api/movie/{id}", s.handleMovie)
	s.mux.HandleFunc("GET /api/poster/{id}", s.handlePoster)
	s.mux.HandleFunc("POST /api/watch/{id}", s.handleWatch)
	s.mux.HandleFunc("GET /api/episodes/{id}", s.handleEpisodes)
	s.mux.HandleFunc("POST /api/watch-episode/{id}", s.handleWatchEpisode)
	s.mux.HandleFunc("POST /api/favorite/{id}", s.handleFavoriteToggle)
	s.mux.HandleFunc("POST /api/favorites", s.handleFavoritesList)
	s.mux.HandleFunc("POST /api/history", s.handleHistory)
}

func (s *Server) Start() error {
	handler := s.withMiddlewares(s.mux)
	addr := ":" + s.cfg.Port
	log.Printf("Kinobot Go server ishga tushdi: http://0.0.0.0%s", addr)
	return http.ListenAndServe(addr, handler)
}

func (s *Server) withMiddlewares(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// No-cache headers
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		w.Header().Set("Pragma", "no-cache")
		w.Header().Set("Expires", "0")

		// CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, HEAD")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join("static", "index.html"))
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePing(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"pong":   true,
	})
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	var update tgbotapi.Update
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		log.Printf("Webhook decode xatosi: %v", err)
		s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
		return
	}

	go bot.HandleUpdate(update, s.cfg)
	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleMeta(w http.ResponseWriter, r *http.Request) {
	s.writeJSON(w, http.StatusOK, map[string]string{
		"admin_username": s.cfg.AdminUsername,
	})
}

func (s *Server) handleMovies(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	filter := database.MovieFilter{
		Category:   q.Get("category"),
		Search:     q.Get("search"),
		SearchType: q.Get("search_type"),
		Sort:       q.Get("sort"),
	}
	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if daysStr := q.Get("days_limit"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil {
			filter.DaysLimit = days
		}
	}

	movies, err := database.GetMovies(filter)
	if err != nil {
		log.Printf("GetMovies xatosi: %v", err)
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, movies)
}

func (s *Server) handleMovie(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	movieID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Noto'g'ri ID"})
		return
	}

	movie, err := database.GetMovieByID(movieID)
	if err != nil || movie == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Topilmadi"})
		return
	}
	s.writeJSON(w, http.StatusOK, movie)
}

func (s *Server) handlePoster(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	movieID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Noto'g'ri ID", http.StatusBadRequest)
		return
	}

	movie, err := database.GetMovieByID(movieID)
	if err != nil || movie == nil || movie.PosterFileID == nil || *movie.PosterFileID == "" {
		http.Error(w, "Poster yo'q", http.StatusNotFound)
		return
	}

	cachedPath, err := bot.DownloadTelegramPoster(*movie.PosterFileID, s.cfg)
	if err != nil {
		log.Printf("Poster yuklashda xato (movie_id=%d): %v", movieID, err)
		http.Error(w, "Poster yuklashda xatolik", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	http.ServeFile(w, r, cachedPath)
}

type authRequest struct {
	InitData string `json:"initData"`
}

func (s *Server) requireUser(r *http.Request) (*security.TelegramUser, error) {
	var body authRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		return nil, fmt.Errorf("Noto'g'ri JSON tanasi")
	}

	user, err := security.ValidateInitData(body.InitData, s.cfg.BotToken)
	if err != nil || user == nil {
		return nil, fmt.Errorf("Tekshiruvdan o'tmadi")
	}

	_ = database.TrackUser(user.ID, user.FirstName, user.Username)
	return user, nil
}

func (s *Server) handleWatch(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	movieID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Noto'g'ri ID"})
		return
	}

	user, err := s.requireUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	movie, err := database.GetMovieByID(movieID)
	if err != nil || movie == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Topilmadi"})
		return
	}

	if movie.HasEpisodes || movie.IsSeries || movie.EpisodeCount > 0 {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Bu ko'p qismli — iltimos kerakli qismni tanlang"})
		return
	}

	videoID, err := database.GetMovieVideo(movieID)
	if err != nil || videoID == "" {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Video fayli topilmadi"})
		return
	}

	caption := fmt.Sprintf("🎬 <b>%s</b>\n\n%s", movie.Title, movie.Description)
	if err := bot.SendVideoToUser(user.ID, videoID, caption); err != nil {
		log.Printf("SendVideoToUser xatosi: %v", err)
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Video yuborishda xato: %v", err)})
		return
	}

	_ = database.IncrementViews(movieID)
	_ = database.AddHistory(user.ID, movieID)

	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleEpisodes(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	movieID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Noto'g'ri ID"})
		return
	}

	episodes, err := database.GetEpisodes(movieID)
	if err != nil {
		log.Printf("GetEpisodes xatosi: %v", err)
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.writeJSON(w, http.StatusOK, episodes)
}

func (s *Server) handleWatchEpisode(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	episodeID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Noto'g'ri ID"})
		return
	}

	user, err := s.requireUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	episode, err := database.GetEpisodeByID(episodeID)
	if err != nil || episode == nil {
		s.writeJSON(w, http.StatusNotFound, map[string]string{"error": "Topilmadi"})
		return
	}

	catEmoji := "🎬"
	if episode.Category == "serial" {
		catEmoji = "📺"
	} else if episode.Category == "multfilm" {
		catEmoji = "🧸"
	}
	caption := fmt.Sprintf("%s <b>%s</b> — %d-qism", catEmoji, episode.Title, episode.EpisodeNumber)

	if err := bot.SendVideoToUser(user.ID, episode.VideoFileID, caption); err != nil {
		log.Printf("SendVideoToUser xatosi: %v", err)
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": fmt.Sprintf("Video yuborishda xato: %v", err)})
		return
	}

	_ = database.IncrementViews(episode.MovieID)
	_ = database.AddHistory(user.ID, episode.MovieID)

	s.writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleFavoriteToggle(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	movieID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		s.writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Noto'g'ri ID"})
		return
	}

	user, err := s.requireUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	favorited, err := database.ToggleFavorite(user.ID, movieID)
	if err != nil {
		s.writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]bool{"favorited": favorited})
}

func (s *Server) handleFavoritesList(w http.ResponseWriter, r *http.Request) {
	user, err := s.requireUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	ids, _ := database.GetFavoriteIDs(user.ID)
	movies, _ := database.GetFavoriteMovies(user.ID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"ids":    ids,
		"movies": movies,
	})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	user, err := s.requireUser(r)
	if err != nil {
		s.writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	movies, _ := database.GetHistoryMovies(user.ID, 30)
	count, _ := database.GetHistoryCount(user.ID)

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"movies": movies,
		"count":  count,
	})
}
