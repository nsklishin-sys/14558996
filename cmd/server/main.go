package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/mail"
	"os"
	"path"
	"regexp"
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
	Region      string    `json:"region"`
	Website     string    `json:"website"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	AvatarURL   string    `json:"avatar_url"`
	CoverURL    string    `json:"cover_url"`
	Color       string    `json:"color"`
	Tags        []string  `json:"tags"`
	Privacy     string    `json:"privacy_level"`
	IsMember    bool      `json:"is_member"`
	MyRole      string    `json:"my_role,omitempty"`
	Members     int       `json:"members_count"`
	CreatorID   string    `json:"creator_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type createCommunityRequest struct {
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Region       string   `json:"region"`
	Website      string   `json:"website"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	Tags         []string `json:"tags"`
	PrivacyLevel string   `json:"privacy_level"`
	AvatarURL    string   `json:"avatar_url"`
	Color        string   `json:"color"`
}

type updateCommunityRequest struct {
	Name         *string   `json:"name,omitempty"`
	Category     *string   `json:"category,omitempty"`
	Description  *string   `json:"description,omitempty"`
	Region       *string   `json:"region,omitempty"`
	Website      *string   `json:"website,omitempty"`
	Email        *string   `json:"email,omitempty"`
	Phone        *string   `json:"phone,omitempty"`
	Tags         *[]string `json:"tags,omitempty"`
	PrivacyLevel *string   `json:"privacy_level,omitempty"`
	AvatarURL    *string   `json:"avatar_url,omitempty"`
	CoverURL     *string   `json:"cover_url,omitempty"`
	Color        *string   `json:"color,omitempty"`
}

type communityMemberDTO struct {
	UserPublicID string    `json:"user_public_id"`
	FullName     string    `json:"full_name"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Role         string    `json:"role"`
	JoinedAt     time.Time `json:"joined_at"`
}

