package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/resend/resend-go/v2"
)

type MonitorWorker struct {
	cfg          *Config
	resendClient *resend.Client
	alertedStates map[string]bool
	mu            sync.Mutex
}

func NewMonitorWorker(cfg *Config) *MonitorWorker {
	return &MonitorWorker{
		cfg:           cfg,
		resendClient:  resend.NewClient(cfg.ApiKey),
		alertedStates: make(map[string]bool),
	}
}

func (w *MonitorWorker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go w.CheckServices()

	for range ticker.C {
		go w.CheckServices()
	}
}

func (w *MonitorWorker) CheckServices() {
	fmt.Printf("[%s] Memulai pemeriksaan berkala...\n", time.Now().Format("2006-01-02 15:04:05"))
	httpClient := &http.Client{Timeout: 10 * time.Second}

	for _, service := range w.cfg.Services {
		resp, err := httpClient.Get(service.URL)
		
		w.mu.Lock()
		isAlreadyAlerted := w.alertedStates[service.URL]

		if err != nil {
			fmt.Printf("[DOWN] %s - Error: %v\n", service.Name, err)
			if !isAlreadyAlerted {
				w.SendAlert(service, "DOWN/TIMEOUT", err.Error())
				w.alertedStates[service.URL] = true // Tandai sudah dikirim alert
			} else {
				fmt.Printf("[SKIP ALERT] %s masih down, email tidak dikirim ulang (mencegah spam).\n", service.Name)
			}
			w.mu.Unlock()
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			fmt.Printf("[ERROR] %s - Status: %d\n", service.Name, resp.StatusCode)
			if !isAlreadyAlerted {
				w.SendAlert(service, fmt.Sprintf("%d", resp.StatusCode), "Response status bukan 200")
				w.alertedStates[service.URL] = true
			} else {
				fmt.Printf("[SKIP ALERT] %s masih error, email tidak dikirim ulang.\n", service.Name)
			}
		} else {
			fmt.Printf("[AMAN] %s - Status: %d\n", service.Name, resp.StatusCode)
			if isAlreadyAlerted {
				fmt.Printf("[RECOVERED] %s sudah kembali normal!\n", service.Name)
				w.alertedStates[service.URL] = false
			}
		}
		w.mu.Unlock()
	}
}

func (w *MonitorWorker) SendAlert(service Service, status string, details string) {
	subject := fmt.Sprintf("⚠️ ALERT: %s Bermasalah!", service.Name)
	htmlContent := fmt.Sprintf(`
		<h2>Peringatan Sistem Monitoring</h2>
		<p>Service berikut mengalami gangguan:</p>
		<ul>
			<li><strong>Nama Service:</strong> %s</li>
			<li><strong>URL:</strong> <a href="%s" target="_blank">%s</a></li>
			<li><strong>Status Respon:</strong> <span style="color: red; font-weight: bold;">%s</span></li>
			<li><strong>Detail:</strong> %s</li>
			<li><strong>Waktu Terdeteksi:</strong> %s</li>
		</ul>
	`, service.Name, service.URL, service.URL, status, details, time.Now().Format("2006-01-02 15:04:05"))

	params := &resend.SendEmailRequest{
		From:    "Monitor Bot <onboarding@resend.dev>",
		To:      []string{w.cfg.TargetEmail},
		Subject: subject,
		Html:    htmlContent,
	}

	_, err := w.resendClient.Emails.Send(params)
	if err != nil {
		log.Printf("Gagal mengirim email alert untuk %s: %v", service.Name, err)
	} else {
		log.Printf("[ALERT TERKIRIM] Email pemberitahuan untuk %s berhasil dikirim.", service.Name)
	}
}