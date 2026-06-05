package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"

	db "github.com/IbrahimBulle/SmartFarm/internal/database"
	"github.com/IbrahimBulle/SmartFarm/internal/handlers"
	"github.com/IbrahimBulle/SmartFarm/internal/middleware"
)

func main() {
	_ = godotenv.Load()

	dbPath := os.Getenv("DATABASE_URL")
	if dbPath == "" {
		dbPath = os.Getenv("DB_PATH")
	}
	if dbPath == "" {
		dbPath = "./farm.db"
	}

	dbConn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	defer dbConn.Close()

	queries := db.New(dbConn)
	if err := ensureWeatherAPIKeyTable(dbConn); err != nil {
		log.Fatal(err)
	}

	authHandler := handlers.NewAuthHandler(queries)
	farmHandler := handlers.NewFarmHandler(queries)
	weatherHandler := handlers.NewWeatherHandler(dbConn)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://sunny-gecko-29a6bf.netlify.app", "http://127.0.0.1:8080/api/farms", "*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "very healthy")
	})
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Post("/logout", authHandler.Logout)

		r.Post("/farms", farmHandler.CreateFarm)
		r.Get("/farms", farmHandler.ListFarms)
		r.Get("/farms/{id}", farmHandler.GetFarm)
		r.Put("/farms/{id}", farmHandler.UpdateFarm)
		r.Delete("/farms/{id}", farmHandler.DeleteFarm)

		r.Get("/weather/key", weatherHandler.KeyStatus)
		r.Post("/weather/key", weatherHandler.SaveKey)
		r.Delete("/weather/key", weatherHandler.DeleteKey)
		r.Get("/weather", weatherHandler.GetWeather)
		r.Get("/weather/usage", weatherHandler.GetUsage)
		r.Get("/weather/trees/quota", weatherHandler.GetTreeQuota)
		r.Post("/weather/trees/analyze", weatherHandler.AnalyzeTrees)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	go func() {
		_, _ = http.Get("https://smartfarmbackend-ypqi.onrender.com/health")
		time.Sleep(5 * time.Minute)
	}()
	log.Printf("server running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func ensureWeatherAPIKeyTable(dbConn *sql.DB) error {
	_, err := dbConn.Exec(`
CREATE TABLE IF NOT EXISTS weather_api_keys (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    api_key TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_weather_api_keys_user_id ON weather_api_keys(user_id);
`)
	return err
}
