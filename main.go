package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	kewpie "github.com/davidbanham/kewpie_go"
	"github.com/davidbanham/kewpie_http/config"
)

var queue kewpie.Kewpie

func init() {
	queue.Connect(config.KEWPIE_BACKEND, config.QUEUES)
}

func main() {
	router := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			healthHandler.ServeHTTP(w, r)
		}

		if r.URL.Path == "/publish" {
			// Take a task over the wire and pass it to the backend
		}

		if r.URL.Path == "/subscribe" {
			// Serve a task and immediately mark it complete yolo
		}
	})

	addr := ":" + os.Getenv("PORT")

	s := &http.Server{
		Handler: router,
		Addr:    addr,
	}

	log.Printf("INFO Listening on: %s", addr)
	log.Fatalf("ERROR %+v", s.ListenAndServe())
}

func taskPostHandler(w http.ResponseWriter, r *http.Request) {
	delay := time.Duration(0)
	if r.FormValue("delay") != "" {
		parsed, err := time.ParseDuration(r.FormValue("delay"))
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "Delay is not a valid duration, eg: 1s", err)
			return
		}
		delay = parsed
	}

	runAt := time.Time{}
	if r.FormValue("run_at") != "" {
		parsed, err := time.Parse(time.RFC3339, r.FormValue("run_at"))
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "Run At is not a valid RFC3339 string eg: 2006-01-02T15:04:05Z07:00", err)
			return
		}
		runAt = parsed
	}

	task := kewpie.Task{
		Body:         r.FormValue("body"),
		Delay:        delay,
		RunAt:        runAt,
		NoExpBackoff: r.FormValue("no_exp_backoff") == "true",
	}

	queueName := strings.Split(r.URL.Path, "/")[2]

	if err := queue.Publish(r.Context(), queueName, &task); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error handling task", err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

var healthHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
	return
})

func errRes(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	fmt.Println("WARN sending error to client", status, message, err)

	response := errorResponse{
		Error: message,
	}

	w.WriteHeader(status)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type errorResponse struct {
	Error string `json:"error"`
}
