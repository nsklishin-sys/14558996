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
	"net"
	"net/http"
	"net/mail"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	_ "github.com/jackc/pgx/v5/stdlib"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
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
	IsMember    bool      `json:"is_member"`
	Members     int       `json:"members_count"`
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

type post struct {
	ID             int64     `json:"id"`
	PublicID       string    `json:"public_id"`
	Type           string    `json:"type"`
	Title          string    `json:"title"`
	Content        string    `json:"content"`
	Text           string    `json:"text"`
	CoverURL       string    `json:"cover_url,omitempty"`
	Tags           []string  `json:"tags"`
	PrivacyLevel   string    `json:"privacy_level"`
	LikesCount     int       `json:"likes_count"`
	CommentsCount  int       `json:"comments_count"`
	ViewsCount     int       `json:"views_count"`
	IsLiked        bool      `json:"is_liked"`
	CreatedAt      time.Time `json:"created_at"`
	AuthorID       int64     `json:"-"`
	AuthorPublicID string    `json:"author_public_id"`
	AuthorName     string    `json:"author"`
	AuthorRole     string    `json:"author_role"`
	AuthorAvatar   string    `json:"author_avatar,omitempty"`
}

type createPostRequest struct {
	Title        string   `json:"title"`
	Content      string   `json:"content"`
	Type         string   `json:"type"`
	Tags         []string `json:"tags"`
	CoverURL     string   `json:"cover_url"`
	PrivacyLevel string   `json:"privacy_level"`
}

