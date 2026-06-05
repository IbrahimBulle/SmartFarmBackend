package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	db "github.com/IbrahimBulle/SmartFarm/internal/database"
	"github.com/IbrahimBulle/SmartFarm/internal/middleware"
	"github.com/go-chi/chi/v5"
)

type FarmHandler struct {
	q *db.Queries
}

func NewFarmHandler(q *db.Queries) *FarmHandler {
	return &FarmHandler{q: q}
}

type CreateFarmRequest struct {
	Name      string  `json:"name"`
	Location  string  `json:"location"`
	CropType  string  `json:"crop_type"`
	SizeAcres float64 `json:"size_acres"`
}

func (h *FarmHandler) CreateFarm(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := r.Context().
		Value(middleware.UserIDKey).(int64)

	var req CreateFarmRequest

	if err := json.NewDecoder(r.Body).
		Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	farm, err := h.q.CreateFarm(
		r.Context(),
		db.CreateFarmParams{
			UserID:   userID,
			Name:     req.Name,
			Location: req.Location,
			CropType: req.CropType,
			SizeAcres: sql.NullFloat64{
				Float64: req.SizeAcres,
				Valid:   true,
			},
		},
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(farm)
}

func (h *FarmHandler) ListFarms(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := r.Context().
		Value(middleware.UserIDKey).(int64)

	farms, err := h.q.ListFarmsByUser(
		r.Context(),
		userID,
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(farms)
}

func (h *FarmHandler) GetFarm(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, _ := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)

	farm, err := h.q.GetFarm(
		r.Context(),
		id,
	)

	if err != nil {
		http.Error(w, "farm not found", 404)
		return
	}

	json.NewEncoder(w).Encode(farm)
}

func (h *FarmHandler) UpdateFarm(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID := r.Context().
		Value(middleware.UserIDKey).(int64)
	id, _ := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)

	var req CreateFarmRequest

	if err := json.NewDecoder(r.Body).
		Decode(&req); err != nil {
		http.Error(w, "invalid body", 400)
		return
	}

	farm, err := h.q.UpdateFarm(
		r.Context(),
		db.UpdateFarmParams{
			Name:     req.Name,
			Location: req.Location,
			CropType: req.CropType,
			SizeAcres: sql.NullFloat64{
				Float64: req.SizeAcres,
				Valid:   true,
			},
			ID:     id,
			UserID: userID,
		},
	)

	if err != nil {
		http.Error(w, "farm not found", 404)
		return
	}

	json.NewEncoder(w).Encode(farm)
}

func (h *FarmHandler) DeleteFarm(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, _ := strconv.ParseInt(
		chi.URLParam(r, "id"),
		10,
		64,
	)

	err := h.q.DeleteFarm(
		r.Context(),
		id,
	)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
