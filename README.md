# build
```go build -o uptime-monitor main.go config.go worker.go handler.go```

# run
```pm2 start ./uptime-monitor --name "uptime-monitor"```
