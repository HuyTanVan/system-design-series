package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"

	"github.com/HuyTanVan/yelp-clone/internal/db"
	"github.com/HuyTanVan/yelp-clone/internal/handler"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	dsn := os.Getenv("BUSINESSES_DB_DSN")
	if dsn == "" {
		// dsn = "postgres://yelp:yelp@yelp-postgres:5432/businesses_db"
		dsn = "postgres://yelp:yelp@localhost:5432/businesses_db"
	}

	pool, err := db.NewPool(context.Background(), dsn)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer pool.Close()

	bh := &handler.BusinessHandler{DB: pool}
	sh := &handler.SearchHandler{DB: pool} // replaced by Elasticsearch
	rh := &handler.RatingHandler{DB: pool}

	synch := &handler.BusinessToESHandler{DB: pool}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, //
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	r.Handle("/metrics", promhttp.Handler())

	r.Get("/test", bh.TestGetByID)
	r.Get("/businesses/{id}", bh.GetByID)  // view business details // mock data: Pns2l4eNsfO8kk83dixA6A
	r.Get("/businesses/search", sh.Search) // search businesses nearby - replaced by Elasticsearch

	r.Patch("/businesses/{id}/rating", rh.UpdateRating)
	// Test bussineses sync to Elasticsearch
	r.Post("/sync", synch.CreateBusiness)
	r.Patch("/sync/{id}", synch.UpdateBusiness)
	r.Delete("/sync/{id}", synch.DeleteBusiness)
	r.Get("/sync/{id}/outbox", synch.GetOutboxStatus)

	log.Println("business-service listening on :8081")
	if err := http.ListenAndServe(":8081", r); err != nil {
		log.Fatal(err)
	}
}
