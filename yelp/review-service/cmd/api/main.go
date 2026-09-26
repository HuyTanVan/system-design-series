package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/HuyTanVan/yelp-clone/reviews-service/internal/auth"
	"github.com/HuyTanVan/yelp-clone/reviews-service/internal/db"
	"github.com/HuyTanVan/yelp-clone/reviews-service/internal/handler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dsn := os.Getenv("REVIEWS_DB_DSN")
	if dsn == "" {
		dsn = "postgres://yelp:yelp@localhost:5432/reviews_db"
	}

	pool, err := db.NewPool(context.Background(), dsn, 2, 100)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	rh := &handler.ReviewHandler{DB: pool}
	ah := &handler.AuthHandler{DB: pool}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Handle("/metrics", promhttp.Handler())

	r.Post("/auth/signup", ah.Signup)
	r.Post("/auth/login", ah.Login)

	r.Get("/businesses/{businessID}/reviews", rh.List)
	r.With(auth.RequireAuth).Post("/reviews", rh.Create)

	log.Println("reviews-service listening on :8082")
	if err := http.ListenAndServe(":8082", r); err != nil {
		log.Fatal(err)
	}
}
