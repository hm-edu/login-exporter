package main

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

func probeHandler(w http.ResponseWriter, r *http.Request, configs map[string]SingleLoginConfig) {

	// Logs all the connections
	logger.WithFields(
		log.Fields{
			"subsytem":     "probe_handler",
			"part":         "connection_info",
			"user_address": r.RemoteAddr,
			"server_host":  r.Host,
			"user_agent":   r.UserAgent(),
		}).Info("This connection was established")

	// Extract the target from the url
	target := r.URL.Query().Get("target")
	if target == "" {
		logger.WithFields(
			log.Fields{
				"subsystem": "probe_handler",
				"part":      "target_check",
			}).Error("The target is not given")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// Find the target in the configuration
	targetConfig, ok := configs[target]
	if !ok {
		logger.WithFields(
			log.Fields{
				"subsystem": "probe_handler",
				"part":      "target_config_check",
			}).Error("The given target does not have configuration")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	loginType := targetConfig.LoginType
	// Get the status and elapsed time for the tests
	status, elapsed, elapsedTotal := getStatus(targetConfig)
	statusValue := 0
	if status {
		statusValue = 1
	}
	var statusMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "login_status",
			Help: "Shows the status of the given target 0 for failure 1 for success",
		},
		[]string{"target", "login_type"},
	)
	var elapsedTotalMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "login_total_elapsed_seconds",
			Help: "Shows how long it took the get the data in seconds",
		},
		[]string{"target", "login_type"},
	)
	var elapsedMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "login_elapsed_seconds",
			Help: "Shows how long it to login in seconds",
		},
		[]string{"target", "login_type"},
	)
	registry := prometheus.NewRegistry()
	registry.MustRegister(statusMetric)
	registry.MustRegister(elapsedMetric)
	registry.MustRegister(elapsedTotalMetric)
	statusMetric.WithLabelValues(target, loginType).Set(float64(statusValue))
	elapsedMetric.WithLabelValues(target, loginType).Set(elapsed)
	elapsedTotalMetric.WithLabelValues(target, loginType).Set(elapsedTotal)
	h := promhttp.HandlerFor(registry, promhttp.HandlerOpts{})
	h.ServeHTTP(w, r)
}

func main() {
	loginConfig := readConfig(configFilePath)

	configMap := make(map[string]SingleLoginConfig, len(loginConfig.Configs))
	for _, c := range loginConfig.Configs {
		configMap[c.Target] = c
	}

	http.HandleFunc("/probe", func(w http.ResponseWriter, r *http.Request) {
		probeHandler(w, r, configMap)
	})

	logger.WithFields(
		log.Fields{
			"subsystem": "main",
			"part":      "port_setting",
		}).Info("Started Listening on " + listenIp + ":" + strconv.Itoa(listenPort))

	logger.Fatal(http.ListenAndServe(listenIp+":"+strconv.Itoa(listenPort), nil))
}
