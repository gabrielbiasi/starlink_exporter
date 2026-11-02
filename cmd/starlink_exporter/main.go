package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/danopstech/starlink_exporter/internal/exporter"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

const metricsPath = "/metrics"

func main() {
	port := flag.String("port", "9817", "listening port to expose metrics on")
	address := flag.String("address", exporter.DishAddress, "IP address and port to reach dish")
	flag.Parse()

	ex, err := exporter.New(*address)
	if err != nil {
		log.Fatalf("could not start exporter: %s", err.Error())
	}
	defer ex.Conn.Close()

	reg := prometheus.NewRegistry()
	reg.MustRegister(ex)

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
             <head><title>Starlink Exporter</title></head>
             <body>
             <h1>Starlink Exporter</h1>
             <p><a href='` + metricsPath + `'>Metrics</a></p>
             <p><a href='/health'>Health (gRPC connection state to Starlink dish)</a></p>
             </body>
             </html>`))
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		switch ex.Conn.GetState() {
		case 0, 2:
			w.WriteHeader(http.StatusOK)
		case 1, 3:
			w.WriteHeader(http.StatusServiceUnavailable)
		case 4:
			w.WriteHeader(http.StatusInternalServerError)
		}
		_, _ = fmt.Fprintf(w, "%s\n", ex.Conn.GetState())
	})

	http.Handle(metricsPath, promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":"+*port, nil))
}
