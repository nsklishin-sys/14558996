package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
)

type greetingResponse struct {
	Message string `json:"message"`
}

type user struct {
	ID          int64  `json:"id"`
	PublicID    string `json:"public_id"`
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	FullName    string `json:"full_name"`
	Email       string `json:"email"`
	Position    string `json:"position,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	Bio         string `json:"bio,omitempty"`
	Phone       string `json:"phone,omitempty"`
	Location    string `json:"location,omitempty"`
	City        string `json:"city,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
}

type registerRequest struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	Password  string `json:"password"`
	Terms     bool   `json:"terms"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type profileUpdateRequest struct {
	FirstName   string `json:"first_name"`
	LastName    string `json:"last_name"`
	FullName    string `json:"full_name"`
	Position    string `json:"position"`
	CompanyName string `json:"company_name"`
	Bio         string `json:"bio"`
	Phone       string `json:"phone"`
	Location    string `json:"location"`
	City        string `json:"city"`
	AvatarURL   string `json:"avatar_url"`
}

type authResponse struct {
	Token string `json:"token"`
	User  user   `json:"user"`
}

type publicUserProfile struct {
	PublicID  string `json:"public_id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
}

type friendDTO struct {
	FriendID    string `json:"friend_id"`
	FriendName  string `json:"friend_name"`
	Email       string `json:"email"`
	Position    string `json:"position,omitempty"`
	CompanyName string `json:"company_name,omitempty"`
	IsOnline    bool   `json:"is_online"`
}

type friendCandidateDTO struct {
	ID       string `json:"id"`
	FullName string `json:"full_name"`
	Email    string `json:"email"`
}

