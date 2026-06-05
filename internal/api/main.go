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

	authHandler := handlers.NewAuthHandler(queries)
	farmHandler := handlers.NewFarmHandler(queries)

	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://sunny-gecko-29a6bf.netlify.app"},
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