type communityJoinRequestDTO struct {
	ID           int64     `json:"id"`
	UserPublicID string    `json:"user_public_id"`
	FullName     string    `json:"full_name"`
	Message      string    `json:"message"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

type changeRoleRequest struct {
	Role string `json:"role"`
}

type joinRequestBody struct {
	Message string `json:"message"`
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
	CommunityID    int64     `json:"community_id,omitempty"`
	CommunityName  string    `json:"community_name,omitempty"`
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

// htmlInject инжектируется во все HTML-ответы перед </head>.
// Содержит:
//  1. CSS для плавного фейда между страницами + View Transitions API.
//  2. JS для перехвата href="/" (замена на /home-auth.html для залогиненного,
//     на /home-guest.html для гостя) — это убирает белую страницу index.html
//     на клике "Главная".
//  3. Редирект залогиненного юзера с /home-guest.html и /index.html
//     сразу на /home-auth.html, без рендера гостевой.
const htmlInject = `<style>
@view-transition { navigation: auto; }
::view-transition-old(root),
::view-transition-new(root) {
  animation-duration: 180ms;
  animation-timing-function: cubic-bezier(0.25, 0.8, 0.25, 1);
}
html.pt-leaving body {
  opacity: 0;
  transition: opacity 140ms cubic-bezier(0.4, 0, 0.2, 1);
}
@keyframes pt-fade-in { from{opacity:0} to{opacity:1} }
html.pt-ready body {
  animation: pt-fade-in 160ms cubic-bezier(0.4, 0, 0.2, 1) both;
}
@media (prefers-reduced-motion: reduce) {
  ::view-transition-old(root),
  ::view-transition-new(root) { animation-duration: 0.01ms !important; }
  html.pt-leaving body, html.pt-ready body {
    animation: none !important;
    transition: none !important;
    opacity: 1 !important;
  }
}
</style>
<script>
(function(){
  'use strict';
  var html = document.documentElement;

  function hasToken(){
    try { var t = localStorage.getItem('token'); return !!(t && t.trim()); }
    catch(_) { return false; }
  }
  function prm(){
    return window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }
  function hasVT(){
    return typeof document.startViewTransition === 'function' ||
      (window.CSS && CSS.supports && CSS.supports('view-transition-name: a'));
  }

  // Мгновенный редирект залогиненного юзера с гостевых страниц
  var path = location.pathname;
  if ((path === '/home-guest.html' || path === '/index.html' || path === '/') && hasToken()) {
    location.replace('/home-auth.html');
    return;
  }

  if (!prm()) html.classList.add('pt-ready');

  function isInternalNav(a){
    if (!a || a.tagName !== 'A') return false;
    if (a.target && a.target !== '' && a.target !== '_self') return false;
    if (a.hasAttribute('download')) return false;
    var raw = a.getAttribute('href');
    if (!raw) return false;
    if (raw.charAt(0) === '#') return false;
    if (/^(mailto:|tel:|javascript:)/i.test(raw)) return false;
    try {
      var u = new URL(a.href, location.href);
      if (u.origin !== location.origin) return false;
      if (u.pathname === location.pathname && u.search === location.search && u.hash) return false;
      return true;
    } catch(_) { return false; }
  }

  // "/" → /home-auth.html для залогиненного, /home-guest.html для гостя
  // /index.html и /home-guest.html для залогиненного → /home-auth.html
  function resolveHome(u){
    var p = u.pathname;
    var isHome = p === '/' || p === '/index.html' || p === '/home-guest.html';
    if (!isHome) return null;
    var target = hasToken() ? '/home-auth.html' : '/home-guest.html';
    if (p === target) return null;
    return target + u.search + u.hash;
  }

  var useFade = !prm() && !hasVT();
  var FADE = 140;

  document.addEventListener('click', function(e){
    if (e.defaultPrevented) return;
    if (e.button !== 0) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
    var a = e.target && e.target.closest ? e.target.closest('a') : null;
    if (!isInternalNav(a)) return;

    var u;
    try { u = new URL(a.href, location.href); } catch(_) { return; }

    var home = resolveHome(u);
    var finalHref = home != null ? new URL(home, location.href).href : a.href;

    if (!useFade) {
      if (home != null) { e.preventDefault(); location.href = finalHref; }
      return;
    }

    e.preventDefault();
    html.classList.add('pt-leaving');
    var fb = setTimeout(function(){ html.classList.remove('pt-leaving'); }, 500);
    setTimeout(function(){ clearTimeout(fb); location.href = finalHref; }, FADE);
  }, true);

  window.addEventListener('pageshow', function(){ html.classList.remove('pt-leaving'); });
})();
</script>`

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
	communityCreateRateLimiter := newIPRateLimiter(5, time.Hour)
	communityPostRateLimiter := newIPRateLimiter(20, time.Hour)
	communityJoinRequestRateLimiter := newIPRateLimiter(10, time.Hour)

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
			communities, err := listCommunities(db, authUserID, hasAuth, communityListFilters{
				Privacy:  r.URL.Query().Get("privacy"),
				Category: r.URL.Query().Get("category"),
				Region:   r.URL.Query().Get("region"),
				Query:    r.URL.Query().Get("q"),
			})
			if err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"communities": communities})
		case http.MethodPost:
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if !communityCreateRateLimiter.Allow(fmt.Sprintf("community-create:%d", userID)) {
				writeError(w, http.StatusTooManyRequests, "Слишком много созданий сообществ, попробуйте позже")
				return
			}

			var req createCommunityRequest
			if err := decodeJSON(w, r, &req); err != nil {
				writeError(w, http.StatusBadRequest, "Некорректный JSON")
				return
			}

			created, err := createCommunity(db, userID, req)
			if err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"community": created})
		default:
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		}
	})

	mux.HandleFunc("/api/communities/", func(w http.ResponseWriter, r *http.Request) {
		tail := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/communities/"), "/")
		if tail == "" {
			writeError(w, http.StatusNotFound, "Не найдено")
			return
		}
		parts := strings.Split(tail, "/")
		communityID, ok := parseCommunityID(parts[0])
		if !ok {
			writeError(w, http.StatusBadRequest, "Некорректный id сообщества")
			return
		}
		authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
		if len(parts) == 1 {
			switch r.Method {
			case http.MethodGet:
				item, err := getCommunityByID(db, communityID, authUserID, hasAuth)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"community": item})
			case http.MethodPatch:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				var req updateCommunityRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				item, err := updateCommunity(db, communityID, userID, req)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"community": item})
			case http.MethodDelete:
				userID, ok := authenticatedUserID(w, r, sessions)
				if !ok {
					return
				}
				if err := deleteCommunity(db, communityID, userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			default:
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
			}
			return
		}
		switch parts[1] {
		case "join":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if err := joinCommunity(db, communityID, userID); err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "leave":
			if r.Method != http.MethodPost {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if err := leaveCommunity(db, communityID, userID); err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		case "join-requests":
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if len(parts) == 2 && r.Method == http.MethodPost {
				if !communityJoinRequestRateLimiter.Allow(fmt.Sprintf("community-join-request:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много заявок, попробуйте позже")
					return
				}
				var body joinRequestBody
				if err := decodeJSON(w, r, &body); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				if err := createJoinRequest(db, communityID, userID, body.Message); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]string{"status": "ok"})
				return
			}
			if len(parts) == 2 && r.Method == http.MethodGet {
				items, err := listJoinRequests(db, communityID, userID)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"requests": items})
				return
			}
			if len(parts) == 3 && parts[2] == "mine" && r.Method == http.MethodDelete {
				if err := cancelMyJoinRequest(db, communityID, userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			if len(parts) == 4 && r.Method == http.MethodPost && (parts[3] == "approve" || parts[3] == "reject") {
				requestID, ok := parseCommunityID(parts[2])
				if !ok {
					writeError(w, http.StatusBadRequest, "Некорректный id заявки")
					return
				}
				var err error
				if parts[3] == "approve" {
					err = approveJoinRequest(db, communityID, requestID, userID)
				} else {
					err = rejectJoinRequest(db, communityID, requestID, userID)
				}
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		case "members":
			if len(parts) == 2 && r.Method == http.MethodGet {
				limit := parseLimit(r.URL.Query().Get("limit"), 50, 200)
				offset := parseLimit(r.URL.Query().Get("offset"), 0, 100000)
				items, err := listCommunityMembers(db, communityID, authUserID, hasAuth, limit, offset)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"members": items})
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if len(parts) == 4 && parts[3] == "role" && r.Method == http.MethodPut {
				var req changeRoleRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				if err := changeCommunityRole(db, communityID, parts[2], req.Role, userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			if len(parts) == 3 && r.Method == http.MethodDelete {
				if err := kickCommunityMember(db, communityID, parts[2], userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		case "posts":
			if len(parts) == 2 && r.Method == http.MethodGet {
				limit := parseLimit(r.URL.Query().Get("limit"), 20, 100)
				beforeID := parseIDOrZero(r.URL.Query().Get("before_id"))
				items, nextCursor, err := listCommunityPosts(db, communityID, authUserID, hasAuth, limit, beforeID)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]any{"posts": items, "next_cursor": nextCursor})
				return
			}
			userID, ok := authenticatedUserID(w, r, sessions)
			if !ok {
				return
			}
			if len(parts) == 2 && r.Method == http.MethodPost {
				if !communityPostRateLimiter.Allow(fmt.Sprintf("community-post:%d", userID)) {
					writeError(w, http.StatusTooManyRequests, "Слишком много публикаций в сообществах, попробуйте позже")
					return
				}
				var req createPostRequest
				if err := decodeJSON(w, r, &req); err != nil {
					writeError(w, http.StatusBadRequest, "Некорректный JSON")
					return
				}
				item, err := createCommunityPost(db, communityID, userID, req)
				if err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusCreated, map[string]any{"post": item})
				return
			}
			if len(parts) == 3 && r.Method == http.MethodDelete {
				if err := deleteCommunityPost(db, communityID, parts[2], userID); err != nil {
					handleCommunityError(w, err)
					return
				}
				writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
				return
			}
			writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
		default:
			writeError(w, http.StatusNotFound, "Не найдено")
		}
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
		if strings.HasSuffix(r.URL.Path, "/communities") {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			publicID := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/communities")
			publicID = strings.TrimSpace(strings.Trim(publicID, "/"))
			if !isValidUserPublicID(publicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id пользователя")
				return
			}
			authUserID, hasAuth := optionalAuthenticatedUserID(r, sessions)
			items, err := listUserCommunities(db, publicID, authUserID, hasAuth)
			if err != nil {
				handleCommunityError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"communities": items})
			return
		}
		if !strings.HasSuffix(r.URL.Path, "/posts") {
			if r.Method != http.MethodGet {
				writeError(w, http.StatusMethodNotAllowed, "Метод не поддерживается")
				return
			}
			publicID := strings.TrimSpace(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/users/"), "/"))
			if !isValidUserPublicID(publicID) {
				writeError(w, http.StatusBadRequest, "Некорректный id пользователя")
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

	mux.Handle("/", staticSecurity(injectHTML(http.FileServer(http.Dir("./web")))))

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

ALTER TABLE communities
    ADD COLUMN IF NOT EXISTS region TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS email TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS avatar_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS cover_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS is_deleted BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS communities_created_at_idx
ON communities(created_at DESC);

CREATE TABLE IF NOT EXISTS community_members (
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (community_id, user_id)
);

ALTER TABLE community_members
    ADD COLUMN IF NOT EXISTS role TEXT NOT NULL DEFAULT 'member';

DO $$
BEGIN
  ALTER TABLE community_members DROP CONSTRAINT IF EXISTS community_members_role_check;
  ALTER TABLE community_members ADD CONSTRAINT community_members_role_check
    CHECK (role IN ('owner', 'admin', 'moderator', 'member'));
END$$;

CREATE UNIQUE INDEX IF NOT EXISTS community_members_owner_uniq_idx
    ON community_members(community_id) WHERE role = 'owner';

UPDATE community_members cm
SET role = 'owner'
FROM communities c
WHERE cm.community_id = c.id
  AND cm.user_id = c.creator_id
  AND cm.role = 'member';

INSERT INTO community_members (community_id, user_id, role)
SELECT c.id, c.creator_id, 'owner'
FROM communities c
WHERE NOT EXISTS (
    SELECT 1 FROM community_members cm
    WHERE cm.community_id = c.id AND cm.user_id = c.creator_id
)
ON CONFLICT (community_id, user_id) DO NOTHING;

CREATE INDEX IF NOT EXISTS community_members_user_idx
ON community_members(user_id);

CREATE TABLE IF NOT EXISTS community_join_requests (
    id BIGSERIAL PRIMARY KEY,
    community_id BIGINT NOT NULL REFERENCES communities(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    message TEXT NOT NULL DEFAULT '' CHECK (char_length(message) <= 500),
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    decided_at TIMESTAMPTZ,
    decided_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
    CHECK (status IN ('pending', 'approved', 'rejected', 'canceled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS community_join_requests_pending_uniq_idx
    ON community_join_requests(community_id, user_id) WHERE status = 'pending';

CREATE INDEX IF NOT EXISTS community_join_requests_community_idx
    ON community_join_requests(community_id, status, created_at DESC);

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

ALTER TABLE posts
    ADD COLUMN IF NOT EXISTS community_id BIGINT REFERENCES communities(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS posts_author_created_idx
    ON posts(author_id, created_at DESC) WHERE is_deleted = FALSE;

CREATE INDEX IF NOT EXISTS posts_feed_idx
    ON posts(created_at DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE INDEX IF NOT EXISTS posts_type_created_idx
    ON posts(type, created_at DESC) WHERE is_deleted = FALSE AND privacy_level = 'public';

CREATE INDEX IF NOT EXISTS posts_community_created_idx
    ON posts(community_id, created_at DESC) WHERE is_deleted = FALSE AND community_id IS NOT NULL;

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

	if err := migratePublicationSeedData(db); err != nil {
		return fmt.Errorf("migrate publication seed data: %w", err)
	}

	return nil
}

type seedPost struct {
	PublicID   string
	Title      string
	Content    string
	Tags       []string
	CoverURL   string
	CreatedAt  time.Time
	ViewsCount int
	LikesCount int
}

func migratePublicationSeedData(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	const seedUserEmail = "seed.lastop@local"
	var seedAuthorID int64
	if err := tx.QueryRow(`
		INSERT INTO users (public_id, first_name, last_name, full_name, email, password_hash, position, company_name, bio, avatar_url)
		VALUES ('usr_seed_lastop', 'LASTOP', 'Digital', 'LASTOP Digital', $1, '$2a$10$H8hT4aSt3D8QdDkbx73Z9OBPFnfRTE8Yv5IadB1iKsFcfoTmya8Ie', 'Редакция LASTOP', 'LASTOP', 'Системный аккаунт для миграции публикаций', '')
		ON CONFLICT (email) DO UPDATE
		SET full_name = EXCLUDED.full_name,
		    position = EXCLUDED.position,
		    company_name = EXCLUDED.company_name
		RETURNING id
	`, seedUserEmail).Scan(&seedAuthorID); err != nil {
		return err
	}

	now := time.Now().UTC()
	seeds := []seedPost{
		{
			PublicID:   "pst_seed_customs_2026",
			Title:      "ФТС обновила требования к таможенному декларированию импорта",
			Content:    "С 1 мая 2026 года вступают в силу изменения в порядке декларирования отдельных категорий товаров. Бизнесу рекомендуется заранее проверить шаблоны документов и обновить внутренние регламенты.",
			Tags:       []string{"таможня", "вэд", "регуляторика"},
			CreatedAt:  now.Add(-18 * time.Hour),
			ViewsCount: 1820,
			LikesCount: 47,
		},
		{
			PublicID:   "pst_seed_routes_2026",
			Title:      "Рынок фиксирует рост контейнерных перевозок на 18% в Q2 2026",
			Content:    "Участники отрасли связывают рост с развитием альтернативных международных маршрутов и повышением прозрачности цифрового трекинга грузов.",
			Tags:       []string{"логистика", "контейнеры", "аналитика"},
			CreatedAt:  now.Add(-48 * time.Hour),
			ViewsCount: 2400,
			LikesCount: 52,
		},
		{
			PublicID:   "pst_seed_tms_ai",
			Title:      "LASTOP Digital запустил модуль аналитики TMS с ИИ‑прогнозированием",
			Content:    "Новый модуль помогает предсказывать отклонения SLA и автоматически сигнализирует о рисках на отдельных участках цепочки поставок.",
			Tags:       []string{"технологии", "tms", "ai"},
			CreatedAt:  now.Add(-72 * time.Hour),
			ViewsCount: 940,
			LikesCount: 34,
		},
	}

	for _, item := range seeds {
		var pgTags pgtype.FlatArray[string] = item.Tags

		if _, err := tx.Exec(`
			INSERT INTO posts (public_id, author_id, type, title, content, tags, cover_url, privacy_level, views_count, likes_count, created_at, updated_at)
			VALUES ($1, $2, 'news', $3, $4, $5, NULLIF($6, ''), 'public', $7, $8, $9, NOW())
			ON CONFLICT (public_id) DO NOTHING
		`, item.PublicID, seedAuthorID, item.Title, item.Content, pgTags, item.CoverURL, item.ViewsCount, item.LikesCount, item.CreatedAt); err != nil {
			return err
		}
	}

	return tx.Commit()
}

var (
	errValidation         = errors.New("validation error")
	errEmailTaken         = errors.New("email taken")
	errInvalidCredentials = errors.New("invalid credentials")
	errConflict           = errors.New("conflict")
	errNotFound           = errors.New("not found")
	errForbidden          = errors.New("forbidden")
	errAlreadyMember      = errors.New("already member")
	errRequestPending     = errors.New("request pending")
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

type communityListFilters struct {
	Privacy  string
	Category string
	Region   string
	Query    string
}

func listCommunities(db *sql.DB, authUserID int64, hasAuth bool, filters communityListFilters) ([]community, error) {
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	args := []any{currentUserID}
	query := `
		SELECT c.id,
		       c.name,
		       c.category,
		       c.description,
		       COALESCE(c.region, ''),
		       COALESCE(c.website, ''),
		       COALESCE(c.email, ''),
		       COALESCE(c.phone, ''),
		       COALESCE(c.avatar_url, ''),
		       COALESCE(c.cover_url, ''),
		       COALESCE(c.color, ''),
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
		       COALESCE(cm_user.role, ''),
		       COALESCE(m.members_count, 0),
		       c.created_at
		FROM communities c
		JOIN users u ON u.id = c.creator_id
		LEFT JOIN community_members cm_user
		       ON cm_user.community_id = c.id
		      AND cm_user.user_id = $1::bigint
		LEFT JOIN (
			SELECT community_id, COUNT(*)::int AS members_count
			FROM community_members
			GROUP BY community_id
		) m ON m.community_id = c.id
		WHERE c.is_deleted = FALSE`
	if p := normalizeCommunityPrivacy(filters.Privacy); p != "" {
		query += fmt.Sprintf(" AND c.privacy_level = $%d", len(args)+1)
		args = append(args, p)
	}
	if cat := strings.TrimSpace(filters.Category); cat != "" {
		query += fmt.Sprintf(" AND c.category = $%d", len(args)+1)
		args = append(args, cat)
	}
	if region := strings.TrimSpace(filters.Region); region != "" {
		query += fmt.Sprintf(" AND c.region ILIKE $%d", len(args)+1)
		args = append(args, "%"+region+"%")
	}
	if q := strings.TrimSpace(filters.Query); q != "" {
		query += fmt.Sprintf(" AND (c.name ILIKE $%d OR c.description ILIKE $%d)", len(args)+1, len(args)+1)
		args = append(args, "%"+q+"%")
	}
	query += " ORDER BY c.created_at DESC"

	rows, err := db.Query(query, args...)
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
			&item.Region,
			&item.Website,
			&item.Email,
			&item.Phone,
			&item.AvatarURL,
			&item.CoverURL,
			&item.Color,
			&rawTags,
			&item.Privacy,
			&item.CreatorID,
			&item.IsMember,
			&item.MyRole,
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
		       COALESCE(c.region, ''),
		       COALESCE(c.website, ''),
		       COALESCE(c.email, ''),
		       COALESCE(c.phone, ''),
		       COALESCE(c.avatar_url, ''),
		       COALESCE(c.cover_url, ''),
		       COALESCE(c.color, ''),
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
		       COALESCE(cm_user.role, ''),
		       (SELECT COUNT(*)::int FROM community_members cm WHERE cm.community_id = c.id) AS members_count,
		       c.created_at
		FROM communities c
		LEFT JOIN users u ON u.id = c.creator_id
		LEFT JOIN community_members cm_user
		       ON cm_user.community_id = c.id
		      AND cm_user.user_id = $2::bigint
		WHERE c.id = $1 AND c.is_deleted = FALSE
	`, communityID, currentUserID).Scan(
		&item.ID,
		&item.Name,
		&item.Category,
		&item.Description,
		&item.Region,
		&item.Website,
		&item.Email,
		&item.Phone,
		&item.AvatarURL,
		&item.CoverURL,
		&item.Color,
		&tagsJSON,
		&item.Privacy,
		&item.CreatorID,
		&item.IsMember,
		&item.MyRole,
		&item.Members,
		&item.CreatedAt,
	)
	if err != nil {
		return community{}, err
	}
	if err := json.Unmarshal(tagsJSON, &item.Tags); err != nil {
		return community{}, fmt.Errorf("decode community tags: %w", err)
	}
	if item.Privacy == "closed" && !item.IsMember && !canModerateCommunity(item.MyRole) {
		item.Description = ""
		item.Region = ""
		item.Website = ""
		item.Email = ""
		item.Phone = ""
		item.CoverURL = ""
		item.Tags = []string{}
	}
	return item, nil
}