type community struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags"`
	Privacy     string    `json:"privacy_level"`
	CreatorID   string    `json:"creator_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type createCommunityRequest struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Tags         []string `json:"tags"`
	PrivacyLevel string   `json:"privacy_level"`
}

type sessionStore struct {
	mu         sync.RWMutex
	tokenToUID map[string]int64
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		tokenToUID: make(map[string]int64),
	}
}

func (s *sessionStore) put(token string, userID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tokenToUID[token] = userID
}

func (s *sessionStore) getUserID(token string) (int64, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	userID, ok := s.tokenToUID[token]
	return userID, ok
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func main() {
	db, err := initDBFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	sessions := newSessionStore()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			writeError(w, http.StatusServiceUnavailable, "database down")
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "up"})
	})

	mux.HandleFunc("/api/greeting", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, greetingResponse{Message: "Привет! Добро пожаловать в новый проект на Go + TypeScript 🚀"})
	})

	mux.HandleFunc("/api/auth/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		var req registerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}

		createdUser, err := createUser(db, req)
		if err != nil {
			if errors.Is(err, errValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if errors.Is(err, errEmailTaken) {
				writeError(w, http.StatusConflict, "Пользователь с таким email уже существует")
				return
			}
			log.Printf("register error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		token, err := newToken()
		if err != nil {
			log.Printf("token generate error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusCreated, authResponse{Token: token, User: createdUser})
		sessions.put(token, createdUser.ID)
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		var req loginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}

		authUser, err := loginUser(db, req)
		if err != nil {
			if errors.Is(err, errInvalidCredentials) {
				writeError(w, http.StatusUnauthorized, "Неверный email или пароль")
				return
			}
			if errors.Is(err, errValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("login error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		token, err := newToken()
		if err != nil {
			log.Printf("token generate error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, authResponse{Token: token, User: authUser})
		sessions.put(token, authUser.ID)
	})

	mux.HandleFunc("/api/me", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		authUser, err := getUserByID(db, userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusUnauthorized, "Пользователь не найден")
				return
			}
			log.Printf("get current user error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"user": authUser})
	})

	mux.HandleFunc("/api/profile", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		switch r.Method {
		case http.MethodGet:
			u, err := getUserByID(db, userID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Пользователь не найден")
					return
				}
				log.Printf("get profile error: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"user": u})
		case http.MethodPut:
			var req profileUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			updatedUser, err := updateUserProfile(db, userID, req)
			if err != nil {
				if errors.Is(err, errValidation) {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				if errors.Is(err, sql.ErrNoRows) {
					writeError(w, http.StatusNotFound, "Пользователь не найден")
					return
				}
				log.Printf("update profile error: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"user": updatedUser})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		publicID := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/users/"))
		if publicID == "" {
			writeError(w, http.StatusBadRequest, "public_id обязателен")
			return
		}

		profile, err := getPublicUserProfile(db, publicID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "Пользователь не найден")
				return
			}
			log.Printf("get public user profile error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"user": profile})
	})

	mux.HandleFunc("/api/friends", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}

		friends, err := listFriends(db, userID)
		if err != nil {
			log.Printf("list friends error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"friends": friends})
	})

	mux.HandleFunc("/api/friends/requests/incoming", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		requests, err := listIncomingFriendRequests(db, userID)
		if err != nil {
			log.Printf("list incoming friend requests error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
	})

	mux.HandleFunc("/api/friends/requests/outgoing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		requests, err := listOutgoingFriendRequests(db, userID)
		if err != nil {
			log.Printf("list outgoing friend requests error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"requests": requests})
	})

	mux.HandleFunc("/api/friends/candidates", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		query := strings.TrimSpace(r.URL.Query().Get("q"))
		candidates, err := listFriendCandidates(db, userID, query)
		if err != nil {
			log.Printf("list friend candidates error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"users": candidates})
	})

	mux.HandleFunc("/api/friends/request-by-name", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Некорректный JSON")
			return
		}
		targetPublicID, err := findUserPublicIDByNameOrEmail(db, userID, req.Name)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "Пользователь не найден")
				return
			}
			if errors.Is(err, errValidation) {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			log.Printf("find user by name/email error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}
		if err := createFriendRequest(db, userID, targetPublicID); err != nil {
			handleFriendActionError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/friends/", func(w http.ResponseWriter, r *http.Request) {
		userID, ok := authenticatedUserID(w, r, sessions)
		if !ok {
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/api/friends/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, "Некорректный путь")
			return
		}

		switch {
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "request":
			if err := createFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "accept":
			if err := acceptFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "reject":
			if err := rejectFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case r.Method == http.MethodPost && len(parts) == 2 && parts[0] == "cancel":
			if err := cancelFriendRequest(db, userID, parts[1]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case r.Method == http.MethodDelete && len(parts) == 1:
			if err := removeFriend(db, userID, parts[0]); err != nil {
				handleFriendActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/communities", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			communities, err := listCommunities(db)
			if err != nil {
				log.Printf("list communities error: %v", err)
				writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"communities": communities})
		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}

			var req createCommunityRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}

			created, err := createCommunity(db, userID, req)
			if err != nil {
				switch {
				case errors.Is(err, errValidation):
					writeError(w, http.StatusBadRequest, err.Error())
				default:
					log.Printf("create community error: %v", err)
					writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				}
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"community": created})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.Handle("/", http.FileServer(http.Dir("./web")))

	addr := ":8080"
	log.Printf("Server started at http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func initDBFromEnv() (*sql.DB, error) {
	databaseURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open postgres db: %w", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := waitForDB(db, 30*time.Second); err != nil {
		_ = db.Close()
		return nil, err
	}

	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func resolveDatabaseURL() (string, error) {
	const railwayDatabaseURL = "postgresql://postgres:QHkIHPzHfSeSKkQnEDFkJmjQJSpUpXpb@shinkansen.proxy.rlwy.net:19703/railway"

	if directURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); directURL != "" {
		return directURL, nil
	}

	return railwayDatabaseURL, nil
}

func waitForDB(db *sql.DB, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		if err := db.PingContext(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("database ping timeout after %s: %w", timeout, ctx.Err())
		case <-ticker.C:
		}
	}
}

func ensureSchema(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS public_id TEXT,
    ADD COLUMN IF NOT EXISTS position TEXT,
    ADD COLUMN IF NOT EXISTS company_name TEXT,
    ADD COLUMN IF NOT EXISTS bio TEXT,
    ADD COLUMN IF NOT EXISTS phone TEXT,
    ADD COLUMN IF NOT EXISTS location TEXT,
    ADD COLUMN IF NOT EXISTS city TEXT,
    ADD COLUMN IF NOT EXISTS avatar_url TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS users_public_id_uniq_idx ON users(public_id);

UPDATE users
SET public_id = CONCAT('u', id)
WHERE public_id IS NULL OR public_id = '';

CREATE TABLE IF NOT EXISTS friend_requests (
    id BIGSERIAL PRIMARY KEY,
    requester_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    addressee_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (requester_id <> addressee_id),
    CHECK (status IN ('pending', 'accepted', 'rejected', 'canceled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS friend_requests_pair_uniq_idx
ON friend_requests(requester_id, addressee_id);

CREATE INDEX IF NOT EXISTS friend_requests_addressee_status_idx
ON friend_requests(addressee_id, status);

CREATE INDEX IF NOT EXISTS friend_requests_requester_status_idx
ON friend_requests(requester_id, status);

CREATE TABLE IF NOT EXISTS communities (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    tags TEXT[] NOT NULL DEFAULT '{}',
    privacy_level TEXT NOT NULL DEFAULT 'open',
    creator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (privacy_level IN ('open', 'closed'))
);

CREATE INDEX IF NOT EXISTS communities_created_at_idx
ON communities(created_at DESC);
`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	return nil
}

