package main

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_demo_http_requests_total",
			Help: "Total HTTP requests received by the demo service.",
		},
		[]string{"path", "method", "code"},
	)

	heartbeatTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "go_demo_heartbeat_total",
			Help: "Number of heartbeat ticks emitted by the demo service.",
		},
	)

	processUp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_demo_process_up",
			Help: "Whether the demo process is considered up.",
		},
	)

	httpRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_demo_http_request_duration_seconds",
			Help:    "HTTP request duration of the demo service in seconds.",
			Buckets: []float64{0.01, 0.05, 0.1, 0.3, 0.5, 1, 2, 5, 10},
		},
		[]string{"path", "method", "code"},
	)

	onlineUsers = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "go_demo_online_users",
			Help: "Simulated online users, oscillating between 1 and 100.",
		},
	)

	queueDepth = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "go_demo_queue_depth",
			Help: "Simulated queue depth for different internal queues.",
		},
		[]string{"queue"},
	)

	backgroundJobsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "go_demo_background_jobs_total",
			Help: "Simulated background jobs processed by the demo service.",
		},
		[]string{"job", "result"},
	)

	backgroundJobDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "go_demo_background_job_duration_seconds",
			Help:    "Simulated background job duration in seconds.",
			Buckets: []float64{0.05, 0.1, 0.2, 0.3, 0.5, 1, 2, 5},
		},
		[]string{"job"},
	)
)

type app struct {
	startedAt      time.Time
	requestCounter atomic.Uint64
	logger         *log.Logger
}

type logEntry struct {
	Level     string `json:"level"`
	Message   string `json:"message"`
	Service   string `json:"service"`
	RequestID string `json:"request_id,omitempty"`
	Path      string `json:"path,omitempty"`
	Method    string `json:"method,omitempty"`
	Status    int    `json:"status,omitempty"`
	UptimeSec int64  `json:"uptime_sec,omitempty"`
	Timestamp string `json:"ts"`
}

func main() {
	prometheus.MustRegister(
		httpRequestsTotal,
		heartbeatTotal,
		processUp,
		httpRequestDurationSeconds,
		onlineUsers,
		queueDepth,
		backgroundJobsTotal,
		backgroundJobDurationSeconds,
	)

	logPath := os.Getenv("APP_LOG_PATH")
	if logPath == "" {
		logPath = "/app/logs/go-monitor-demo.log"
	}

	logger, logFile, err := newLogger(logPath)
	if err != nil {
		log.Fatalf("init logger failed: %v", err)
	}
	defer logFile.Close()

	application := &app{
		startedAt: time.Now(),
		logger:    logger,
	}

	processUp.Set(1)

	appPort := envOrDefault("APP_PORT", "18080")
	metricsPort := envOrDefault("METRICS_PORT", "12112")

	appMux := http.NewServeMux()
	appMux.HandleFunc("/", application.handleRoot)
	appMux.HandleFunc("/healthz", application.handleHealth)
	appMux.HandleFunc("/work", application.handleWork)

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())

	appServer := &http.Server{
		Addr:              ":" + appPort,
		Handler:           application.withMetrics(appMux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	metricsServer := &http.Server{
		Addr:              ":" + metricsPort,
		Handler:           metricsMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go application.heartbeatLoop()
	go application.demoMetricsLoop()
	go func() {
		application.writeLog("info", "application server starting", "", "", "", 0)
		if err := appServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.writeLog("error", "application server failed", "", "", "", 500)
			log.Fatalf("app server failed: %v", err)
		}
	}()

	go func() {
		application.writeLog("info", "metrics server starting", "", "", "", 0)
		if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			application.writeLog("error", "metrics server failed", "", "", "", 500)
			log.Fatalf("metrics server failed: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	processUp.Set(0)
	application.writeLog("info", "shutdown signal received", "", "", "", 0)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_ = appServer.Shutdown(ctx)
	_ = metricsServer.Shutdown(ctx)
}

func newLogger(logPath string) (*log.Logger, *os.File, error) {
	if err := os.MkdirAll(filepath.Dir(logPath), 0o755); err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, nil, err
	}

	writer := io.MultiWriter(os.Stdout, file)

	return log.New(writer, "", 0), file, nil
}

func (a *app) handleRoot(w http.ResponseWriter, r *http.Request) {
	count := a.requestCounter.Add(1)

	response := map[string]any{
		"message":      "go-monitor-demo is running",
		"requestCount": count,
		"uptimeSec":    int64(time.Since(a.startedAt).Seconds()),
		"time":         time.Now().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *app) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := map[string]any{
		"status":    "ok",
		"service":   "go-monitor-demo",
		"uptimeSec": int64(time.Since(a.startedAt).Seconds()),
		"time":      time.Now().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, response)
}

func (a *app) handleWork(w http.ResponseWriter, r *http.Request) {
	delayMS := 300
	if raw := r.URL.Query().Get("delay_ms"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err == nil && value >= 0 && value <= 10000 {
			delayMS = value
		}
	}

	status := http.StatusOK
	if raw := r.URL.Query().Get("status"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err == nil && value >= 100 && value <= 599 {
			status = value
		}
	}

	time.Sleep(time.Duration(delayMS) * time.Millisecond)

	response := map[string]any{
		"message":   "work finished",
		"delayMs":   delayMS,
		"status":    status,
		"uptimeSec": int64(time.Since(a.startedAt).Seconds()),
		"time":      time.Now().Format(time.RFC3339),
	}

	writeJSON(w, status, response)
}

func (a *app) withMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next.ServeHTTP(recorder, r)

		codeLabel := strconv.Itoa(recorder.statusCode)
		httpRequestsTotal.WithLabelValues(r.URL.Path, r.Method, codeLabel).Inc()
		httpRequestDurationSeconds.WithLabelValues(r.URL.Path, r.Method, codeLabel).Observe(time.Since(startedAt).Seconds())
		a.writeLog("info", "http request processed", requestID(r), r.URL.Path, r.Method, recorder.statusCode)
	})
}

