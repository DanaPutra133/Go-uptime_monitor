package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func main() {
	cfg := LoadConfig()

	worker := NewMonitorWorker(cfg)
	go worker.Start(5 * time.Minute)

	handler := NewServerHandler(cfg)
	router := handler.SetupRouter()

	fmt.Printf("Server berjalan di port %s...\n", cfg.Port)
	log.Fatal(http.ListenAndServe(":"+cfg.Port, router))
}
