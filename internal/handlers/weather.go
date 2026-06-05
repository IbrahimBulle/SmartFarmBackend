package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/IbrahimBulle/SmartFarm/internal/middleware"
)

const defaultWeatherAIBaseURL = "https://api.weather-ai.co/v1"

type WeatherHandler struct {
	db              *sql.DB
	client          *http.Client
	upstreamBaseURL string
}

func NewWeatherHandler(dbConn *sql.DB) *WeatherHandler {
	baseURL := strings.TrimRight(os.Getenv("WEATHER_AI_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = defaultWeatherAIBaseURL
	}

	return &WeatherHandler{
		db: dbConn,
		client: &http.Client{
			Timeout: 2 * time.Minute,
		},
		upstreamBaseURL: baseURL,
	}
}

type weatherAPIKeyRequest struct {
	APIKey string `json:"api_key"`
}

type weatherAPIKeyStatus struct {
	Configured bool `json:"configured"`
}

func (h *WeatherHandler) KeyStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}

	var exists int
	err := h.db.QueryRowContext(
		r.Context(),
		`SELECT EXISTS(SELECT 1 FROM weather_api_keys WHERE user_id = ?)`,
		userID,
	).Scan(&exists)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not check WeatherAI key")
		return
	}

	writeJSON(w, http.StatusOK, weatherAPIKeyStatus{Configured: exists == 1})
}

func (h *WeatherHandler) SaveKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}

	var req weatherAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}

	apiKey := normalizeWeatherAPIKey(req.APIKey)
	if apiKey == "" {
		writeJSONError(w, http.StatusBadRequest, "WeatherAI API key is required")
		return
	}

	if err := h.validateWeatherAPIKey(r.Context(), apiKey); err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}

	_, err := h.db.ExecContext(
		r.Context(),
		`INSERT INTO weather_api_keys (user_id, api_key)
		 VALUES (?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		     api_key = excluded.api_key,
		     updated_at = CURRENT_TIMESTAMP`,
		userID,
		apiKey,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not save WeatherAI key")
		return
	}

	writeJSON(w, http.StatusOK, weatherAPIKeyStatus{Configured: true})
}

func (h *WeatherHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}

	_, err := h.db.ExecContext(
		r.Context(),
		`DELETE FROM weather_api_keys WHERE user_id = ?`,
		userID,
	)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not delete WeatherAI key")
		return
	}

	writeJSON(w, http.StatusOK, weatherAPIKeyStatus{Configured: false})
}

func (h *WeatherHandler) GetWeather(w http.ResponseWriter, r *http.Request) {
	h.proxyWeatherAI(w, r, "/weather")
}

func (h *WeatherHandler) GetUsage(w http.ResponseWriter, r *http.Request) {
	h.proxyWeatherAI(w, r, "/usage")
}

func (h *WeatherHandler) GetTreeQuota(w http.ResponseWriter, r *http.Request) {
	h.proxyWeatherAI(w, r, "/trees/quota")
}

func (h *WeatherHandler) AnalyzeTrees(w http.ResponseWriter, r *http.Request) {
	h.proxyWeatherAI(w, r, "/trees/analyze")
}

func (h *WeatherHandler) proxyWeatherAI(w http.ResponseWriter, r *http.Request, upstreamPath string) {
	userID, ok := userIDFromRequest(w, r)
	if !ok {
		return
	}

	apiKey, err := h.getWeatherAPIKey(r.Context(), userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeJSONError(w, http.StatusPreconditionRequired, "Add your WeatherAI API key before using weather services")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not load WeatherAI key")
		return
	}

	target, err := h.upstreamURL(upstreamPath, r.URL.RawQuery)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "invalid WeatherAI upstream URL")
		return
	}

	upstreamReq, err := http.NewRequestWithContext(r.Context(), r.Method, target, r.Body)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "could not create WeatherAI request")
		return
	}
	upstreamReq.ContentLength = r.ContentLength
	copyForwardHeader(upstreamReq.Header, r.Header, "Accept")
	copyForwardHeader(upstreamReq.Header, r.Header, "Content-Type")
	upstreamReq.Header.Set("Authorization", "Bearer "+apiKey)

	upstreamResp, err := h.client.Do(upstreamReq)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "WeatherAI request failed")
		return
	}
	defer upstreamResp.Body.Close()

	copyForwardHeader(w.Header(), upstreamResp.Header, "Content-Type")
	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json")
	}

	w.WriteHeader(upstreamResp.StatusCode)
	_, _ = io.Copy(w, upstreamResp.Body)
}

func (h *WeatherHandler) getWeatherAPIKey(ctx context.Context, userID int64) (string, error) {
	var apiKey string
	err := h.db.QueryRowContext(
		ctx,
		`SELECT api_key FROM weather_api_keys WHERE user_id = ? LIMIT 1`,
		userID,
	).Scan(&apiKey)
	return apiKey, err
}

func (h *WeatherHandler) validateWeatherAPIKey(ctx context.Context, apiKey string) error {
	target, err := h.upstreamURL("/usage", "")
	if err != nil {
		return fmt.Errorf("invalid WeatherAI upstream URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return fmt.Errorf("could not validate WeatherAI key")
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("could not reach WeatherAI to validate this key")
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("WeatherAI rejected this key: %s", upstreamErrorMessage(body, resp.Status))
	}

	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
	return nil
}

func (h *WeatherHandler) upstreamURL(upstreamPath string, rawQuery string) (string, error) {
	parsedURL, err := url.Parse(h.upstreamBaseURL + upstreamPath)
	if err != nil {
		return "", err
	}
	parsedURL.RawQuery = rawQuery
	return parsedURL.String(), nil
}

func normalizeWeatherAPIKey(value string) string {
	key := strings.TrimSpace(value)
	key = strings.TrimPrefix(key, "Bearer ")
	key = strings.TrimPrefix(key, "bearer ")
	return strings.TrimSpace(key)
}

func upstreamErrorMessage(body []byte, fallback string) string {
	var payload struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal(body, &payload); err == nil {
		if strings.TrimSpace(payload.Message) != "" {
			return strings.TrimSpace(payload.Message)
		}
		if strings.TrimSpace(payload.Error) != "" {
			return strings.TrimSpace(payload.Error)
		}
	}

	if text := strings.TrimSpace(string(body)); text != "" {
		return text
	}

	return fallback
}

func userIDFromRequest(w http.ResponseWriter, r *http.Request) (int64, bool) {
	userID, ok := r.Context().Value(middleware.UserIDKey).(int64)
	if !ok {
		writeJSONError(w, http.StatusUnauthorized, "missing authenticated user")
		return 0, false
	}

	return userID, true
}

func copyForwardHeader(dst http.Header, src http.Header, key string) {
	if value := src.Get(key); value != "" {
		dst.Set(key, value)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}
