package database

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"time"

	_ "github.com/tursodatabase/libsql-client-go/libsql"
	_ "modernc.org/sqlite"
)

var DB *sql.DB

type Movie struct {
	ID           int64   `json:"id"`
	Code         string  `json:"code"`
	Title        string  `json:"title"`
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	PosterFileID *string `json:"poster_file_id"`
	Views        int64   `json:"views"`
	EpisodeCount int     `json:"episode_count"`
	IsSeries     bool    `json:"is_series"`
	HasEpisodes  bool    `json:"has_episodes"`
}

type Episode struct {
	ID            int64 `json:"id"`
	EpisodeNumber int   `json:"episode_number"`
}

type EpisodeDetail struct {
	ID            int64  `json:"id"`
	EpisodeNumber int    `json:"episode_number"`
	VideoFileID   string `json:"video_file_id"`
	MovieID       int64  `json:"movie_id"`
	Title         string `json:"title"`
	Category      string `json:"category"`
}

type User struct {
	UserID    int64   `json:"user_id"`
	FirstName *string `json:"first_name"`
	Username  *string `json:"username"`
	JoinedAt  string  `json:"joined_at"`
}

type AdminInfo struct {
	UserID    int64   `json:"user_id"`
	AddedBy   *int64  `json:"added_by"`
	AddedAt   string  `json:"added_at"`
	FirstName *string `json:"first_name"`
	Username  *string `json:"username"`
}

type Stats struct {
	Users    int   `json:"users"`
	Kino     int   `json:"kino"`
	Multfilm int   `json:"multfilm"`
	Serial   int   `json:"serial"`
	Episodes int   `json:"episodes"`
	Views    int64 `json:"views"`
}

type MovieFilter struct {
	Category   string
	Search     string
	SearchType string
	Sort       string
	Limit      int
	DaysLimit  int
}

const movieSelect = `
	SELECT m.id, m.code, m.title, m.category, COALESCE(m.description, ''), m.poster_file_id, COALESCE(m.views, 0),
	       (SELECT COUNT(*) FROM movie_episodes e WHERE e.movie_id = m.id) AS episode_count
	FROM movies m
`

