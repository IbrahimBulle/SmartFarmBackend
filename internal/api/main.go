package main

import (
	"database/sql"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
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

	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth)

		r.Post("/logout", authHandler.Logout)

		r.Post("/farms", farmHandler.CreateFarm)
		r.Get("/farms", farmHandler.ListFarms)
		r.Get("/farms/{id}", farmHandler.GetFarm)
		r.Delete("/farms/{id}", farmHandler.DeleteFarm)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("server running on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