var (
	errValidation         = errors.New("validation error")
	errEmailTaken         = errors.New("email taken")
	errInvalidCredentials = errors.New("invalid credentials")
	errConflict           = errors.New("conflict")
	errNotFound           = errors.New("not found")
)

func createUser(db *sql.DB, req registerRequest) (user, error) {
	if strings.TrimSpace(req.FirstName) == "" {
		return user{}, fmt.Errorf("%w: имя обязательно", errValidation)
	}
	if strings.TrimSpace(req.LastName) == "" {
		return user{}, fmt.Errorf("%w: фамилия обязательна", errValidation)
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !emailRe.MatchString(email) {
		return user{}, fmt.Errorf("%w: email некорректен", errValidation)
	}
	if len(req.Password) < 8 {
		return user{}, fmt.Errorf("%w: пароль должен быть не короче 8 символов", errValidation)
	}
	if !req.Terms {
		return user{}, fmt.Errorf("%w: необходимо принять условия", errValidation)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return user{}, fmt.Errorf("hash password: %w", err)
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		fullName = strings.TrimSpace(req.FirstName + " " + req.LastName)
	}

	for attempts := 0; attempts < 5; attempts++ {
		publicID, genErr := newPublicUserID()
		if genErr != nil {
			return user{}, fmt.Errorf("generate public_id: %w", genErr)
		}

		var created user
		err = db.QueryRow(`
			INSERT INTO users(public_id, first_name, last_name, full_name, email, password_hash)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, public_id, first_name, last_name, full_name, email,
				COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
				COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, '')
		`, publicID, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), fullName, email, string(hash)).
			Scan(&created.ID, &created.PublicID, &created.FirstName, &created.LastName, &created.FullName, &created.Email, &created.Position, &created.CompanyName, &created.Bio, &created.Phone, &created.Location, &created.City, &created.AvatarURL)
		if err == nil {
			return created, nil
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
				return user{}, errEmailTaken
			}
			if pgErr.Code == "23505" {
				continue
			}
		}

		return user{}, err
	}

	return user{}, fmt.Errorf("failed to allocate public_id")
}

func loginUser(db *sql.DB, req loginRequest) (user, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if !emailRe.MatchString(email) {
		return user{}, fmt.Errorf("%w: email некорректен", errValidation)
	}
	if strings.TrimSpace(req.Password) == "" {
		return user{}, fmt.Errorf("%w: пароль обязателен", errValidation)
	}

	var u user
	var passwordHash string
	err := db.QueryRow(`
		SELECT id, public_id, first_name, last_name, full_name, email,
			COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
			COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, ''),
			password_hash
		FROM users
		WHERE email = $1
	`, email).Scan(&u.ID, &u.PublicID, &u.FirstName, &u.LastName, &u.FullName, &u.Email, &u.Position, &u.CompanyName, &u.Bio, &u.Phone, &u.Location, &u.City, &u.AvatarURL, &passwordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return user{}, errInvalidCredentials
		}
		return user{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return user{}, errInvalidCredentials
	}

	return u, nil
}

