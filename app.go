package main

import (
	"net/http"
	"os"

	"github.com/Financial-Times/base-ft-rw-app-go"
	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/http-handlers-go"
	"github.com/Financial-Times/public-organisation-api/organisation"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
	"github.com/jawher/mow.cli"
	"github.com/jmcvetta/neoism"
	"github.com/rcrowley/go-metrics"
)

func main() {
	log.SetLevel(log.InfoLevel)
	log.Infof("Application started with args %s", os.Args)

	app := cli.App("public-organisation-api-neo4j", "A public RESTful API for accessing organisation in neo4j")
	neoURL := app.StringOpt("neo-url", "http://localhost:7474/db/data", "neo4j endpoint URL")
	//neoURL := app.StringOpt("neo-url", "http://ftper58827-law1b-eu-t:8080/db/data", "neo4j endpoint URL")
	port := app.StringOpt("port", "8080", "Port to listen on")
	graphiteTCPAddress := app.StringOpt("graphiteTCPAddress", "",
		"Graphite TCP address, e.g. graphite.ft.com:2003. Leave as default if you do NOT want to output to graphite (e.g. if running locally)")
	graphitePrefix := app.StringOpt("graphitePrefix", "",
		"Prefix to use. Should start with content, include the environment, and the host name. e.g. content.test.public.organisation.api.ftaps59382-law1a-eu-t")
	logMetrics := app.BoolOpt("logMetrics", false, "Whether to log metrics. Set to true if running locally and you want metrics output")

	app.Action = func() {
		baseftrwapp.OutputMetricsIfRequired(*graphiteTCPAddress, *graphitePrefix, *logMetrics)
		log.Infof("public-organisation-api will listen on port: %s, connecting to: %s", *port, *neoURL)
		runServer(*neoURL, *port)
	}
	app.Run(os.Args)
}

func runServer(neoURL string, port string) {
	db, err := neoism.Connect(neoURL)
	db.Session.Client = &http.Client{Transport: &http.Transport{MaxIdleConnsPerHost: 100}}
	if err != nil {
		log.Fatalf("Error connecting to neo4j %s", err)
	}
	organisation.OrganisationDriver = organisation.NewCypherDriver(db)

	r := mux.NewRouter()

	// Healthchecks and standards first
	r.HandleFunc("/__health", v1a.Handler("OrganisationReadWriteNeo4j Healthchecks",
		"Checks for accessing neo4j", organisation.HealthCheck()))
	r.HandleFunc("/ping", organisation.Ping)
	r.HandleFunc("/__ping", organisation.Ping)

	// Then API specific ones:
	r.HandleFunc("/organisations/{uuid}", organisation.GetOrganisation).Methods("GET")

	if err := http.ListenAndServe(":"+port,
		httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry,
			httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), r))); err != nil {
		log.Fatalf("Unable to start server: %v", err)
	}
}