func (a *app) heartbeatLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		heartbeatTotal.Inc()
		a.writeLog("info", "heartbeat tick", "", "", "", 0)
	}
}

func (a *app) demoMetricsLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	users := 1
	direction := 1
	tick := 0

	a.writeLog("info", "demo metrics loop started", "", "", "", 0)

	for range ticker.C {
		tick++

		onlineUsers.Set(float64(users))

		emailDepth := 5 + (users % 20)
		reportDepth := 8 + ((100 - users) % 25)
		queueDepth.WithLabelValues("email").Set(float64(emailDepth))
		queueDepth.WithLabelValues("report").Set(float64(reportDepth))

		syncSuccess := float64(users/15 + 1)
		reportSuccess := float64((100-users)/20 + 1)
		backgroundJobsTotal.WithLabelValues("sync", "success").Add(syncSuccess)
		backgroundJobsTotal.WithLabelValues("report", "success").Add(reportSuccess)

		if tick%7 == 0 {
			backgroundJobsTotal.WithLabelValues("sync", "failed").Inc()
		}
		if tick%11 == 0 {
			backgroundJobsTotal.WithLabelValues("report", "failed").Inc()
		}

		syncDuration := 0.05 + float64(users%25)/100
		reportDuration := 0.20 + float64((users+tick)%30)/100
		backgroundJobDurationSeconds.WithLabelValues("sync").Observe(syncDuration)
		backgroundJobDurationSeconds.WithLabelValues("report").Observe(reportDuration)

		if users == 100 {
			direction = -1
		} else if users == 1 {
			direction = 1
		}
		users += direction
	}
}

func (a *app) writeLog(level, message, reqID, path, method string, status int) {
	entry := logEntry{
		Level:     level,
		Message:   message,
		Service:   "go-monitor-demo",
		RequestID: reqID,
		Path:      path,
		Method:    method,
		Status:    status,
		UptimeSec: int64(time.Since(a.startedAt).Seconds()),
		Timestamp: time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	a.logger.Println(string(data))
}

func requestID(r *http.Request) string {
	reqID := r.Header.Get("X-Request-Id")
	if reqID != "" {
		return reqID
	}
	return time.Now().Format("20060102150405.000000000")
}

func writeJSON(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(data)
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}
