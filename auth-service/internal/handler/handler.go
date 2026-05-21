package handler

import (
	"AuthService/internal/domain"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type Handler struct {
	authSvc  domain.AuthService
	validate *validator.Validate
}

func New(authSvc domain.AuthService) *Handler {
	return &Handler{
		authSvc:  authSvc,
		validate: validator.New(),
	}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /auth/register", h.Register)
	mux.HandleFunc("POST /auth/login", h.Login)
	mux.HandleFunc("POST /auth/refresh", h.Refresh)
	mux.HandleFunc("POST /auth/logout", h.Logout)

	mux.Handle("GET /auth/me", h.AuthMiddleware(http.HandlerFunc(h.Me)))
	return mux
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var input domain.RegisterInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.authSvc.Register(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrUserAlreadyExists) {
			writeError(w, http.StatusConflict, "email already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusCreated, tokens)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var input domain.LoginInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	tokens, err := h.authSvc.Login(r.Context(), input)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	tokens, err := h.authSvc.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		if errors.Is(err, domain.ErrInvalidToken) || errors.Is(err, domain.ErrTokenExpired) {
			writeError(w, http.StatusUnauthorized, "invalid or expired refresh token")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}

	writeJSON(w, http.StatusOK, tokens)
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		writeError(w, http.StatusBadRequest, "refresh_token is required")
		return
	}

	_ = h.authSvc.Logout(r.Context(), body.RefreshToken)
	writeJSON(w, http.StatusOK, map[string]string{"message": "logged out"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	claims := ClaimsFromContext(r.Context())
	writeJSON(w, http.StatusOK, map[string]string{
		"user_id": claims.UserID,
		"email":   claims.Email,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