func createCommunity(db *sql.DB, creatorID int64, req createCommunityRequest) (community, error) {
	name, err := validateCommunityName(req.Name)
	if err != nil {
		return community{}, err
	}
	category := validateCommunityCategory(req.Category)
	privacy := normalizeCommunityPrivacy(req.PrivacyLevel)
	if privacy == "" {
		return community{}, fmt.Errorf("%w: privacy_level должен быть open или closed", errValidation)
	}
	description, err := validateCommunityDescription(req.Description)
	if err != nil {
		return community{}, err
	}
	region, err := validateRegion(req.Region)
	if err != nil {
		return community{}, err
	}
	website, err := validateWebsite(req.Website)
	if err != nil {
		return community{}, err
	}
	email, err := validateEmail(req.Email)
	if err != nil {
		return community{}, err
	}
	phone, err := validatePhone(req.Phone)
	if err != nil {
		return community{}, err
	}
	avatarURL, err := validateAvatarURL(req.AvatarURL)
	if err != nil {
		return community{}, err
	}
	color, err := validateColor(req.Color)
	if err != nil {
		return community{}, err
	}
	tags, err := validateTags(req.Tags)
	if err != nil {
		return community{}, err
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
		INSERT INTO communities(name, category, description, region, website, email, phone, avatar_url, color, tags, privacy_level, creator_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::text[], $11, $12)
		RETURNING id, name, category, description, region, website, email, phone, avatar_url, cover_url, color,
			COALESCE(array_to_json(tags), '[]'::json), privacy_level, created_at,
			(SELECT public_id FROM users WHERE id = creator_id)
	`, name, category, description, region, website, email, phone, avatarURL, color, pgTags, privacy, creatorID).
		Scan(&created.ID, &created.Name, &created.Category, &created.Description, &created.Region, &created.Website, &created.Email, &created.Phone, &created.AvatarURL, &created.CoverURL, &created.Color, &rawTags, &created.Privacy, &created.CreatedAt, &created.CreatorID)
	if err != nil {
		return community{}, err
	}
	if err := json.Unmarshal(rawTags, &created.Tags); err != nil {
		return community{}, fmt.Errorf("decode created community tags: %w", err)
	}
	if _, err := tx.Exec(`
		INSERT INTO community_members(community_id, user_id, role)
		VALUES ($1, $2, 'owner')
		ON CONFLICT (community_id, user_id) DO NOTHING
	`, created.ID, creatorID); err != nil {
		return community{}, err
	}
	if err := tx.Commit(); err != nil {
		return community{}, err
	}
	created.Members = 1
	created.IsMember = true
	created.MyRole = "owner"
	return created, nil
}

func joinCommunity(db *sql.DB, communityID, userID int64) error {
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id = $1 AND is_deleted = FALSE`, communityID).Scan(&privacy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("%w: сообщество не найдено", errNotFound)
		}
		return err
	}
	if role, err := getUserCommunityRole(db, communityID, userID); err != nil {
		return err
	} else if role != "" {
		return fmt.Errorf("%w: пользователь уже состоит в сообществе", errAlreadyMember)
	}
	if privacy == "closed" {
		return fmt.Errorf("%w: сообщество закрытое, отправьте заявку", errForbidden)
	}
	if _, err := db.Exec(`
		INSERT INTO community_members(community_id, user_id, role)
		VALUES ($1, $2, 'member')
		ON CONFLICT (community_id, user_id) DO NOTHING
	`, communityID, userID); err != nil {
		return err
	}
	return nil
}