func InitDB(databaseURL, authToken string) error {
	var dsn string
	if databaseURL != "" {
		dsn = databaseURL
		if authToken != "" {
			if strings.Contains(dsn, "?") {
				dsn += "&authToken=" + authToken
			} else {
				dsn += "?authToken=" + authToken
			}
		}
	} else {
		dsn = "file:local.db"
	}

	var err error
	DB, err = sql.Open("libsql", dsn)
	if err != nil {
		return fmt.Errorf("sql.Open xatosi: %w", err)
	}

	createStatements := []string{
		`CREATE TABLE IF NOT EXISTS movies (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			title TEXT NOT NULL,
			category TEXT NOT NULL,
			description TEXT DEFAULT '',
			poster_file_id TEXT,
			video_file_id TEXT,
			views INTEGER DEFAULT 0,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS movie_episodes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			movie_id INTEGER NOT NULL,
			episode_number INTEGER NOT NULL,
			video_file_id TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS favorites (
			user_id INTEGER NOT NULL,
			movie_id INTEGER NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, movie_id)
		);`,
		`CREATE TABLE IF NOT EXISTS history (
			user_id INTEGER NOT NULL,
			movie_id INTEGER NOT NULL,
			watched_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			PRIMARY KEY (user_id, movie_id)
		);`,
		`CREATE TABLE IF NOT EXISTS bot_users (
			user_id INTEGER PRIMARY KEY,
			first_name TEXT,
			username TEXT,
			joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
			last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS admins (
			user_id INTEGER PRIMARY KEY,
			added_by INTEGER,
			added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, stmt := range createStatements {
		if _, err := DB.Exec(stmt); err != nil {
			return fmt.Errorf("sxema yaratishda xatolik (%s): %w", stmt, err)
		}
	}

	// Schema migratsiyasi (views ustuni mavjudligini tekshirish)
	rows, err := DB.Query("PRAGMA table_info(movies)")
	if err == nil {
		hasViews := false
		for rows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var dflt sql.NullString
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err == nil {
				if name == "views" {
					hasViews = true
				}
			}
		}
		rows.Close()
		if !hasViews {
			_, _ = DB.Exec("ALTER TABLE movies ADD COLUMN views INTEGER DEFAULT 0")
		}
	}

	log.Println("Ma'lumotlar bazasi muvaffaqiyatli ulandi va sozlandi.")
	return nil
}

func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}

func scanMovie(rows *sql.Rows) (*Movie, error) {
	var m Movie
	var poster sql.NullString
	if err := rows.Scan(&m.ID, &m.Code, &m.Title, &m.Category, &m.Description, &poster, &m.Views, &m.EpisodeCount); err != nil {
		return nil, err
	}
	if poster.Valid {
		m.PosterFileID = &poster.String
	}
	m.IsSeries = m.Category == "serial"
	m.HasEpisodes = m.EpisodeCount > 0 || m.IsSeries
	return &m, nil
}

func GenerateUniqueCode() (string, error) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 1000; i++ {
		code := strconv.Itoa(r.Intn(9000) + 1000)
		var id int64
		err := DB.QueryRow("SELECT id FROM movies WHERE code = ?", code).Scan(&id)
		if err == sql.ErrNoRows {
			return code, nil
		}
		if err != nil {
			return "", err
		}
	}
	return "", fmt.Errorf("unikal kod generatsiya qilib bo'lmadi")
}

func AddMovie(title, category, description, posterFileID, videoFileID string) (string, error) {
	code, err := GenerateUniqueCode()
	if err != nil {
		return "", err
	}
	var poster sql.NullString
	if posterFileID != "" {
		poster = sql.NullString{String: posterFileID, Valid: true}
	}
	_, err = DB.Exec(
		"INSERT INTO movies (code, title, category, description, poster_file_id, video_file_id) VALUES (?, ?, ?, ?, ?, ?)",
		code, title, category, description, poster, videoFileID,
	)
	return code, err
}

func AddMultipartMovie(title, category, description, posterFileID string) (string, int64, error) {
	code, err := GenerateUniqueCode()
	if err != nil {
		return "", 0, err
	}
	var poster sql.NullString
	if posterFileID != "" {
		poster = sql.NullString{String: posterFileID, Valid: true}
	}
	res, err := DB.Exec(
		"INSERT INTO movies (code, title, category, description, poster_file_id, video_file_id) VALUES (?, ?, ?, ?, ?, NULL)",
		code, title, category, description, poster,
	)
	if err != nil {
		return "", 0, err
	}
	movieID, err := res.LastInsertId()
	return code, movieID, err
}

func AddEpisode(movieID int64, videoFileID string) (int, error) {
	var maxEp sql.NullInt64
	_ = DB.QueryRow("SELECT MAX(episode_number) FROM movie_episodes WHERE movie_id = ?", movieID).Scan(&maxEp)
	nextNum := int(maxEp.Int64) + 1

	_, err := DB.Exec(
		"INSERT INTO movie_episodes (movie_id, episode_number, video_file_id) VALUES (?, ?, ?)",
		movieID, nextNum, videoFileID,
	)
	if err != nil {
		return 0, err
	}
	return nextNum, nil
}

func EnsureMovieHasEpisodes(movieID int64) (int, error) {
	episodes, err := GetEpisodes(movieID)
	if err != nil {
		return 0, err
	}
	if len(episodes) == 0 {
		videoID, err := GetMovieVideo(movieID)
		if err == nil && videoID != "" {
			_, err = AddEpisode(movieID, videoID)
			if err == nil {
				_, _ = DB.Exec("UPDATE movies SET video_file_id = NULL WHERE id = ?", movieID)
				return 1, nil
			}
		}
	}
	return len(episodes), nil
}

func GetEpisodes(movieID int64) ([]Episode, error) {
	rows, err := DB.Query("SELECT id, episode_number FROM movie_episodes WHERE movie_id = ? ORDER BY episode_number ASC", movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []Episode
	for rows.Next() {
		var ep Episode
		if err := rows.Scan(&ep.ID, &ep.EpisodeNumber); err == nil {
			list = append(list, ep)
		}
	}
	if list == nil {
		list = []Episode{}
	}
	return list, nil
}

func GetEpisodeByID(episodeID int64) (*EpisodeDetail, error) {
	row := DB.QueryRow(`
		SELECT e.id, e.episode_number, e.video_file_id, e.movie_id, m.title, m.category
		FROM movie_episodes e JOIN movies m ON m.id = e.movie_id
		WHERE e.id = ?
	`, episodeID)

	var ep EpisodeDetail
	err := row.Scan(&ep.ID, &ep.EpisodeNumber, &ep.VideoFileID, &ep.MovieID, &ep.Title, &ep.Category)
	if err != nil {
		return nil, err
	}
	return &ep, nil
}

func GetMovies(f MovieFilter) ([]Movie, error) {
	query := movieSelect + " WHERE 1=1"
	var params []interface{}
	var orderParams []interface{}
	customOrder := ""

	if f.Category != "" {
		query += " AND m.category = ?"
		params = append(params, f.Category)
	}

	if f.DaysLimit > 0 {
		query += " AND m.created_at >= datetime('now', '-' || ? || ' days')"
		params = append(params, f.DaysLimit)
	}

	if f.Search != "" {
		searchClean := strings.TrimSpace(f.Search)
		tokens := strings.Fields(searchClean)

		if f.SearchType == "code" {
			query += " AND m.code LIKE ?"
			params = append(params, "%"+searchClean+"%")
			customOrder = `
				ORDER BY 
					CASE 
						WHEN m.code = ? THEN 1
						WHEN m.code LIKE (? || '%') THEN 2
						ELSE 3
					END ASC, m.views DESC, m.id DESC
			`
			orderParams = []interface{}{searchClean, searchClean}
		} else if f.SearchType == "title" {
			conds := []string{"(LOWER(m.title) LIKE LOWER(?) OR LOWER(m.description) LIKE LOWER(?))"}
			params = append(params, "%"+searchClean+"%", "%"+searchClean+"%")
			for _, t := range tokens {
				conds = append(conds, "LOWER(m.title) LIKE LOWER(?)")
				params = append(params, "%"+t+"%")
			}
			query += " AND (" + strings.Join(conds, " OR ") + ")"
			customOrder = `
				ORDER BY 
					CASE 
						WHEN LOWER(m.title) = LOWER(?) THEN 1
						WHEN LOWER(m.title) LIKE LOWER(? || '%') THEN 2
						WHEN LOWER(m.title) LIKE LOWER('%' || ? || '%') THEN 3
						ELSE 4
					END ASC, m.views DESC, m.id DESC
			`
			orderParams = []interface{}{searchClean, searchClean, searchClean}
		} else {
			conds := []string{"(LOWER(m.title) LIKE LOWER(?) OR m.code LIKE ? OR LOWER(m.description) LIKE LOWER(?))"}
			params = append(params, "%"+searchClean+"%", "%"+searchClean+"%", "%"+searchClean+"%")
			for _, t := range tokens {
				conds = append(conds, "(LOWER(m.title) LIKE LOWER(?) OR m.code LIKE ?)")
				params = append(params, "%"+t+"%", "%"+t+"%")
			}
			query += " AND (" + strings.Join(conds, " OR ") + ")"
			customOrder = `
				ORDER BY 
					CASE 
						WHEN m.code = ? THEN 1
						WHEN LOWER(m.title) = LOWER(?) THEN 2
						WHEN m.code LIKE (? || '%') THEN 3
						WHEN LOWER(m.title) LIKE LOWER(? || '%') THEN 4
						WHEN LOWER(m.title) LIKE LOWER('%' || ? || '%') THEN 5
						WHEN m.code LIKE ('%' || ? || '%') THEN 6
						ELSE 7
					END ASC, m.views DESC, m.id DESC
			`
			orderParams = []interface{}{
				searchClean, searchClean, searchClean,
				searchClean, searchClean, searchClean,
			}
		}
	}

	if customOrder != "" {
		query += customOrder
		params = append(params, orderParams...)
	} else {
		if f.Sort == "top" {
			query += " ORDER BY m.views DESC, m.id DESC"
		} else {
			query += " ORDER BY m.id DESC"
		}
	}

	if f.Limit > 0 {
		query += " LIMIT ?"
		params = append(params, f.Limit)
	}

	rows, err := DB.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		m, err := scanMovie(rows)
		if err == nil {
			movies = append(movies, *m)
		}
	}
	if movies == nil {
		movies = []Movie{}
	}
	return movies, nil
}

func GetMovieByID(movieID int64) (*Movie, error) {
	rows, err := DB.Query(movieSelect+" WHERE m.id = ?", movieID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanMovie(rows)
	}
	return nil, sql.ErrNoRows
}

func GetMovieByCode(code string) (*Movie, error) {
	rows, err := DB.Query(movieSelect+" WHERE m.code = ?", strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanMovie(rows)
	}
	return nil, sql.ErrNoRows
}

func GetMovieVideo(movieID int64) (string, error) {
	var vid sql.NullString
	err := DB.QueryRow("SELECT video_file_id FROM movies WHERE id = ?", movieID).Scan(&vid)
	if err != nil {
		return "", err
	}
	return vid.String, nil
}

func IncrementViews(movieID int64) error {
	_, err := DB.Exec("UPDATE movies SET views = COALESCE(views, 0) + 1 WHERE id = ?", movieID)
	return err
}

func DeleteMovie(code string) (bool, error) {
	var movieID int64
	err := DB.QueryRow("SELECT id FROM movies WHERE code = ?", code).Scan(&movieID)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	_, _ = DB.Exec("DELETE FROM movies WHERE code = ?", code)
	_, _ = DB.Exec("DELETE FROM movie_episodes WHERE movie_id = ?", movieID)
	_, _ = DB.Exec("DELETE FROM favorites WHERE movie_id = ?", movieID)
	_, _ = DB.Exec("DELETE FROM history WHERE movie_id = ?", movieID)
	return true, nil
}

func ToggleFavorite(userID, movieID int64) (bool, error) {
	var dummy int
	err := DB.QueryRow("SELECT 1 FROM favorites WHERE user_id = ? AND movie_id = ?", userID, movieID).Scan(&dummy)
	if err == nil {
		_, err = DB.Exec("DELETE FROM favorites WHERE user_id = ? AND movie_id = ?", userID, movieID)
		return false, err
	}
	_, err = DB.Exec("INSERT INTO favorites (user_id, movie_id) VALUES (?, ?)", userID, movieID)
	return true, err
}

func GetFavoriteIDs(userID int64) ([]int64, error) {
	rows, err := DB.Query("SELECT movie_id FROM favorites WHERE user_id = ?", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if ids == nil {
		ids = []int64{}
	}
	return ids, nil
}

func GetFavoriteMovies(userID int64) ([]Movie, error) {
	query := movieSelect + " JOIN favorites f ON m.id = f.movie_id WHERE f.user_id = ? ORDER BY f.created_at DESC"
	rows, err := DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		m, err := scanMovie(rows)
		if err == nil {
			movies = append(movies, *m)
		}
	}
	if movies == nil {
		movies = []Movie{}
	}
	return movies, nil
}

func AddHistory(userID, movieID int64) error {
	_, err := DB.Exec(`
		INSERT INTO history (user_id, movie_id, watched_at) VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id, movie_id) DO UPDATE SET watched_at = CURRENT_TIMESTAMP
	`, userID, movieID)
	return err
}

func GetHistoryMovies(userID int64, limit int) ([]Movie, error) {
	if limit <= 0 {
		limit = 30
	}
	query := movieSelect + ` JOIN history h ON m.id = h.movie_id WHERE h.user_id = ? ORDER BY h.watched_at DESC LIMIT ?`
	rows, err := DB.Query(query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var movies []Movie
	for rows.Next() {
		m, err := scanMovie(rows)
		if err == nil {
			movies = append(movies, *m)
		}
	}
	if movies == nil {
		movies = []Movie{}
	}
	return movies, nil
}

func GetHistoryCount(userID int64) (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM history WHERE user_id = ?", userID).Scan(&count)
	return count, err
}

func TrackUser(userID int64, firstName, username string) error {
	var fn, un sql.NullString
	if firstName != "" {
		fn = sql.NullString{String: firstName, Valid: true}
	}
	if username != "" {
		un = sql.NullString{String: username, Valid: true}
	}

	_, err := DB.Exec(`
		INSERT INTO bot_users (user_id, first_name, username, joined_at, last_seen)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
			first_name = excluded.first_name,
			username = excluded.username,
			last_seen = CURRENT_TIMESTAMP
	`, userID, fn, un)
	return err
}

func GetUserCount() (int, error) {
	var count int
	err := DB.QueryRow("SELECT COUNT(*) FROM bot_users").Scan(&count)
	return count, err
}

func GetRecentUsers(limit int) ([]User, error) {
	if limit <= 0 {
		limit = 30
	}
	rows, err := DB.Query("SELECT user_id, first_name, username, joined_at FROM bot_users ORDER BY joined_at DESC LIMIT ?", limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var u User
		var fn, un sql.NullString
		if err := rows.Scan(&u.UserID, &fn, &un, &u.JoinedAt); err == nil {
			if fn.Valid {
				u.FirstName = &fn.String
			}
			if un.Valid {
				u.Username = &un.String
			}
			users = append(users, u)
		}
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

func GetStats() (Stats, error) {
	var s Stats
	s.Users, _ = GetUserCount()

	rows, err := DB.Query("SELECT category, COUNT(*) FROM movies GROUP BY category")
	if err == nil {
		for rows.Next() {
			var cat string
			var count int
			if err := rows.Scan(&cat, &count); err == nil {
				switch cat {
				case "kino":
					s.Kino = count
				case "multfilm":
					s.Multfilm = count
				case "serial":
					s.Serial = count
				}
			}
		}
		rows.Close()
	}

	_ = DB.QueryRow("SELECT COUNT(*) FROM movie_episodes").Scan(&s.Episodes)
	var totalViews sql.NullInt64
	_ = DB.QueryRow("SELECT COALESCE(SUM(views), 0) FROM movies").Scan(&totalViews)
	s.Views = totalViews.Int64

	return s, nil
}

func AddAdmin(userID, addedBy int64) error {
	_, err := DB.Exec("INSERT OR IGNORE INTO admins (user_id, added_by) VALUES (?, ?)", userID, addedBy)
	return err
}

func RemoveAdmin(userID int64) (bool, error) {
	res, err := DB.Exec("DELETE FROM admins WHERE user_id = ?", userID)
	if err != nil {
		return false, err
	}
	rowsAffected, _ := res.RowsAffected()
	return rowsAffected > 0, nil
}

func IsAdmin(userID int64) (bool, error) {
	var dummy int
	err := DB.QueryRow("SELECT 1 FROM admins WHERE user_id = ?", userID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func GetDBAdmins() ([]AdminInfo, error) {
	rows, err := DB.Query(`
		SELECT a.user_id, a.added_by, a.added_at, u.first_name, u.username
		FROM admins a LEFT JOIN bot_users u ON a.user_id = u.user_id
		ORDER BY a.added_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var admins []AdminInfo
	for rows.Next() {
		var a AdminInfo
		var fn, un sql.NullString
		var ab sql.NullInt64
		if err := rows.Scan(&a.UserID, &ab, &a.AddedAt, &fn, &un); err == nil {
			if ab.Valid {
				a.AddedBy = &ab.Int64
			}
			if fn.Valid {
				a.FirstName = &fn.String
			}
			if un.Valid {
				a.Username = &un.String
			}
			admins = append(admins, a)
		}
	}
	if admins == nil {
		admins = []AdminInfo{}
	}
	return admins, nil
}

func GetUserByUsernameOrID(query string) (*User, error) {
	q := strings.TrimPrefix(strings.TrimSpace(query), "@")
	if uid, err := strconv.ParseInt(q, 10, 64); err == nil {
		var u User
		var fn, un sql.NullString
		err := DB.QueryRow("SELECT user_id, first_name, username, joined_at FROM bot_users WHERE user_id = ?", uid).Scan(&u.UserID, &fn, &un, &u.JoinedAt)
		if err == sql.ErrNoRows {
			return &User{UserID: uid}, nil
		}
		if err != nil {
			return nil, err
		}
		if fn.Valid {
			u.FirstName = &fn.String
		}
		if un.Valid {
			u.Username = &un.String
		}
		return &u, nil
	}

	var u User
	var fn, un sql.NullString
	err := DB.QueryRow("SELECT user_id, first_name, username, joined_at FROM bot_users WHERE LOWER(username) = LOWER(?)", q).Scan(&u.UserID, &fn, &un, &u.JoinedAt)
	if err != nil {
		return nil, err
	}
	if fn.Valid {
		u.FirstName = &fn.String
	}
	if un.Valid {
		u.Username = &un.String
	}
	return &u, nil
}