func getPublicUserProfile(db *sql.DB, publicID string) (publicUserProfile, error) {
	var profile publicUserProfile
	err := db.QueryRow(`
		SELECT public_id, first_name, last_name, full_name, email
		FROM users
		WHERE public_id = $1
	`, publicID).Scan(&profile.PublicID, &profile.FirstName, &profile.LastName, &profile.FullName, &profile.Email)
	if err != nil {
		return publicUserProfile{}, err
	}

	return profile, nil
}

func getUserByID(db *sql.DB, userID int64) (user, error) {
	var u user
	err := db.QueryRow(`
		SELECT id, public_id, first_name, last_name, full_name, email,
			COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
			COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, '')
		FROM users
		WHERE id = $1
	`, userID).Scan(&u.ID, &u.PublicID, &u.FirstName, &u.LastName, &u.FullName, &u.Email, &u.Position, &u.CompanyName, &u.Bio, &u.Phone, &u.Location, &u.City, &u.AvatarURL)
	if err != nil {
		return user{}, err
	}

	return u, nil
}

func authenticatedUserID(w http.ResponseWriter, r *http.Request, sessions *sessionStore) (int64, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		writeError(w, http.StatusUnauthorized, "Требуется авторизация")
		return 0, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		writeError(w, http.StatusUnauthorized, "Требуется авторизация")
		return 0, false
	}
	userID, ok := sessions.getUserID(token)
	if !ok {
		writeError(w, http.StatusUnauthorized, "Сессия не найдена")
		return 0, false
	}
	return userID, true
}

func handleFriendActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
	}
}