func leaveCommunity(db *sql.DB, communityID, userID int64) error {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return err
	}
	if role == "" {
		return fmt.Errorf("%w: участник не найден", errNotFound)
	}
	if role == "owner" {
		return fmt.Errorf("%w: Сначала передайте роль владельца другому участнику или удалите сообщество", errForbidden)
	}
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
		return errNotFound
	}
	return nil
}

func normalizePrivacyLevel(value string) string {
	return normalizeCommunityPrivacy(value)
}

func normalizeCommunityPrivacy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "open":
		return "open"
	case "closed":
		return "closed"
	default:
		return ""
	}
}

func validateCommunityName(s string) (string, error) {
	s = strings.TrimSpace(s)
	if count := utf8.RuneCountInString(s); count < 3 || count > 120 {
		return "", fmt.Errorf("%w: название сообщества должно быть от 3 до 120 символов", errValidation)
	}
	return s, nil
}

func validateCommunityCategory(s string) string {
	s = strings.TrimSpace(s)
	switch s {
	case "Логистика", "ВЭД", "IT", "Регионы", "Карьера", "Другое":
		return s
	default:
		return "Другое"
	}
}

func validateCommunityDescription(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > 1200 {
		return "", fmt.Errorf("%w: описание слишком длинное", errValidation)
	}
	return s, nil
}

