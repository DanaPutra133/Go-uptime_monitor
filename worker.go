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
	cfg           *Config
	resendClient  *resend.Client
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


func getTimeWIB() string {
	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
			loc = time.FixedZone("WIB", 7*3600)
	}
	return time.Now().In(loc).Format("2006-01-02 15:04:05 WIB")
}

func (w *MonitorWorker) Start(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go w.CheckServices()

	for range ticker.C {
		w.CheckServices()
	}
}

func (w *MonitorWorker) CheckServices() {
	fmt.Printf("[%s] Memulai pemeriksaan berkala...\n", getTimeWIB())
	
	httpClient := &http.Client{Timeout: 15 * time.Second}
	var wg sync.WaitGroup
	for _, service := range w.cfg.Services {
		wg.Add(1)
		go func(svc Service) {
			defer wg.Done()
			w.checkSingleService(httpClient, svc)
		}(service)
	}
	
	wg.Wait()
}

func (w *MonitorWorker) checkSingleService(httpClient *http.Client, service Service) {

	req, err := http.NewRequest("GET", service.URL, nil)
	if err != nil {
		log.Printf("Gagal membuat request untuk %s: %v", service.URL, err)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) MonitorBot/1.0")

	resp, err := httpClient.Do(req)


	w.mu.Lock()
	isAlreadyAlerted := w.alertedStates[service.URL]
	w.mu.Unlock() 
	if err != nil {
		fmt.Printf("[DOWN] %s - Error: %v\n", service.Name, err)
		if !isAlreadyAlerted {
	
			w.mu.Lock()
			w.alertedStates[service.URL] = true
			w.mu.Unlock()


			w.SendAlert(service, "DOWN/TIMEOUT", err.Error())
		} else {
			fmt.Printf("[SKIP ALERT] %s masih down, email tidak dikirim ulang.\n", service.Name)
		}
		return
	}
	

	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("[ERROR] %s - Status: %d\n", service.Name, resp.StatusCode)
		if !isAlreadyAlerted {
			w.mu.Lock()
			w.alertedStates[service.URL] = true
			w.mu.Unlock()

			w.SendAlert(service, fmt.Sprintf("%d", resp.StatusCode), "Response status bukan 200")
		} else {
			fmt.Printf("[SKIP ALERT] %s masih error, email tidak dikirim ulang.\n", service.Name)
		}
	} else {
		fmt.Printf("[AMAN] %s - Status: %d\n", service.Name, resp.StatusCode)
		if isAlreadyAlerted {
			fmt.Printf("[RECOVERED] %s sudah kembali normal!\n", service.Name)
			w.mu.Lock()
			w.alertedStates[service.URL] = false
			w.mu.Unlock()
		}
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
	`, service.Name, service.URL, service.URL, status, details, getTimeWIB())

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