type postComment struct {
	ID             int64     `json:"id"`
	AuthorPublicID string    `json:"author_public_id"`
	AuthorName     string    `json:"author_name"`
	Content        string    `json:"content"`
	ParentID       *int64    `json:"parent_id,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type createCommentRequest struct {
	Content  string `json:"content"`
	ParentID *int64 `json:"parent_id,omitempty"`
}

type sessionStore struct {
	// TODO: keep in-memory sessions for now; move to durable/shared storage in a dedicated task.
	mu         sync.RWMutex
	tokenToUID map[string]int64
}

type ipRateLimiter struct {
	mu      sync.Mutex
	limit   rate.Limit
	burst   int
	entries map[string]*rateEntry
}

type rateEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func newIPRateLimiter(limit int, interval time.Duration) *ipRateLimiter {
	return &ipRateLimiter{
		limit:   rate.Every(interval / time.Duration(limit)),
		burst:   limit,
		entries: make(map[string]*rateEntry),
	}
}

func (l *ipRateLimiter) Allow(ip string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	entry, ok := l.entries[ip]
	if !ok {
		entry = &rateEntry{
			limiter: rate.NewLimiter(l.limit, l.burst),
		}
		l.entries[ip] = entry
	}
	entry.lastSeen = now

	if len(l.entries) > 10000 {
		next := make(map[string]*rateEntry, len(l.entries))
		for key, item := range l.entries {
			if now.Sub(item.lastSeen) <= 10*time.Minute {
				next[key] = item
			}
		}
		l.entries = next
	}

	return entry.limiter.Allow()
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

func main() {
	db, err := initDBFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	sessions := newSessionStore()
	authRateLimiter := newIPRateLimiter(10, time.Minute)
	postRateLimiter := newIPRateLimiter(10, time.Minute)
	commentRateLimiter := newIPRateLimiter(30, time.Minute)

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
		if !authRateLimiter.Allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
			return
		}

		var req registerRequest
		if err := decodeJSON(w, r, &req); err != nil {
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

		sessions.put(token, createdUser.ID)
		writeJSON(w, http.StatusCreated, authResponse{Token: token, User: createdUser})
	})

	mux.HandleFunc("/api/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		if !authRateLimiter.Allow(clientIP(r)) {
			writeError(w, http.StatusTooManyRequests, "Слишком много попыток, попробуйте позже")
			return
		}

		var req loginRequest
		if err := decodeJSON(w, r, &req); err != nil {
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

		sessions.put(token, authUser.ID)
		writeJSON(w, http.StatusOK, authResponse{Token: token, User: authUser})
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
			if err := decodeJSON(w, r, &req); err != nil {
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
		if err := decodeJSON(w, r, &req); err != nil {
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
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			communities, err := listCommunities(db, authUserID, hasAuth)
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
			if err := decodeJSON(w, r, &req); err != nil {
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

	mux.HandleFunc("/api/communities/", func(w http.ResponseWriter, r *http.Request) {
		idPart := strings.TrimPrefix(r.URL.Path, "/api/communities/")
		if idPart == "" {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}

		if strings.HasSuffix(idPart, "/join") || strings.HasSuffix(idPart, "/leave") {
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}

			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}

			action := "join"
			if strings.HasSuffix(idPart, "/leave") {
				action = "leave"
				idPart = strings.TrimSuffix(idPart, "/leave")
			} else {
				idPart = strings.TrimSuffix(idPart, "/join")
			}
			if idPart == "" || strings.Contains(idPart, "/") {
				writeError(w, http.StatusBadRequest, "Некорректный id сообщества")
				return
			}

			communityID, err := strconv.ParseInt(idPart, 10, 64)
			if err != nil || communityID <= 0 {
				writeError(w, http.StatusBadRequest, "Некорректный id сообщества")
				return
			}

			var actionErr error
			if action == "leave" {
				actionErr = leaveCommunity(db, communityID, userID)
			} else {
				actionErr = joinCommunity(db, communityID, userID)
			}
			if actionErr != nil {
				switch {
				case errors.Is(actionErr, errNotFound):
					writeError(w, http.StatusNotFound, "Сообщество не найдено")
				default:
					log.Printf("%s community error: %v", action, actionErr)
					writeError(w, http.StatusInternalServerError, "Ошибка сервера")
				}
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}

		if r.Method != http.MethodGet || strings.Contains(idPart, "/") {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}

		communityID, err := strconv.ParseInt(idPart, 10, 64)
		if err != nil || communityID <= 0 {
			writeError(w, http.StatusBadRequest, "Некорректный id сообщества")
			return
		}

		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		item, err := getCommunityByID(db, communityID, authUserID, hasAuth)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, "Сообщество не найдено")
				return
			}
			log.Printf("get community by id error: %v", err)
			writeError(w, http.StatusInternalServerError, "Ошибка сервера")
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{"community": item})
	})

	mux.HandleFunc("/api/posts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !postRateLimiter.Allow(fmt.Sprintf("post:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много публикаций, попробуйте позже")
				return
			}
			var req createPostRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}
			created, err := createPost(db, userID, req)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"post": created})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/posts/", func(w http.ResponseWriter, r *http.Request) {
		publicID, ok := extractPathParam(r.URL.Path, "/api/posts/")
		if !ok {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		if strings.HasSuffix(publicID, "/like") {
			postPublicID := strings.TrimSuffix(publicID, "/like")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, hasAuth := authenticatedUserID(w, r, sessions)
			if !hasAuth {
				return
			}
			isLiked, likesCount, err := toggleLike(db, postPublicID, userID)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"is_liked": isLiked, "likes_count": likesCount})
			return
		}
		if strings.HasSuffix(publicID, "/comments") {
			postPublicID := strings.TrimSuffix(publicID, "/comments")
			if !isValidPostPublicID(postPublicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id поста")
				return
			}
			switch r.Method {
			case http.MethodGet:
				limit := parseLimit(r.URL.Query().Get("limit"), 50, 200)
				comments, err := listComments(db, postPublicID, limit)
				if err != nil {
					handlePostActionError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"comments": comments})
			case http.MethodPost:
				userID, hasAuth := authenticatedUserID(w, r, sessions)
				if !hasAuth {
					return
				}
				if !commentRateLimiter.Allow(fmt.Sprintf("comment:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много комментариев, попробуйте позже")
					return
				}
				var req createCommentRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				comment, err := createComment(db, postPublicID, userID, req)
				if err != nil {
					handlePostActionError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]any{"comment": comment})
			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}
		if !isValidPostPublicID(publicID) {
			writeError(w, http.StatusBadRequest, "Некорректный id поста")
			return
		}
		switch r.Method {
		case http.MethodGet:
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			item, err := getPostByID(db, publicID, authUserID, hasAuth, true)
			if err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"post": item})
		case http.MethodDelete:
			userID, hasAuth := authenticatedUserID(w, r, sessions)
			if !hasAuth {
				return
			}
			if err := softDeletePost(db, publicID, userID); err != nil {
				handlePostActionError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/feed", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
		beforeID := parseIDOrZero(r.URL.Query().Get("before_id"))
		postType := strings.TrimSpace(r.URL.Query().Get("type"))
		posts, nextCursor, err := listFeed(db, authUserID, hasAuth, limit, beforeID, postType)
		if err != nil {
			handlePostActionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"posts": posts, "next_cursor": nextCursor})
	})

	mux.HandleFunc("/api/users/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/posts") {
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
			return
		}
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			return
		}
		publicID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/posts")
		publicID = strings.TrimSpace(strings.TrimSuffix(publicID, "/"))
		if !isValidUserPublicID(publicID) {
			writeError(w, http.StatusBadRequest, "Некорректный id пользователя")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
		beforeID := parseIDOrZero(r.URL.Query().Get("before_id"))
		posts, nextCursor, err := listUserPosts(db, publicID, authUserID, hasAuth, limit, beforeID)
		if err != nil {
			handlePostActionError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"posts": posts, "next_cursor": nextCursor})
	})

	mux.HandleFunc("/api/news/", func(w http.ResponseWriter, r *http.Request) {
		suffix := strings.TrimPrefix(r.URL.Path, "/api/news/")
		if suffix == "" || strings.Contains(suffix, "//") {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		r.URL.Path = "/api/posts/" + suffix
		mux.ServeHTTP(w, r)
	})

	mux.Handle("/", staticSecurity(http.FileServer(http.Dir("./web"))))

	addr := ":8080"
	handler := accessLog(securityHeaders(mux))
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Printf("Server started at http://localhost%s", addr)
	if err := srv.ListenAndServe(); err != nil {
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
	if directURL := strings.TrimSpace(os.Getenv("DATABASE_URL")); directURL != "" {
		return directURL, nil
	}

	return "", fmt.Errorf("DATABASE_URL is required")
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

-- Миграция существующих users без public_id должна выполняться вручную.

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

DO $$
BEGIN
    ALTER TABLE friend_requests DROP CONSTRAINT IF EXISTS friend_requests_status_check;
    ALTER TABLE friend_requests ADD CONSTRAINT friend_requests_status_check
      CHECK (status IN ('pending', 'accepted', 'rejected', 'canceled', 'unfriended'));
END$$;

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

CREATE TABLE IF NOT EXISTS community_members (
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (community_id, user_id)
);

CREATE INDEX IF NOT EXISTS community_members_user_idx
ON community_members(user_id);

CREATE TABLE IF NOT EXISTS posts (
    id BIGSERIAL PRIMARY KEY,
    public_id TEXT UNIQUE NOT NULL,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL DEFAULT 'news',
    title TEXT NOT NULL,
    content TEXT NOT NULL,
    cover_url TEXT,
    tags TEXT[] NOT NULL DEFAULT '{}',
    privacy_level TEXT NOT NULL DEFAULT 'public',
    likes_count INTEGER NOT NULL DEFAULT 0,
    comments_count INTEGER NOT NULL DEFAULT 0,
    views_count INTEGER NOT NULL DEFAULT 0,
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (type IN ('news', 'project', 'note', 'case')),
    CHECK (privacy_level IN ('public', 'friends', 'private')),
    CHECK (char_length(title) BETWEEN 1 AND 200),
    CHECK (char_length(content) BETWEEN 1 AND 20000)
);

CREATE INDEX IF NOT EXISTS posts_author_created_idx
    ON posts(author_id, created_at DESC) WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS posts_feed_idx
    ON posts(created_at DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE INDEX IF NOT EXISTS posts_type_created_idx
    ON posts(type, created_at DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE TABLE IF NOT EXISTS post_likes (
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS post_likes_user_idx ON post_likes(user_id);

CREATE TABLE IF NOT EXISTS post_comments (
    id BIGSERIAL PRIMARY KEY,
    post_id BIGINT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id BIGINT REFERENCES post_comments(id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (char_length(content) BETWEEN 1 AND 5000),
    is_deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS post_comments_post_idx
    ON post_comments(post_id, created_at DESC) WHERE is_deleted = FALSE;
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
	if !isValidEmail(email) {
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
	if !isValidEmail(email) {
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

func optionalAuthenticatedUserID(r *http.Request, sessions *sessionStore) (int64, bool) {
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return 0, false
	}
	token := strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
	if token == "" {
		return 0, false
	}
	userID, ok := sessions.getUserID(token)
	if !ok {
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
		ORDER BY LOWER(u.full_name)
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
		ORDER BY LOWER(u.full_name)
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
		ORDER BY LOWER(u.full_name)
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
	trimmed := strings.TrimSpace(query)
	var (
		rows *sql.Rows
		err  error
	)
	if trimmed == "" {
		rows, err = db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM users u
		WHERE u.id <> $1
		  AND NOT EXISTS (
		      SELECT 1
		      FROM friend_requests fr
		      WHERE (
		           (fr.requester_id = $1 AND fr.addressee_id = u.id)
		        OR (fr.requester_id = u.id AND fr.addressee_id = $1)
		      )
		      AND fr.status IN ('pending', 'accepted')
		  )
		ORDER BY LOWER(u.full_name)
		LIMIT 50
	`, userID)
	} else {
		esc := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(strings.ToLower(trimmed))
		search := "%" + esc + "%"
		rows, err = db.Query(`
		SELECT u.public_id, u.full_name, u.email
		FROM users u
		WHERE u.id <> $1
		  AND (
		      LOWER(u.full_name) LIKE $2 ESCAPE '\'
		      OR LOWER(u.email) LIKE $2 ESCAPE '\'
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
		ORDER BY LOWER(u.full_name)
		LIMIT 50
	`, userID, search)
	}
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
	if isValidEmail(vLower) {
		var publicID string
		err := db.QueryRow(`
			SELECT public_id
			FROM users
			WHERE id <> $1
			  AND LOWER(email) = $2
			LIMIT 1
		`, userID, vLower).Scan(&publicID)
		return publicID, err
	}
	rows, err := db.Query(`
		SELECT public_id
		FROM users
		WHERE id <> $1
		  AND LOWER(full_name) = $2
		ORDER BY id
		LIMIT 2
	`, userID, vLower)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if scanErr := rows.Scan(&id); scanErr != nil {
			return "", scanErr
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	if len(ids) == 0 {
		return "", sql.ErrNoRows
	}
	if len(ids) > 1 {
		return "", fmt.Errorf("%w: Найдено несколько пользователей с таким именем, используйте email", errValidation)
	}
	return ids[0], nil
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
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			log.Printf("tx rollback error: %v", rerr)
		}
	}()

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
		case "rejected", "canceled", "unfriended":
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
		SET status = 'unfriended', updated_at = NOW()
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

