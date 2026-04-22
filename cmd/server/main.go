package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"
)

type greetingResponse struct {
	Message string `json:"message"`
}

type user struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
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

type authResponse struct {
	Token string `json:"token"`
	User  user   `json:"user"`
}

var emailRe = regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`)

func main() {
	db, err := initDB("data/app.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	mux := http.NewServeMux()

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
	})

	mux.Handle("/", http.FileServer(http.Dir("./web")))

	addr := ":8080"
	log.Printf("Server started at http://localhost%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func initDB(dbPath string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(time.Minute)

	const schema = `
CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    full_name TEXT NOT NULL,
    email TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`
	if _, err = db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}

	return db, nil
}

var (
	errValidation         = errors.New("validation error")
	errEmailTaken         = errors.New("email taken")
	errInvalidCredentials = errors.New("invalid credentials")
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

	res, err := db.Exec(`
		INSERT INTO users(first_name, last_name, full_name, email, password_hash)
		VALUES (?, ?, ?, ?, ?)
	`, strings.TrimSpace(req.FirstName), strings.TrimSpace(req.LastName), fullName, email, string(hash))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return user{}, errEmailTaken
		}
		return user{}, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return user{}, fmt.Errorf("last insert id: %w", err)
	}

	return user{ID: id, FirstName: strings.TrimSpace(req.FirstName), LastName: strings.TrimSpace(req.LastName), FullName: fullName, Email: email}, nil
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
		SELECT id, first_name, last_name, full_name, email, password_hash
		FROM users
		WHERE email = ?
	`, email).Scan(&u.ID, &u.FirstName, &u.LastName, &u.FullName, &u.Email, &passwordHash)
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
