package main

import (
	"context"
	"errors"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"google.golang.org/api/idtoken"
)

var instituteCodes = []string{
	"1",
	"10",
	"11",
	"12",
	"14",
	"16",
	"17",
	"18",
	"2",
	"21",
	"22",
	"23",
	"25",
	"27",
	"3",
	"39",
	"41",
	"42",
	"43",
	"44",
	"45",
	"46",
	"47",
	"48",
	"5",
	"55",
	"58",
	"59",
	"6",
	"60",
	"66",
	"7",
	"74",
	"75",
	"76",
	"8",
	"81",
	"86",
	"87",
	"88",
	"89",
	"9",
	"90",
	"97",
	"98",
	"99",
}

var workerEndpoint string
var client *http.Client = http.DefaultClient

func init() {
	if level, ok := os.LookupEnv("LOG_LEVEL"); ok {
		switch level {
		case "info":
			log.SetLevel(log.InfoLevel)
		case "warn":
			log.SetLevel(log.WarnLevel)
		case "error":
			log.SetLevel(log.ErrorLevel)
		default:
			log.SetLevel(log.DebugLevel)
		}
	}

	if worker, ok := os.LookupEnv("WORKER_ENDPOINT"); ok {
		workerEndpoint = worker
	} else {
		log.Fatal("WORKER_ENDPOINT environment variable not set")
	}

	log.SetFormatter(&log.TextFormatter{
		ForceColors:  true,
		PadLevelText: true,
	})

	ctx := context.Background()
	gcpClient, err := idtoken.NewClient(ctx, workerEndpoint)

	if err != nil {
		log.Warnf("using default client, failed to fetch ID token necessary for scheduler to comm with worker: %s", err.Error())
	} else {
		log.Info("successfully fetched ID token")
		client = gcpClient
	}

}

func schedule(w http.ResponseWriter, r *http.Request) {
	// run a single request for each institute in a goroutine

	for _, inst := range instituteCodes {
		url := workerEndpoint + "/update?institute=" + inst
		log.Info("Sending request to ", url)
		_, err := client.Post(url, "application/json", nil)
		if err != nil {
			log.Error("Error while sending request to ", url)
		}
	}
}

func main() {
	http.HandleFunc("/schedule", schedule)

	port := "8080"
	if envPort, ok := os.LookupEnv("PORT"); ok {
		port = envPort
	}

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Info("Server closed")
			return
		}

		log.Fatal(err)
	}
}