func listCommunities(db *sql.DB, authUserID int64, hasAuth bool) ([]community, error) {
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}

	rows, err := db.Query(`
		SELECT c.id,
		       c.name,
		       c.category,
		       c.description,
		       COALESCE(array_to_json(c.tags), '[]'::json),
		       c.privacy_level,
		       u.public_id,
		       (
		           $1::bigint IS NOT NULL
		           AND EXISTS (
		               SELECT 1
		               FROM community_members cm_user
		               WHERE cm_user.community_id = c.id
		                 AND cm_user.user_id = $1::bigint
		           )
		       ) AS is_member,
		       COALESCE(m.members_count, 0),
		       c.created_at
		FROM communities c
		JOIN users u ON u.id = c.creator_id
		LEFT JOIN (
			SELECT community_id, COUNT(*)::int AS members_count
			FROM community_members
			GROUP BY community_id
		) m ON m.community_id = c.id
		ORDER BY c.created_at DESC
	`, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []community
	for rows.Next() {
		var item community
		var rawTags []byte
		if scanErr := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Category,
			&item.Description,
			&rawTags,
			&item.Privacy,
			&item.CreatorID,
			&item.IsMember,
			&item.Members,
			&item.CreatedAt,
		); scanErr != nil {
			return nil, scanErr
		}
		if err := json.Unmarshal(rawTags, &item.Tags); err != nil {
			return nil, fmt.Errorf("decode community tags: %w", err)
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func getCommunityByID(db *sql.DB, communityID, authUserID int64, hasAuth bool) (community, error) {
	var item community
	var tagsJSON []byte
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	err := db.QueryRow(`
		SELECT c.id,
		       c.name,
		       c.category,
		       COALESCE(c.description, ''),
		       COALESCE(array_to_json(c.tags), '[]'::json),
		       c.privacy_level,
		       u.public_id,
		       (
		           $2::bigint IS NOT NULL
		           AND EXISTS (
		               SELECT 1
		               FROM community_members cm_user
		               WHERE cm_user.community_id = c.id
		                 AND cm_user.user_id = $2::bigint
		           )
		       ) AS is_member,
		       (SELECT COUNT(*)::int FROM community_members cm WHERE cm.community_id = c.id) AS members_count,
		       c.created_at
		FROM communities c
		LEFT JOIN users u ON u.id = c.creator_id
		WHERE c.id = $1
	`, communityID, currentUserID).Scan(
		&item.ID,
		&item.Name,
		&item.Category,
		&item.Description,
		&tagsJSON,
		&item.Privacy,
		&item.CreatorID,
		&item.IsMember,
		&item.Members,
		&item.CreatedAt,
	)
	if err != nil {
		return community{}, err
	}
	if err := json.Unmarshal(tagsJSON, &item.Tags); err != nil {
		return community{}, fmt.Errorf("decode community tags: %w", err)
	}
	return item, nil
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

	tx, err := db.Begin()
	if err != nil {
		return community{}, err
	}
	defer func() {
		if rerr := tx.Rollback(); rerr != nil && !errors.Is(rerr, sql.ErrTxDone) {
			log.Printf("tx rollback error: %v", rerr)
		}
	}()

	var created community
	var rawTags []byte
	var pgTags pgtype.FlatArray[string] = tags
	err = tx.QueryRow(`
		INSERT INTO communities(name, category, description, tags, privacy_level, creator_id)
		VALUES ($1, $2, $3, $4::text[], $5, $6)
		RETURNING id, name, category, description, COALESCE(array_to_json(tags), '[]'::json), privacy_level, created_at,
			(SELECT public_id FROM users WHERE id = creator_id)
	`, name, category, description, pgTags, privacy, creatorID).
		Scan(&created.ID, &created.Name, &created.Category, &created.Description, &rawTags, &created.Privacy, &created.CreatedAt, &created.CreatorID)
	if err != nil {
		return community{}, err
	}
	if err := json.Unmarshal(rawTags, &created.Tags); err != nil {
		return community{}, fmt.Errorf("decode created community tags: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO community_members(community_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (community_id, user_id) DO NOTHING
	`, created.ID, creatorID); err != nil {
		return community{}, err
	}
	if err := tx.Commit(); err != nil {
		return community{}, err
	}
	created.Members = 1
	return created, nil
}

func joinCommunity(db *sql.DB, communityID, userID int64) error {
	res, err := db.Exec(`
		INSERT INTO community_members(community_id, user_id)
		SELECT $1, $2
		WHERE EXISTS (SELECT 1 FROM communities WHERE id = $1)
		ON CONFLICT (community_id, user_id) DO NOTHING
	`, communityID, userID)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		var exists bool
		if scanErr := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM communities WHERE id = $1)`, communityID).Scan(&exists); scanErr != nil {
			return scanErr
		}
		if !exists {
			return fmt.Errorf("%w: сообщество не найдено", errNotFound)
		}
	}
	return nil
}

func leaveCommunity(db *sql.DB, communityID, userID int64) error {
	result, err := db.Exec(`
		DELETE FROM community_members
		WHERE community_id = $1 AND user_id = $2
	`, communityID, userID)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		var exists bool
		if scanErr := db.QueryRow(`SELECT EXISTS (SELECT 1 FROM communities WHERE id = $1)`, communityID).Scan(&exists); scanErr != nil {
			return scanErr
		}
		if !exists {
			return errNotFound
		}
	}
	return nil
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

func newPublicPostID() (string, error) {
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return "p" + hex.EncodeToString(buf), nil
}

func createPost(db *sql.DB, authorID int64, req createPostRequest) (post, error) {
	title := strings.TrimSpace(req.Title)
	if count := utf8.RuneCountInString(title); count < 3 || count > 200 {
		return post{}, fmt.Errorf("%w: заголовок должен быть от 3 до 200 символов", errValidation)
	}
	content := strings.TrimSpace(req.Content)
	if count := utf8.RuneCountInString(content); count < 10 || count > 20000 {
		return post{}, fmt.Errorf("%w: текст должен быть от 10 до 20000 символов", errValidation)
	}
	postType := strings.ToLower(strings.TrimSpace(req.Type))
	if postType == "" {
		postType = "news"
	}
	if postType != "news" && postType != "project" && postType != "note" && postType != "case" {
		return post{}, fmt.Errorf("%w: type должен быть news/project/note/case", errValidation)
	}
	privacy := strings.ToLower(strings.TrimSpace(req.PrivacyLevel))
	if privacy == "" {
		privacy = "public"
	}
	if privacy != "public" && privacy != "friends" && privacy != "private" {
		return post{}, fmt.Errorf("%w: privacy_level должен быть public/friends/private", errValidation)
	}
	coverURL := strings.TrimSpace(req.CoverURL)
	if coverURL != "" {
		if utf8.RuneCountInString(coverURL) > 500 || (!strings.HasPrefix(coverURL, "http://") && !strings.HasPrefix(coverURL, "https://")) {
			return post{}, fmt.Errorf("%w: cover_url должен начинаться с http:// или https:// и быть не длиннее 500 символов", errValidation)
		}
	}
	tags := make([]string, 0, len(req.Tags))
	seen := make(map[string]struct{}, len(req.Tags))
	for _, raw := range req.Tags {
		tag := strings.TrimSpace(raw)
		if tag == "" {
			continue
		}
		if utf8.RuneCountInString(tag) > 40 {
			return post{}, fmt.Errorf("%w: тег должен быть не длиннее 40 символов", errValidation)
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tags = append(tags, tag)
		if len(tags) > 10 {
			return post{}, fmt.Errorf("%w: максимум 10 тегов", errValidation)
		}
	}

	var pgTags pgtype.FlatArray[string] = tags
	for attempts := 0; attempts < 5; attempts++ {
		publicID, err := newPublicPostID()
		if err != nil {
			return post{}, err
		}
		created, err := scanPost(db.QueryRow(`
			INSERT INTO posts (public_id, author_id, type, title, content, cover_url, tags, privacy_level)
			VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7::text[], $8)
			RETURNING id, public_id, type, title, content, COALESCE(cover_url, ''), COALESCE(array_to_json(tags), '[]'::json),
					  privacy_level, likes_count, comments_count, views_count, created_at, author_id
		`, publicID, authorID, postType, title, content, coverURL, pgTags, privacy))
		if err == nil {
			return hydratePostAuthor(db, created)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			continue
		}
		return post{}, err
	}
	return post{}, fmt.Errorf("failed to allocate post public_id")
}

func getPostByID(db *sql.DB, publicID string, authUserID int64, hasAuth, incrementViews bool) (post, error) {
	tx, err := db.Begin()
	if err != nil {
		return post{}, err
	}
	defer tx.Rollback()
	if incrementViews {
		if _, err := tx.Exec(`UPDATE posts SET views_count = views_count + 1, updated_at = NOW() WHERE public_id = $1 AND is_deleted = FALSE`, publicID); err != nil {
			return post{}, err
		}
	}
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	var item post
	var tagsJSON []byte
	var coverURL string
	err = tx.QueryRow(`
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count,
		       p.comments_count, p.views_count, p.created_at, p.author_id,
		       COALESCE(pl.user_id IS NOT NULL, FALSE)
		FROM posts p
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $2::bigint
		WHERE p.public_id = $1 AND p.is_deleted = FALSE
		LIMIT 1
	`, publicID, currentUser).Scan(
		&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &coverURL, &tagsJSON,
		&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.CreatedAt, &item.AuthorID, &item.IsLiked,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return post{}, errNotFound
		}
		return post{}, err
	}
	item.CoverURL = coverURL
	item.Text = item.Content
	_ = json.Unmarshal(tagsJSON, &item.Tags)
	if item.PrivacyLevel == "private" && (!hasAuth || authUserID != item.AuthorID) {
		return post{}, errNotFound
	}
	if item.PrivacyLevel == "friends" && (!hasAuth || !canViewFriendsPost(tx, authUserID, item.AuthorID)) {
		return post{}, errNotFound
	}
	if err := tx.Commit(); err != nil {
		return post{}, err
	}
	return hydratePostAuthor(db, item)
}

func listFeed(db *sql.DB, authUserID int64, hasAuth bool, limit int, beforeID int64, postType string) ([]post, *int64, error) {
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	args := []any{currentUser, limit + 1}
	query := `
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       COALESCE(pl.user_id IS NOT NULL, FALSE)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		WHERE p.is_deleted = FALSE AND p.privacy_level = 'public'`
	if postType != "" {
		postType = strings.ToLower(strings.TrimSpace(postType))
		query += fmt.Sprintf(" AND p.type = $%d", len(args)+1)
		args = append(args, postType)
	}
	if beforeID > 0 {
		query += fmt.Sprintf(" AND p.id < $%d", len(args)+1)
		args = append(args, beforeID)
	}
	query += " ORDER BY p.created_at DESC, p.id DESC LIMIT $2"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []post
	for rows.Next() {
		var item post
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &item.CoverURL, &tagsJSON,
			&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.CreatedAt, &item.AuthorID,
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var nextCursor *int64
	if len(items) > limit {
		last := items[limit-1].ID
		nextCursor = &last
		items = items[:limit]
	}
	return items, nextCursor, nil
}

func listUserPosts(db *sql.DB, userPublicID string, authUserID int64, hasAuth bool, limit int, beforeID int64) ([]post, *int64, error) {
	var targetID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id = $1`, userPublicID).Scan(&targetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, errNotFound
		}
		return nil, nil, err
	}
	isOwner := hasAuth && authUserID == targetID
	isFriend := hasAuth && areFriends(db, authUserID, targetID)
	currentUser := sql.NullInt64{}
	if hasAuth {
		currentUser = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	query := `
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       COALESCE(pl.user_id IS NOT NULL, FALSE)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		WHERE p.author_id = $2 AND p.is_deleted = FALSE`
	args := []any{currentUser, targetID, limit + 1}
	if !isOwner {
		if isFriend {
			query += " AND p.privacy_level IN ('public', 'friends')"
		} else {
			query += " AND p.privacy_level = 'public'"
		}
	}
	if beforeID > 0 {
		query += fmt.Sprintf(" AND p.id < $%d", len(args)+1)
		args = append(args, beforeID)
	}
	query += " ORDER BY p.created_at DESC, p.id DESC LIMIT $3"
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var items []post
	for rows.Next() {
		var item post
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &item.CoverURL, &tagsJSON,
			&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.CreatedAt, &item.AuthorID,
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}
	var nextCursor *int64
	if len(items) > limit {
		last := items[limit-1].ID
		nextCursor = &last
		items = items[:limit]
	}
	return items, nextCursor, nil
}

func softDeletePost(db *sql.DB, publicID string, userID int64) error {
	res, err := db.Exec(`UPDATE posts SET is_deleted = TRUE, updated_at = NOW() WHERE public_id = $1 AND author_id = $2 AND is_deleted = FALSE`, publicID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func toggleLike(db *sql.DB, publicID string, userID int64) (bool, int, error) {
	tx, err := db.Begin()
	if err != nil {
		return false, 0, err
	}
	defer tx.Rollback()
	var postID int64
	if err := tx.QueryRow(`SELECT id FROM posts WHERE public_id = $1 AND is_deleted = FALSE`, publicID).Scan(&postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, 0, errNotFound
		}
		return false, 0, err
	}
	var exists bool
	if err := tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM post_likes WHERE post_id = $1 AND user_id = $2)`, postID, userID).Scan(&exists); err != nil {
		return false, 0, err
	}
	isLiked := false
	if exists {
		if _, err := tx.Exec(`DELETE FROM post_likes WHERE post_id = $1 AND user_id = $2`, postID, userID); err != nil {
			return false, 0, err
		}
		if _, err := tx.Exec(`UPDATE posts SET likes_count = likes_count - 1, updated_at = NOW() WHERE id = $1 AND likes_count > 0`, postID); err != nil {
			return false, 0, err
		}
	} else {
		if _, err := tx.Exec(`INSERT INTO post_likes(post_id, user_id) VALUES ($1, $2)`, postID, userID); err != nil {
			return false, 0, err
		}
		if _, err := tx.Exec(`UPDATE posts SET likes_count = likes_count + 1, updated_at = NOW() WHERE id = $1`, postID); err != nil {
			return false, 0, err
		}
		isLiked = true
	}
	var likesCount int
	if err := tx.QueryRow(`SELECT likes_count FROM posts WHERE id = $1`, postID).Scan(&likesCount); err != nil {
		return false, 0, err
	}
	if err := tx.Commit(); err != nil {
		return false, 0, err
	}
	return isLiked, likesCount, nil
}

func listComments(db *sql.DB, postPublicID string, limit int) ([]postComment, error) {
	rows, err := db.Query(`
		SELECT pc.id, u.public_id, u.full_name, pc.content, pc.parent_id, pc.created_at
		FROM post_comments pc
		JOIN posts p ON p.id = pc.post_id
		JOIN users u ON u.id = pc.author_id
		WHERE p.public_id = $1 AND p.is_deleted = FALSE AND pc.is_deleted = FALSE
		ORDER BY pc.created_at ASC, pc.id ASC
		LIMIT $2
	`, postPublicID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []postComment
	for rows.Next() {
		var c postComment
		if err := rows.Scan(&c.ID, &c.AuthorPublicID, &c.AuthorName, &c.Content, &c.ParentID, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func createComment(db *sql.DB, postPublicID string, authorID int64, req createCommentRequest) (postComment, error) {
	content := strings.TrimSpace(req.Content)
	if count := utf8.RuneCountInString(content); count < 1 || count > 5000 {
		return postComment{}, fmt.Errorf("%w: комментарий должен быть от 1 до 5000 символов", errValidation)
	}
	tx, err := db.Begin()
	if err != nil {
		return postComment{}, err
	}
	defer tx.Rollback()
	var postID int64
	if err := tx.QueryRow(`SELECT id FROM posts WHERE public_id = $1 AND is_deleted = FALSE`, postPublicID).Scan(&postID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return postComment{}, errNotFound
		}
		return postComment{}, err
	}
	if req.ParentID != nil {
		var parentPostID int64
		if err := tx.QueryRow(`SELECT post_id FROM post_comments WHERE id = $1 AND is_deleted = FALSE`, *req.ParentID).Scan(&parentPostID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return postComment{}, fmt.Errorf("%w: родительский комментарий не найден", errValidation)
			}
			return postComment{}, err
		}
		if parentPostID != postID {
			return postComment{}, fmt.Errorf("%w: parent_id должен принадлежать тому же посту", errValidation)
		}
	}
	var created postComment
	if err := tx.QueryRow(`
		INSERT INTO post_comments(post_id, author_id, parent_id, content)
		VALUES ($1, $2, $3, $4)
		RETURNING id, parent_id, content, created_at
	`, postID, authorID, req.ParentID, content).Scan(&created.ID, &created.ParentID, &created.Content, &created.CreatedAt); err != nil {
		return postComment{}, err
	}
	if _, err := tx.Exec(`UPDATE posts SET comments_count = comments_count + 1, updated_at = NOW() WHERE id = $1`, postID); err != nil {
		return postComment{}, err
	}
	if err := tx.QueryRow(`SELECT public_id, full_name FROM users WHERE id = $1`, authorID).Scan(&created.AuthorPublicID, &created.AuthorName); err != nil {
		return postComment{}, err
	}
	if err := tx.Commit(); err != nil {
		return postComment{}, err
	}
	return created, nil
}

func scanPost(row *sql.Row) (post, error) {
	var item post
	var tagsJSON []byte
	var coverURL string
	if err := row.Scan(&item.ID, &item.PublicID, &item.Type, &item.Title, &item.Content, &coverURL, &tagsJSON,
		&item.PrivacyLevel, &item.LikesCount, &item.CommentsCount, &item.ViewsCount, &item.CreatedAt, &item.AuthorID); err != nil {
		return post{}, err
	}
	item.CoverURL = coverURL
	item.Text = item.Content
	_ = json.Unmarshal(tagsJSON, &item.Tags)
	return item, nil
}

func hydratePostAuthor(db *sql.DB, item post) (post, error) {
	if err := db.QueryRow(`
		SELECT public_id, full_name, COALESCE(NULLIF(position, ''), company_name, ''), COALESCE(avatar_url, '')
		FROM users WHERE id = $1
	`, item.AuthorID).Scan(&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar); err != nil {
		return post{}, err
	}
	return item, nil
}

func canViewFriendsPost(tx *sql.Tx, authUserID, authorID int64) bool {
	if authUserID == authorID {
		return true
	}
	var ok bool
	_ = tx.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM friend_requests
			WHERE status = 'accepted'
			  AND ((requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1))
		)
	`, authUserID, authorID).Scan(&ok)
	return ok
}

func areFriends(db *sql.DB, a, b int64) bool {
	var ok bool
	_ = db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM friend_requests
			WHERE status = 'accepted'
			  AND ((requester_id = $1 AND addressee_id = $2) OR (requester_id = $2 AND addressee_id = $1))
		)
	`, a, b).Scan(&ok)
	return ok
}

func parseLimit(raw string, def, max int) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v <= 0 {
		return def
	}
	if v > max {
		return max
	}
	return v
}

func parseIDOrZero(raw string) int64 {
	v, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func extractPathParam(pathValue, prefix string) (string, bool) {
	if !strings.HasPrefix(pathValue, prefix) {
		return "", false
	}
	v := strings.Trim(strings.TrimPrefix(pathValue, prefix), "/")
	if v == "" {
		return "", false
	}
	return v, true
}

func isValidPostPublicID(s string) bool {
	if len(s) != 13 || !strings.HasPrefix(s, "p") {
		return false
	}
	for _, ch := range s[1:] {
		if (ch < 'a' || ch > 'f') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

func isValidUserPublicID(s string) bool {
	if len(s) != 11 || !strings.HasPrefix(s, "u") {
		return false
	}
	for _, ch := range s[1:] {
		if (ch < 'a' || ch > 'f') && (ch < '0' || ch > '9') {
			return false
		}
	}
	return true
}

func handlePostActionError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, "Не найдено")
	default:
		log.Printf("post action error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
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

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}

func isValidEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s
}

func securityHeaders(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.ServeHTTP(w, r)
	})
}

func staticSecurity(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := path.Clean("/" + r.URL.Path)
		for _, segment := range strings.Split(clean, "/") {
			if strings.HasPrefix(segment, ".") && segment != "." && segment != ".." {
				http.NotFound(w, r)
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}

func accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		log.Printf("%s %s status=%d duration=%s ip=%s", r.Method, r.URL.Path, rec.status, time.Since(start), clientIP(r))
	})
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip := strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
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