func validateRegion(s string) (string, error) {
	s = strings.TrimSpace(s)
	if utf8.RuneCountInString(s) > 120 {
		return "", fmt.Errorf("%w: регион слишком длинный", errValidation)
	}
	return s, nil
}

func validateWebsite(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > 500 || !(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
		return "", fmt.Errorf("%w: website должен начинаться с http:// или https:// и быть не длиннее 500", errValidation)
	}
	return s, nil
}

func validateEmail(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if _, err := mail.ParseAddress(s); err != nil {
		return "", fmt.Errorf("%w: email некорректен", errValidation)
	}
	return s, nil
}

func validatePhone(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if utf8.RuneCountInString(s) < 5 || utf8.RuneCountInString(s) > 32 {
		return "", fmt.Errorf("%w: телефон должен быть от 5 до 32 символов", errValidation)
	}
	if ok, _ := regexp.MatchString(`^[0-9+\-()\s]+$`, s); !ok {
		return "", fmt.Errorf("%w: телефон содержит недопустимые символы", errValidation)
	}
	return s, nil
}

func validateColor(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if ok, _ := regexp.MatchString(`^#[0-9A-Fa-f]{6}$`, s); !ok {
		return "", fmt.Errorf("%w: color должен быть HEX-цветом вида #RRGGBB", errValidation)
	}
	return s, nil
}

func validateAvatarURL(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil
	}
	if len(s) > 500 || !(strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")) {
		return "", fmt.Errorf("%w: avatar_url должен начинаться с http:// или https:// и быть не длиннее 500", errValidation)
	}
	return s, nil
}

func validateTags(tags []string) ([]string, error) {
	out := make([]string, 0, len(tags))
	seen := make(map[string]struct{}, len(tags))
	for _, t := range tags {
		tag := strings.TrimSpace(t)
		if tag == "" {
			continue
		}
		if count := utf8.RuneCountInString(tag); count < 1 || count > 40 {
			return nil, fmt.Errorf("%w: каждый тег должен быть длиной 1..40 символов", errValidation)
		}
		key := strings.ToLower(tag)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, tag)
		if len(out) > 10 {
			return nil, fmt.Errorf("%w: максимум 10 тегов", errValidation)
		}
	}
	return out, nil
}

func canManageCommunity(role string) bool { return role == "owner" || role == "admin" }
func canModerateCommunity(role string) bool {
	return role == "owner" || role == "admin" || role == "moderator"
}

func getUserCommunityRole(db *sql.DB, communityID, userID int64) (string, error) {
	var role string
	err := db.QueryRow(`SELECT role FROM community_members WHERE community_id=$1 AND user_id=$2`, communityID, userID).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return role, err
}

