package main

import (
	"log"
	"net/http"
	"os"

	"github.com/HuyTanVan/yelp-clone/search-service/internal/handler"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	esURL := os.Getenv("ELASTICSEARCH_URL")
	if esURL == "" {
		esURL = "http://yelp-elasticsearch:9200"
	}

	// 1. init new es client
	esClient, err := elasticsearch.NewClient(elasticsearch.Config{Addresses: []string{esURL}})
	if err != nil {
		log.Fatalf("failed to create es client: %v", err)
	}
	// 2. test if es is available
	res, err := esClient.Ping()
	if err != nil {
		log.Fatalf("Elasticsearch unavailable: %v", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		log.Fatalf("Elasticsearch returned error: %s", res.Status())
	}

	log.Println("Elasticsearch is healthy")
	sh := &handler.SearchHandler{ES: esClient}

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"}, //
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Get("/search", sh.Search)
	r.Handle("/metrics", promhttp.Handler())
	log.Println("search-service listening on :8083")
	if err := http.ListenAndServe(":8083", r); err != nil {
		log.Fatal(err)
	}
}