func listFriends(db *sql.DB, userID int64) ([]friendDTO, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM friend_requests fr
		JOIN users u ON u.id = CASE
			WHEN fr.requester_id = $1 THEN fr.addressee_id
			ELSE fr.requester_id
		END
		WHERE fr.status = 'accepted'
		  AND ($1 IN (fr.requester_id, fr.addressee_id))
		ORDER BY u.full_name
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendDTO
	for rows.Next() {
		var dto friendDTO
		if scanErr := rows.Scan(&dto.FriendID, &dto.FriendName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func listIncomingFriendRequests(db *sql.DB, userID int64) ([]friendDTO, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM friend_requests fr
		JOIN users u ON u.id = fr.requester_id
		WHERE fr.addressee_id = $1
		  AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendDTO
	for rows.Next() {
		var dto friendDTO
		if scanErr := rows.Scan(&dto.FriendID, &dto.FriendName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func listOutgoingFriendRequests(db *sql.DB, userID int64) ([]friendDTO, error) {
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM friend_requests fr
		JOIN users u ON u.id = fr.addressee_id
		WHERE fr.requester_id = $1
		  AND fr.status = 'pending'
		ORDER BY fr.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendDTO
	for rows.Next() {
		var dto friendDTO
		if scanErr := rows.Scan(&dto.FriendID, &dto.FriendName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func listFriendCandidates(db *sql.DB, userID int64, query string) ([]friendCandidateDTO, error) {
	search := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM users u
		WHERE u.id <> $1
		  AND (
		      $2 = '%%'
		      OR LOWER(u.full_name) LIKE $2
		      OR LOWER(u.email) LIKE $2
		  )
		  AND NOT EXISTS (
		      SELECT 1
		      FROM friend_requests fr
		      WHERE (
		           (fr.requester_id = $1 AND fr.addressee_id = u.id)
		        OR (fr.requester_id = u.id AND fr.addressee_id = $1)
		      )
		      AND fr.status IN ('pending', 'accepted')
		  )
		ORDER BY u.full_name
		LIMIT 50
	`, userID, search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []friendCandidateDTO
	for rows.Next() {
		var dto friendCandidateDTO
		if scanErr := rows.Scan(&dto.ID, &dto.FullName, &dto.Email); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, dto)
	}
	return result, rows.Err()
}

func findUserPublicIDByNameOrEmail(db *sql.DB, userID int64, value string) (string, error) {
	v := strings.TrimSpace(value)
	if v == "" {
		return "", fmt.Errorf("%w: укажите имя или email", errValidation)
	}
	vLower := strings.ToLower(v)
	var publicID string
	err := db.QueryRow(`
		SELECT public_id
		FROM users
		WHERE id <> $1
		  AND (LOWER(email) = $2 OR LOWER(full_name) = $2)
		ORDER BY CASE WHEN LOWER(email) = $2 THEN 0 ELSE 1 END, id
		LIMIT 1
	`, userID, vLower).Scan(&publicID)
	return publicID, err
}

func createFriendRequest(db *sql.DB, requesterID int64, targetPublicID string) error {
	targetPublicID = strings.TrimSpace(targetPublicID)
	if targetPublicID == "" {
		return fmt.Errorf("%w: id пользователя обязателен", errValidation)
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var addresseeID int64
	if err := tx.QueryRow(`SELECT id FROM users WHERE public_id = $1`, targetPublicID).Scan(&addresseeID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: пользователь не найден", errNotFound)
		}
		return err
	}
	if addresseeID == requesterID {
		return fmt.Errorf("%w: нельзя добавить себя в друзья", errValidation)
	}

	var existingStatus string
	var existingRequesterID, existingAddresseeID int64
	err = tx.QueryRow(`
		SELECT requester_id, addressee_id, status
		FROM friend_requests
		WHERE (requester_id = $1 AND addressee_id = $2)
		   OR (requester_id = $2 AND addressee_id = $1)
		LIMIT 1
	`, requesterID, addresseeID).Scan(&existingRequesterID, &existingAddresseeID, &existingStatus)
	if err == nil {
		switch existingStatus {
		case "accepted":
			return fmt.Errorf("%w: вы уже друзья", errConflict)
		case "pending":
			if existingRequesterID == requesterID {
				return fmt.Errorf("%w: заявка уже отправлена", errConflict)
			}
			if _, err = tx.Exec(`
				UPDATE friend_requests
				SET status = 'accepted', updated_at = NOW()
				WHERE requester_id = $1 AND addressee_id = $2
			`, existingRequesterID, existingAddresseeID); err != nil {
				return err
			}
			return tx.Commit()
		case "rejected", "canceled":
			if _, err = tx.Exec(`
				UPDATE friend_requests
				SET requester_id = $1, addressee_id = $2, status = 'pending', updated_at = NOW()
				WHERE requester_id = $3 AND addressee_id = $4
			`, requesterID, addresseeID, existingRequesterID, existingAddresseeID); err != nil {
				return err
			}
			return tx.Commit()
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}

	if _, err = tx.Exec(`
		INSERT INTO friend_requests(requester_id, addressee_id, status)
		VALUES ($1, $2, 'pending')
	`, requesterID, addresseeID); err != nil {
		return err
	}
	return tx.Commit()
}

func acceptFriendRequest(db *sql.DB, userID int64, requesterPublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'accepted', updated_at = NOW()
		FROM users u
		WHERE fr.requester_id = u.id
		  AND fr.addressee_id = $1
		  AND fr.status = 'pending'
		  AND u.public_id = $2
	`, userID, strings.TrimSpace(requesterPublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: заявка не найдена", errNotFound)
	}
	return nil
}

func rejectFriendRequest(db *sql.DB, userID int64, requesterPublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'rejected', updated_at = NOW()
		FROM users u
		WHERE fr.requester_id = u.id
		  AND fr.addressee_id = $1
		  AND fr.status = 'pending'
		  AND u.public_id = $2
	`, userID, strings.TrimSpace(requesterPublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: заявка не найдена", errNotFound)
	}
	return nil
}

func cancelFriendRequest(db *sql.DB, userID int64, addresseePublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'canceled', updated_at = NOW()
		FROM users u
		WHERE fr.addressee_id = u.id
		  AND fr.requester_id = $1
		  AND fr.status = 'pending'
		  AND u.public_id = $2
	`, userID, strings.TrimSpace(addresseePublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: заявка не найдена", errNotFound)
	}
	return nil
}

func removeFriend(db *sql.DB, userID int64, friendPublicID string) error {
	res, err := db.Exec(`
		UPDATE friend_requests fr
		SET status = 'canceled', updated_at = NOW()
		FROM users u
		WHERE u.public_id = $2
		  AND fr.status = 'accepted'
		  AND (
		     (fr.requester_id = $1 AND fr.addressee_id = u.id)
		     OR
		     (fr.requester_id = u.id AND fr.addressee_id = $1)
		  )
	`, userID, strings.TrimSpace(friendPublicID))
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("%w: друг не найден", errNotFound)
	}
	return nil
}

func listCommunities(db *sql.DB) ([]community, error) {
	rows, err := db.Query(`
		SELECT c.id, c.name, c.category, c.description, c.tags, c.privacy_level, u.public_id, c.created_at
		FROM communities c
		JOIN users u ON u.id = c.creator_id
		ORDER BY c.created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []community
	for rows.Next() {
		var item community
		if scanErr := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Category,
			&item.Description,
			&item.Tags,
			&item.Privacy,
			&item.CreatorID,
			&item.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func createCommunity(db *sql.DB, creatorID int64, req createCommunityRequest) (community, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return community{}, fmt.Errorf("%w: название сообщества обязательно", errValidation)
	}
	if len(name) > 120 {
		return community{}, fmt.Errorf("%w: название слишком длинное", errValidation)
	}
	category := strings.TrimSpace(req.Category)
	if category == "" {
		category = "Другое"
	}

	privacy := normalizePrivacyLevel(req.PrivacyLevel)
	if privacy == "" {
		return community{}, fmt.Errorf("%w: privacy_level должен быть open или closed", errValidation)
	}

	description := strings.TrimSpace(req.Description)
	if len(description) > 1200 {
		return community{}, fmt.Errorf("%w: описание слишком длинное", errValidation)
	}

	tags := make([]string, 0, len(req.Tags))
	seen := make(map[string]struct{}, len(req.Tags))
	for _, t := range req.Tags {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}
		if len(tag) > 40 {
			return community{}, fmt.Errorf("%w: теги должны быть не длиннее 40 символов", errValidation)
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, tag)
		if len(tags) == 10 {
			break
		}
	}

	var created community
	err := db.QueryRow(`
		INSERT INTO communities(name, category, description, tags, privacy_level, creator_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, category, description, tags, privacy_level, created_at
	`, name, category, description, tags, privacy, creatorID).
		Scan(&created.ID, &created.Name, &created.Category, &created.Description, &created.Tags, &created.Privacy, &created.CreatedAt)
	if err != nil {
		return community{}, err
	}
	return created, nil
}

func normalizePrivacyLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return "open"
	case "closed":
		return "closed"
	default:
		return ""
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func newPublicUserID() (string, error) {
	buf := make([]byte, 5)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "u" + hex.EncodeToString(buf), nil
}

func updateUserProfile(db *sql.DB, userID int64, req profileUpdateRequest) (user, error) {
	firstName := strings.TrimSpace(req.FirstName)
	lastName := strings.TrimSpace(req.LastName)
	fullName := strings.TrimSpace(req.FullName)
	if firstName == "" {
		return user{}, fmt.Errorf("%w: имя обязательно", errValidation)
	}
	if lastName == "" {
		return user{}, fmt.Errorf("%w: фамилия обязательна", errValidation)
	}
	if fullName == "" {
		fullName = strings.TrimSpace(firstName + " " + lastName)
	}

	var updated user
	err := db.QueryRow(`
		UPDATE users
		SET first_name = $2,
		    last_name = $3,
		    full_name = $4,
		    position = NULLIF($5, ''),
		    company_name = NULLIF($6, ''),
		    bio = NULLIF($7, ''),
		    phone = NULLIF($8, ''),
		    location = NULLIF($9, ''),
		    city = NULLIF($10, ''),
		    avatar_url = NULLIF($11, '')
		WHERE id = $1
		RETURNING id, public_id, first_name, last_name, full_name, email,
			COALESCE(position, ''), COALESCE(company_name, ''), COALESCE(bio, ''),
			COALESCE(phone, ''), COALESCE(location, ''), COALESCE(city, ''), COALESCE(avatar_url, '')
	`, userID,
		firstName,
		lastName,
		fullName,
		strings.TrimSpace(req.Position),
		strings.TrimSpace(req.CompanyName),
		strings.TrimSpace(req.Bio),
		strings.TrimSpace(req.Phone),
		strings.TrimSpace(req.Location),
		strings.TrimSpace(req.City),
		strings.TrimSpace(req.AvatarURL),
	).Scan(&updated.ID, &updated.PublicID, &updated.FirstName, &updated.LastName, &updated.FullName, &updated.Email, &updated.Position, &updated.CompanyName, &updated.Bio, &updated.Phone, &updated.Location, &updated.City, &updated.AvatarURL)
	if err != nil {
		return user{}, err
	}
	return updated, nil
}
