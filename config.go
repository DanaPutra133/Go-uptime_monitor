package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Service struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Config struct {
	ApiKey      string
	TargetEmail string
	Port        string
	Services    []Service
}

func LoadConfig() *Config {
	_ = godotenv.Load()

	apiKey := os.Getenv("RESEND_API_KEY")
	targetEmail := os.Getenv("NOTIFICATION_EMAIL")
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	if apiKey == "" || targetEmail == "" {
		log.Fatal("Konfigurasi RESEND_API_KEY atau NOTIFICATION_EMAIL di .env belum lengkap!")
	}

	fileData, err := os.ReadFile("services.json")
	if err != nil {
		log.Fatalf("Gagal membaca file services.json: %v", err)
	}

	var services []Service
	err = json.Unmarshal(fileData, &services)
	if err != nil {
		log.Fatalf("Gagal parsing file services.json: %v", err)
	}

	return &Config{
		ApiKey:      apiKey,
		TargetEmail: targetEmail,
		Port:        port,
		Services:    services,
	}
}