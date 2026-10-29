package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/Jmastelpur/Chirpy/internal/auth"
	"github.com/Jmastelpur/Chirpy/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type Server struct {
	Handler http.Handler
	Addr    string
}

type apiConfig struct {
	fileserverHits atomic.Int32
	database       *database.Queries
	tokenSecret    string
	polkaKey       string
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	// Increment the fileserverHits counter for requests to /app/
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getFileserverHits() int32 {
	return cfg.fileserverHits.Load()
}

func cleanupJsonString(jsonStr string) string {
	// Removes kerfuffle, sharbert, and fornax from the JSON string with ****
	cleaned := strings.ReplaceAll(jsonStr, "kerfuffle", "****")
	cleaned = strings.ReplaceAll(cleaned, "Kerfuffle", "****")
	cleaned = strings.ReplaceAll(cleaned, "sharbert", "****")
	cleaned = strings.ReplaceAll(cleaned, "Sharbert", "****")
	cleaned = strings.ReplaceAll(cleaned, "fornax", "****")
	cleaned = strings.ReplaceAll(cleaned, "Fornax", "****")

	return cleaned
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(fmt.Sprintf(`{"error": "%s"}`, msg)))
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write([]byte(fmt.Sprintf(`%s`, payload)))

}

func main() {

	godotenv.Load()

	fmt.Println("Starting server on :8080")
	cfg := &apiConfig{}

	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v\n", err)
		return
	}
	dbQueries := database.New(db)

	cfg.database = dbQueries

	deletePermission := os.Getenv("PLATFORM")

	cfg.tokenSecret = os.Getenv("SECRET")

	cfg.polkaKey = os.Getenv("POLKA_KEY")

	mux := http.NewServeMux()
	mux.HandleFunc("GET /admin/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "OK"}`))
	})

	mux.HandleFunc("GET /admin/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.getFileserverHits())))
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		type ChirpResponse struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}

		s := r.URL.Query().Get("author_id")
		o := r.URL.Query().Get("sort")
		chirps := []database.Chirp{}

		if o == "desc" {
			if s != "" {
				authorID, err := uuid.Parse(s)
				if err != nil {
					respondWithError(w, http.StatusBadRequest, "Invalid author_id")
					return
				}
				chirps, err = cfg.database.GetChirpsByUserIDDesc(r.Context(), authorID)
				if err != nil {
					respondWithError(w, http.StatusInternalServerError, "Failed to get chirps")
					return
				}
			} else {
				chirps, err = cfg.database.GetChirpsDesc(r.Context())
				if err != nil {
					respondWithError(w, http.StatusInternalServerError, "Failed to get chirps")
					return
				}
			}
		} else {
			if s != "" {
				authorID, err := uuid.Parse(s)
				if err != nil {
					respondWithError(w, http.StatusBadRequest, "Invalid author_id")
					return
				}
				chirps, err = cfg.database.GetChirpsByUserID(r.Context(), authorID)
				if err != nil {
					respondWithError(w, http.StatusInternalServerError, "Failed to get chirps")
					return
				}
			} else {
				chirps, err = cfg.database.GetChirps(r.Context())
				if err != nil {
					respondWithError(w, http.StatusInternalServerError, "Failed to get chirps")
					return
				}
			}
		}

		var chirpResponses []ChirpResponse
		for _, chirp := range chirps {
			chirpResponses = append(chirpResponses, ChirpResponse{
				ID:        chirp.ID,
				CreatedAt: chirp.CreatedAt,
				UpdatedAt: chirp.UpdatedAt,
				Body:      chirp.Body,
				UserID:    chirp.UserID,
			})
		}

		data, err := json.Marshal(chirpResponses)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
			return
		}

		respondWithJSON(w, http.StatusOK, data)

	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		type ChirpRequest struct {
			Body string `json:"body"`
		}
		type Chirp struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Body      string    `json:"body"`
			UserID    uuid.UUID `json:"user_id"`
		}

		var chirpReq ChirpRequest
		err := json.NewDecoder(r.Body).Decode(&chirpReq)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		userID, err := auth.ValidateJWT(bearerToken, cfg.tokenSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		if len(chirpReq.Body) > 140 {
			respondWithError(w, http.StatusBadRequest, "Chirp is too long")
			return
		}
		chirpReq.Body = cleanupJsonString(chirpReq.Body)

		chirps := database.CreateChirpParams{
			Body:   chirpReq.Body,
			UserID: userID,
		}

		chirp, err := cfg.database.CreateChirp(r.Context(), chirps)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create chirp")
			return
		}
		userChirp := Chirp{
			ID:        chirp.ID,
			CreatedAt: chirp.CreatedAt,
			UpdatedAt: chirp.UpdatedAt,
			Body:      chirp.Body,
			UserID:    userID,
		}
		data, err := json.Marshal(userChirp)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Failed to marshal response")
			return
		}
		respondWithJSON(w, http.StatusCreated, data)
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		type UserRequest struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}

		type User struct {
			ID        uuid.UUID `json:"id"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
			Email     string    `json:"email"`
			IsRed     bool      `json:"is_chirpy_red"`
		}

		var userReq UserRequest
		err := json.NewDecoder(r.Body).Decode(&userReq)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		userReq.Password, err = auth.HashPassword(userReq.Password)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		user, err := cfg.database.CreateUser(r.Context(), database.CreateUserParams{
			Email:          userReq.Email,
			HashedPassword: userReq.Password,
		})
		if err != nil {
			fmt.Printf("Failed to create user: %v\n", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		userResp := User{
			ID:        user.ID,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
			Email:     user.Email,
			IsRed:     user.IsRed,
		}

		data, err := json.Marshal(userResp)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
			return
		}

		respondWithJSON(w, http.StatusCreated, data)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		type LoginRequest struct {
			Password string `json:"password"`
			Email    string `json:"email"`
		}
		type User struct {
			ID           uuid.UUID `json:"id"`
			CreatedAt    time.Time `json:"created_at"`
			UpdatedAt    time.Time `json:"updated_at"`
			Email        string    `json:"email"`
			Token        string    `json:"token"`
			RefreshToken string    `json:"refresh_token"`
			IsRed        bool      `json:"is_chirpy_red"`
		}

		var loginReq LoginRequest
		err := json.NewDecoder(r.Body).Decode(&loginReq)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		user, err := cfg.database.GetUserByEmail(r.Context(), loginReq.Email)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Failed to get user by email")
			return
		}

		passwordLogin := auth.CheckPasswordHash(loginReq.Password, user.HashedPassword)
		if !passwordLogin {
			respondWithError(w, http.StatusUnauthorized, "Invalid password")
			return
		}

		token, err := auth.MakeJWT(user.ID, cfg.tokenSecret, time.Duration(1)*time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create JWT")
			return
		}

		refreshToken, err := cfg.database.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
			Token:     auth.MakeRefreshToken(),
			UserID:    user.ID,
			ExpiresAt: time.Now().Add(60 * time.Hour),
		})
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create refresh token")
			return
		}

		userResp := User{
			ID:           user.ID,
			CreatedAt:    user.CreatedAt,
			UpdatedAt:    user.UpdatedAt,
			Email:        user.Email,
			Token:        token,
			RefreshToken: refreshToken.Token,
			IsRed:        user.IsRed,
		}

		data, err := json.Marshal(userResp)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
			return
		}

		respondWithJSON(w, http.StatusOK, data)
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if deletePermission != "dev" {
			respondWithError(w, http.StatusForbidden, "Delete permission not granted")
			return
		}
		err := cfg.database.DeleteAllUsers(r.Context())
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to reset users")
			return
		}
		respondWithJSON(w, http.StatusOK, []byte(`{"message": "All users have been deleted"}`))
	})

	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("AUTHORIZATION", "application/json; charset=utf-8")

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		token, err := cfg.database.GetRefreshToken(r.Context(), bearerToken)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}

		if token.ExpiresAt.Before(time.Now()) || token.RevokedAt.Valid {
			respondWithError(w, http.StatusUnauthorized, "Expired or revoked refresh token")
			return
		}

		newAccessToken, err := auth.MakeJWT(token.UserID, cfg.tokenSecret, time.Hour)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to create new access token")
			return
		}

		type RefreshResponse struct {
			Token string `json:"token"`
		}

		response := RefreshResponse{
			Token: newAccessToken,
		}

		data, err := json.Marshal(response)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
			return
		}

		respondWithJSON(w, http.StatusOK, data)
	})

	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("AUTHORIZATION", "application/json; charset=utf-8")

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}
		err = cfg.database.RevokeRefreshToken(r.Context(), bearerToken)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid refresh token")
			return
		}
		respondWithJSON(w, http.StatusNoContent, []byte(``))
	})

	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("AUTHORIZATION", "application/json; charset=utf-8")

		type WebhookRequest struct {
			Data struct {
				UserID uuid.UUID `json:"user_id"`
			} `json:"data"`
			Event string `json:"event"`
		}

		givenAPIKey, err := auth.GetAPIKey(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}
		if givenAPIKey != cfg.polkaKey {
			respondWithError(w, http.StatusUnauthorized, "Invalid API key")
			return
		}

		var webhookReq WebhookRequest
		err = json.NewDecoder(r.Body).Decode(&webhookReq)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		if webhookReq.Event != "user.upgraded" {
			respondWithJSON(w, http.StatusNoContent, []byte(``))
			return
		}

		_, err = cfg.database.BecomeRedByID(r.Context(), webhookReq.Data.UserID)
		if err != nil {
			fmt.Printf("Failed to update user: %v\n", err)
			respondWithError(w, http.StatusNotFound, "Failed to update user")
			return
		}

		respondWithJSON(w, http.StatusNoContent, []byte(``))
	})

	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("AUTHORIZATION", "application/json; charset=utf-8")

		type UpdateUserRequest struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}

		var updateReq UpdateUserRequest
		err := json.NewDecoder(r.Body).Decode(&updateReq)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		userID, err := auth.ValidateJWT(bearerToken, cfg.tokenSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		updateReq.Password, err = auth.HashPassword(updateReq.Password)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to hash password")
			return
		}

		user, err := cfg.database.UpdateUserEmailandPassword(r.Context(), database.UpdateUserEmailandPasswordParams{
			ID:             userID,
			Email:          updateReq.Email,
			HashedPassword: updateReq.Password,
		})

		if err != nil {
			fmt.Printf("Failed to update user: %v\n", err)
			respondWithError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}

		type User struct {
			Email string `json:"email"`
		}

		userResp := User{
			Email: user.Email,
		}

		data, err := json.Marshal(userResp)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to marshal response")
			return
		}

		respondWithJSON(w, http.StatusOK, data)

	})

	mux.HandleFunc("DELETE /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("AUTHORIZATION", "application/json; charset=utf-8")

		bearerToken, err := auth.GetBearerToken(r.Header)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		userID, err := auth.ValidateJWT(bearerToken, cfg.tokenSecret)
		if err != nil {
			respondWithError(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		idParam := r.PathValue("chirpID")
		chirpID, err := uuid.Parse(idParam)
		if err != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid chirp ID")
			return
		}

		chirp, err := cfg.database.GetChirpByID(r.Context(), chirpID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "Chirp not found")
			return
		}

		if chirp.UserID != userID {
			respondWithError(w, http.StatusForbidden, "You are not authorized to delete this chirp")
			return
		}

		err = cfg.database.DeleteChirp(r.Context(), chirpID)
		if err != nil {
			respondWithError(w, http.StatusInternalServerError, "Failed to delete chirp")
			return
		}

		respondWithJSON(w, http.StatusNoContent, []byte(``))
	})

	mux.Handle("/app/", cfg.middlewareMetricsInc(http.StripPrefix("/app", http.FileServer(http.Dir(".")))))
	server := &Server{
		Handler: mux,
		Addr:    ":8080",
	}

	err = http.ListenAndServe(server.Addr, server.Handler)
	if err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}

}