func updateCommunity(db *sql.DB, communityID, userID int64, req updateCommunityRequest) (community, error) {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return community{}, err
	}
	if !canManageCommunity(role) {
		return community{}, fmt.Errorf("%w: недостаточно прав", errForbidden)
	}
	var sets []string
	var args []any
	i := 1
	if req.Name != nil {
		v, err := validateCommunityName(*req.Name)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("name=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Category != nil {
		sets = append(sets, fmt.Sprintf("category=$%d", i))
		args = append(args, validateCommunityCategory(*req.Category))
		i++
	}
	if req.Description != nil {
		v, err := validateCommunityDescription(*req.Description)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("description=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Region != nil {
		v, err := validateRegion(*req.Region)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("region=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Website != nil {
		v, err := validateWebsite(*req.Website)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("website=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Email != nil {
		v, err := validateEmail(*req.Email)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("email=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Phone != nil {
		v, err := validatePhone(*req.Phone)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("phone=$%d", i))
		args = append(args, v)
		i++
	}
	if req.PrivacyLevel != nil {
		v := normalizeCommunityPrivacy(*req.PrivacyLevel)
		if v == "" {
			return community{}, fmt.Errorf("%w: privacy_level должен быть open или closed", errValidation)
		}
		sets = append(sets, fmt.Sprintf("privacy_level=$%d", i))
		args = append(args, v)
		i++
	}
	if req.AvatarURL != nil {
		v, err := validateAvatarURL(*req.AvatarURL)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("avatar_url=$%d", i))
		args = append(args, v)
		i++
	}
	if req.CoverURL != nil {
		v, err := validateAvatarURL(*req.CoverURL)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("cover_url=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Color != nil {
		v, err := validateColor(*req.Color)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("color=$%d", i))
		args = append(args, v)
		i++
	}
	if req.Tags != nil {
		v, err := validateTags(*req.Tags)
		if err != nil {
			return community{}, err
		}
		sets = append(sets, fmt.Sprintf("tags=$%d::text[]", i))
		args = append(args, pgtype.FlatArray[string](v))
		i++
	}
	if len(sets) > 0 {
		sets = append(sets, "updated_at=NOW()")
		args = append(args, communityID)
		if _, err := db.Exec("UPDATE communities SET "+strings.Join(sets, ", ")+" WHERE id=$"+strconv.Itoa(i)+" AND is_deleted = FALSE", args...); err != nil {
			return community{}, err
		}
	}
	return getCommunityByID(db, communityID, userID, true)
}

func deleteCommunity(db *sql.DB, communityID, userID int64) error {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return err
	}
	if role != "owner" {
		return fmt.Errorf("%w: удалить сообщество может только владелец", errForbidden)
	}
	res, err := db.Exec(`UPDATE communities SET is_deleted=TRUE, updated_at=NOW() WHERE id=$1 AND is_deleted=FALSE`, communityID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func createJoinRequest(db *sql.DB, communityID, userID int64, message string) error {
	message = strings.TrimSpace(message)
	if utf8.RuneCountInString(message) > 500 {
		return fmt.Errorf("%w: сообщение не должно превышать 500 символов", errValidation)
	}
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id=$1 AND is_deleted=FALSE`, communityID).Scan(&privacy); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if privacy != "closed" {
		return fmt.Errorf("%w: заявки доступны только для закрытых сообществ", errValidation)
	}
	if role, err := getUserCommunityRole(db, communityID, userID); err != nil {
		return err
	} else if role != "" {
		return errAlreadyMember
	}
	_, err := db.Exec(`INSERT INTO community_join_requests(community_id, user_id, message) VALUES ($1,$2,$3)`, communityID, userID, message)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return errRequestPending
		}
		return err
	}
	return nil
}

func listJoinRequests(db *sql.DB, communityID, userID int64) ([]communityJoinRequestDTO, error) {
	role, err := getUserCommunityRole(db, communityID, userID)
	if err != nil {
		return nil, err
	}
	if !canModerateCommunity(role) {
		return nil, errForbidden
	}
	rows, err := db.Query(`
		SELECT r.id, u.public_id, u.full_name, r.message, r.status, r.created_at
		FROM community_join_requests r
		JOIN users u ON u.id = r.user_id
		WHERE r.community_id = $1 AND r.status = 'pending'
		ORDER BY r.created_at DESC
	`, communityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []communityJoinRequestDTO
	for rows.Next() {
		var item communityJoinRequestDTO
		if err := rows.Scan(&item.ID, &item.UserPublicID, &item.FullName, &item.Message, &item.Status, &item.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func approveJoinRequest(db *sql.DB, communityID, requestID, deciderID int64) error {
	return decideJoinRequest(db, communityID, requestID, deciderID, "approved")
}

func rejectJoinRequest(db *sql.DB, communityID, requestID, deciderID int64) error {
	return decideJoinRequest(db, communityID, requestID, deciderID, "rejected")
}

func decideJoinRequest(db *sql.DB, communityID, requestID, deciderID int64, status string) error {
	role, err := getUserCommunityRole(db, communityID, deciderID)
	if err != nil {
		return err
	}
	if !canModerateCommunity(role) {
		return errForbidden
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var applicantID int64
	if err := tx.QueryRow(`
		UPDATE community_join_requests
		SET status = $3, decided_at = NOW(), decided_by = $4
		WHERE id = $1 AND community_id = $2 AND status = 'pending'
		RETURNING user_id
	`, requestID, communityID, status, deciderID).Scan(&applicantID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if status == "approved" {
		if _, err := tx.Exec(`
			INSERT INTO community_members(community_id, user_id, role)
			VALUES ($1, $2, 'member')
			ON CONFLICT (community_id, user_id) DO NOTHING
		`, communityID, applicantID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func cancelMyJoinRequest(db *sql.DB, communityID, userID int64) error {
	res, err := db.Exec(`
		UPDATE community_join_requests
		SET status='canceled', decided_at=NOW(), decided_by=$3
		WHERE community_id=$1 AND user_id=$2 AND status='pending'
	`, communityID, userID, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errNotFound
	}
	return nil
}

func listCommunityMembers(db *sql.DB, communityID, authUserID int64, hasAuth bool, limit, offset int) ([]communityMemberDTO, error) {
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id=$1 AND is_deleted=FALSE`, communityID).Scan(&privacy); err != nil {
		return nil, err
	}
	if privacy == "closed" {
		if !hasAuth {
			return nil, errForbidden
		}
		role, err := getUserCommunityRole(db, communityID, authUserID)
		if err != nil {
			return nil, err
		}
		if role == "" {
			return nil, errForbidden
		}
	}
	rows, err := db.Query(`
		SELECT u.public_id, u.full_name, COALESCE(u.avatar_url, ''), cm.role, cm.joined_at
		FROM community_members cm
		JOIN users u ON u.id = cm.user_id
		WHERE cm.community_id = $1
		ORDER BY CASE cm.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 WHEN 'moderator' THEN 2 ELSE 3 END, cm.joined_at ASC
		LIMIT $2 OFFSET $3
	`, communityID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []communityMemberDTO
	for rows.Next() {
		var m communityMemberDTO
		if err := rows.Scan(&m.UserPublicID, &m.FullName, &m.AvatarURL, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func changeCommunityRole(db *sql.DB, communityID int64, targetUserPublicID, newRole string, actorID int64) error {
	if newRole != "admin" && newRole != "moderator" && newRole != "member" {
		return fmt.Errorf("%w: роль должна быть admin/moderator/member", errValidation)
	}
	actorRole, err := getUserCommunityRole(db, communityID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" {
		return errForbidden
	}
	var targetID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id=$1`, targetUserPublicID).Scan(&targetID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if targetID == actorID {
		return fmt.Errorf("%w: нельзя менять свою роль", errValidation)
	}
	var currentRole string
	if err := db.QueryRow(`SELECT role FROM community_members WHERE community_id=$1 AND user_id=$2`, communityID, targetID).Scan(&currentRole); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if currentRole == "owner" {
		return fmt.Errorf("%w: нельзя менять роль владельца", errForbidden)
	}
	_, err = db.Exec(`UPDATE community_members SET role=$3 WHERE community_id=$1 AND user_id=$2`, communityID, targetID, newRole)
	return err
}

func kickCommunityMember(db *sql.DB, communityID int64, targetUserPublicID string, actorID int64) error {
	actorRole, err := getUserCommunityRole(db, communityID, actorID)
	if err != nil {
		return err
	}
	if actorRole != "owner" && actorRole != "admin" {
		return errForbidden
	}
	var targetID int64
	var targetRole string
	if err := db.QueryRow(`
		SELECT u.id, cm.role
		FROM users u
		JOIN community_members cm ON cm.user_id = u.id AND cm.community_id = $1
		WHERE u.public_id = $2
	`, communityID, targetUserPublicID).Scan(&targetID, &targetRole); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if targetID == actorID {
		return fmt.Errorf("%w: нельзя исключить себя", errValidation)
	}
	if targetRole == "owner" {
		return errForbidden
	}
	if actorRole == "admin" && targetRole == "admin" {
		return errForbidden
	}
	_, err = db.Exec(`DELETE FROM community_members WHERE community_id=$1 AND user_id=$2`, communityID, targetID)
	return err
}

func listCommunityPosts(db *sql.DB, communityID, userID int64, hasAuth bool, limit int, beforeID int64) ([]post, *int64, error) {
	var privacy string
	if err := db.QueryRow(`SELECT privacy_level FROM communities WHERE id=$1 AND is_deleted=FALSE`, communityID).Scan(&privacy); err != nil {
		return nil, nil, err
	}
	if privacy == "closed" {
		if !hasAuth {
			return nil, nil, errForbidden
		}
		role, err := getUserCommunityRole(db, communityID, userID)
		if err != nil {
			return nil, nil, err
		}
		if role == "" {
			return nil, nil, errForbidden
		}
	}
	args := []any{communityID, limit + 1}
	query := `
		SELECT p.id, p.public_id, p.type, p.title, p.content, COALESCE(p.cover_url, ''),
		       COALESCE(array_to_json(p.tags), '[]'::json), p.privacy_level, p.likes_count, p.comments_count,
		       p.views_count, p.created_at, p.author_id,
		       COALESCE(u.public_id, ''), COALESCE(u.full_name, ''), COALESCE(NULLIF(u.position, ''), u.company_name, ''), COALESCE(u.avatar_url, ''),
		       FALSE, COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN communities c ON c.id = p.community_id
		WHERE p.community_id = $1 AND p.is_deleted = FALSE`
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
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.CommunityName, &item.CommunityID); err != nil {
			return nil, nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		item.Text = item.Content
		items = append(items, item)
	}
	var next *int64
	if len(items) > limit {
		last := items[limit-1].ID
		next = &last
		items = items[:limit]
	}
	return items, next, rows.Err()
}

func createCommunityPost(db *sql.DB, communityID, authorID int64, req createPostRequest) (post, error) {
	role, err := getUserCommunityRole(db, communityID, authorID)
	if err != nil {
		return post{}, err
	}
	if role == "" {
		return post{}, errForbidden
	}
	title := strings.TrimSpace(req.Title)
	if count := utf8.RuneCountInString(title); count < 3 || count > 200 {
		return post{}, fmt.Errorf("%w: заголовок должен быть от 3 до 200 символов", errValidation)
	}
	content := strings.TrimSpace(req.Content)
	if count := utf8.RuneCountInString(content); count < 10 || count > 20000 {
		return post{}, fmt.Errorf("%w: текст должен быть от 10 до 20000 символов", errValidation)
	}
	tags, err := validateTags(req.Tags)
	if err != nil {
		return post{}, err
	}
	publicID, err := newPublicPostID()
	if err != nil {
		return post{}, err
	}
	var created post
	var tagsJSON []byte
	if err := db.QueryRow(`
		INSERT INTO posts (public_id, author_id, community_id, type, title, content, tags, privacy_level)
		VALUES ($1,$2,$3,'news',$4,$5,$6::text[],'public')
		RETURNING id, public_id, type, title, content, COALESCE(cover_url,''), COALESCE(array_to_json(tags),'[]'::json),
		          privacy_level, likes_count, comments_count, views_count, created_at, author_id
	`, publicID, authorID, communityID, title, content, pgtype.FlatArray[string](tags)).Scan(
		&created.ID, &created.PublicID, &created.Type, &created.Title, &created.Content, &created.CoverURL, &tagsJSON,
		&created.PrivacyLevel, &created.LikesCount, &created.CommentsCount, &created.ViewsCount, &created.CreatedAt, &created.AuthorID,
	); err != nil {
		return post{}, err
	}
	_ = json.Unmarshal(tagsJSON, &created.Tags)
	created.Text = created.Content
	created.CommunityID = communityID
	return hydratePostAuthor(db, created)
}

func deleteCommunityPost(db *sql.DB, communityID int64, postPublicID string, userID int64) error {
	var authorID int64
	if err := db.QueryRow(`SELECT author_id FROM posts WHERE public_id=$1 AND community_id=$2 AND is_deleted=FALSE`, postPublicID, communityID).Scan(&authorID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errNotFound
		}
		return err
	}
	if authorID != userID {
		role, err := getUserCommunityRole(db, communityID, userID)
		if err != nil {
			return err
		}
		if !canModerateCommunity(role) {
			return errForbidden
		}
	}
	_, err := db.Exec(`UPDATE posts SET is_deleted=TRUE, updated_at=NOW() WHERE public_id=$1 AND community_id=$2`, postPublicID, communityID)
	return err
}

func listUserCommunities(db *sql.DB, userPublicID string, authUserID int64, hasAuth bool) ([]community, error) {
	var userID int64
	if err := db.QueryRow(`SELECT id FROM users WHERE public_id=$1`, userPublicID).Scan(&userID); err != nil {
		return nil, err
	}
	var currentUserID sql.NullInt64
	if hasAuth {
		currentUserID = sql.NullInt64{Int64: authUserID, Valid: true}
	}
	rows, err := db.Query(`
		SELECT c.id, c.name, c.category, c.description, COALESCE(c.region,''), COALESCE(c.website,''), COALESCE(c.email,''), COALESCE(c.phone,''),
		       COALESCE(c.avatar_url,''), COALESCE(c.cover_url,''), COALESCE(c.color,''), COALESCE(array_to_json(c.tags), '[]'::json),
		       c.privacy_level, cu.public_id,
		       ($2::bigint IS NOT NULL AND EXISTS (SELECT 1 FROM community_members cmx WHERE cmx.community_id=c.id AND cmx.user_id=$2::bigint)) AS is_member,
		       COALESCE(cm2.role,''), (SELECT COUNT(*)::int FROM community_members cmcnt WHERE cmcnt.community_id=c.id), c.created_at
		FROM communities c
		JOIN community_members cm ON cm.community_id=c.id
		JOIN users cu ON cu.id = c.creator_id
		LEFT JOIN community_members cm2 ON cm2.community_id=c.id AND cm2.user_id=$2::bigint
		WHERE cm.user_id=$1 AND c.is_deleted=FALSE
		ORDER BY cm.joined_at DESC
	`, userID, currentUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []community
	for rows.Next() {
		var item community
		var tagsJSON []byte
		if err := rows.Scan(&item.ID, &item.Name, &item.Category, &item.Description, &item.Region, &item.Website, &item.Email, &item.Phone,
			&item.AvatarURL, &item.CoverURL, &item.Color, &tagsJSON, &item.Privacy, &item.CreatorID, &item.IsMember, &item.MyRole, &item.Members, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(tagsJSON, &item.Tags)
		out = append(out, item)
	}
	return out, rows.Err()
}

func parseCommunityID(s string) (int64, bool) {
	id, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	return id, err == nil && id > 0
}

func handleCommunityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, errForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, errNotFound), errors.Is(err, sql.ErrNoRows):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, errConflict), errors.Is(err, errAlreadyMember), errors.Is(err, errRequestPending):
		writeError(w, http.StatusConflict, err.Error())
	default:
		log.Printf("community error: %v", err)
		writeError(w, http.StatusInternalServerError, "Ошибка сервера")
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
		       COALESCE(pl.user_id IS NOT NULL, FALSE), COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		LEFT JOIN communities c ON c.id = p.community_id
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
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.CommunityName, &item.CommunityID); err != nil {
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
		       COALESCE(pl.user_id IS NOT NULL, FALSE), COALESCE(c.name, ''), COALESCE(c.id, 0)
		FROM posts p
		JOIN users u ON u.id = p.author_id
		LEFT JOIN post_likes pl ON pl.post_id = p.id AND pl.user_id = $1::bigint
		LEFT JOIN communities c ON c.id = p.community_id
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
			&item.AuthorPublicID, &item.AuthorName, &item.AuthorRole, &item.AuthorAvatar, &item.IsLiked, &item.CommunityName, &item.CommunityID); err != nil {
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

// injectHTML возвращает http.Handler, который оборачивает next и для
// HTML-ответов вставляет htmlInject перед </head>. Для всех остальных
// ответов (CSS, JS, PNG, JSON API) ничего не меняет — пропускает as-is.
//
// TODO: если в цепочке появится gzip-сжатие, injectHTML должен стоять до gzip,
// чтобы вставка выполнялась по несжатому телу ответа.
func injectHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		rec := &htmlInjectWriter{
			ResponseWriter: w,
			buf:            &bytes.Buffer{},
		}
		next.ServeHTTP(rec, r)
		rec.flush()
	})
}

// htmlInjectWriter буферизует ответ, чтобы мы могли решить, HTML это или нет,
// и при необходимости вставить htmlInject перед </head>.
type htmlInjectWriter struct {
	http.ResponseWriter
	buf         *bytes.Buffer
	status      int
	wroteHeader bool
	isHTML      bool
	passThrough bool
}

func (w *htmlInjectWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true

	ct := w.ResponseWriter.Header().Get("Content-Type")
	w.isHTML = strings.HasPrefix(strings.ToLower(ct), "text/html")

	if !w.isHTML || status >= 300 {
		w.passThrough = true
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *htmlInjectWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	if w.passThrough {
		return w.ResponseWriter.Write(b)
	}
	return w.buf.Write(b)
}

func (w *htmlInjectWriter) flush() {
	if w.passThrough {
		return
	}
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	body := w.buf.Bytes()
	needle := []byte("</head>")
	idx := bytes.Index(body, needle)
	if idx < 0 {
		needleUpper := []byte("</HEAD>")
		idx = bytes.Index(body, needleUpper)
		if idx >= 0 {
			needle = needleUpper
		}
	}

	if idx < 0 {
		w.ResponseWriter.Header().Del("Content-Length")
		w.ResponseWriter.WriteHeader(w.status)
		_, _ = w.ResponseWriter.Write(body)
		return
	}

	var out bytes.Buffer
	out.Grow(len(body) + len(htmlInject))
	out.Write(body[:idx])
	out.WriteString(htmlInject)
	out.Write(body[idx:])

	w.ResponseWriter.Header().Del("Content-Length")
	w.ResponseWriter.WriteHeader(w.status)
	_, _ = io.Copy(w.ResponseWriter, &out)
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
