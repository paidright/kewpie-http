package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	kewpie "github.com/davidbanham/kewpie_go"
	"github.com/davidbanham/kewpie_go/types"
	"github.com/paidright/kewpie_http/config"
)

var queue kewpie.Kewpie

func init() {
	queue.Connect(config.KEWPIE_BACKEND, config.QUEUES)
}

var queueRoute = regexp.MustCompile(`/queues/.*`)
var publishMany = regexp.MustCompile(`/queues/.*/publish-many`)

func Router() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			healthHandler.ServeHTTP(w, r)
			return
		}

		if r.URL.Path == "/healthz" {
			healthHandler.ServeHTTP(w, r)
			return
		}

		if publishMany.MatchString(r.URL.Path) {
			if r.Method != "POST" {
				errRes(w, r, http.StatusMethodNotAllowed, "Publish must be done with a POST", nil)
				return
			}

			// Take a task over the wire and pass it to the backend
			publishManyHandler.ServeHTTP(w, r)
			return
		}

		if queueRoute.MatchString(r.URL.Path) {
			switch r.Method {
			case "POST":
				// Take a task over the wire and pass it to the backend
				publishHandler.ServeHTTP(w, r)
				return
			case "GET":
				// Serve a task and immediately mark it complete yolo
				subscribeHandler.ServeHTTP(w, r)
				return
			case "DELETE":
				// Purge the named queue
				fmt.Printf("DEBUG r.Body: %+v \n", r.Body)
				purgeHandler.ServeHTTP(w, r)
				return
			}
		}

		notFoundHandler.ServeHTTP(w, r)
		return
	}
}

func main() {
	addr := ":" + os.Getenv("PORT")

	s := &http.Server{
		Handler: http.HandlerFunc(Router()),
		Addr:    addr,
	}

	log.Printf("INFO Listening on: %s", addr)
	log.Fatalf("ERROR %+v", s.ListenAndServe())
}

var publishHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	task := kewpie.Task{}

	if r.Header.Get("Content-Type") == "application/json" {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "Error receiving payload", err)
			return
		}
		if err := json.Unmarshal(bytes, &task); err != nil {
			errRes(w, r, http.StatusBadRequest, "Error decoding payload", err)
			return
		}
	} else if r.Header.Get("Content-Type") == "application/vnd.api+json" {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "Error receiving payload", err)
			return
		}
		payload := jsonAPIPayload{}
		if err := json.Unmarshal(bytes, &payload); err != nil {
			errRes(w, r, http.StatusBadRequest, "Error decoding payload", err)
			return
		}
		task = payload.Data
	} else {
		if decoded, err := decodeForm(r.Form); err != nil {
			errRes(w, r, http.StatusBadRequest, err.Error(), err)
			return
		} else {
			task = decoded[0]
		}
	}

	queueName := strings.Split(r.URL.Path, "/")[2]

	if err := queue.Publish(r.Context(), queueName, &task); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error handling task", err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	sendPayload(w, r, task)
})

var publishManyHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	tasks := []kewpie.Task{}

	if r.Header.Get("Content-Type") == "application/json" {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "Error receiving payload", err)
			return
		}
		if err := json.Unmarshal(bytes, &tasks); err != nil {
			errRes(w, r, http.StatusBadRequest, "Error decoding payload", err)
			return
		}
	} else if r.Header.Get("Content-Type") == "application/vnd.api+json" {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			errRes(w, r, http.StatusBadRequest, "Error receiving payload", err)
			return
		}
		payload := jsonAPIManyPayload{}
		if err := json.Unmarshal(bytes, &payload); err != nil {
			errRes(w, r, http.StatusBadRequest, "Error decoding payload", err)
			return
		}
		tasks = payload.Data
	} else {
		if decoded, err := decodeForm(r.Form); err != nil {
			errRes(w, r, http.StatusBadRequest, err.Error(), err)
			return
		} else {
			tasks = decoded
		}
	}

	queueName := strings.Split(r.URL.Path, "/")[2]

	for _, task := range tasks {
		if err := queue.Publish(r.Context(), queueName, &task); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error handling task", err)
			return
		}
	}

	w.WriteHeader(http.StatusCreated)
	sendManyPayload(w, r, tasks)
})

func sendPayload(w http.ResponseWriter, r *http.Request, task kewpie.Task) {
	if r.Header.Get("Accept") == "application/vnd.api+json" {
		w.Header().Set("Content-Type", "application/json")
		payload := jsonAPIPayload{
			Data: task,
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error encoding response", err)
			return
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(task); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error encoding response", err)
		return
	}
}

func sendManyPayload(w http.ResponseWriter, r *http.Request, tasks []kewpie.Task) {
	if r.Header.Get("Accept") == "application/vnd.api+json" {
		w.Header().Set("Content-Type", "application/json")
		payload := jsonAPIManyPayload{
			Data: tasks,
		}
		if err := json.NewEncoder(w).Encode(payload); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error encoding response", err)
			return
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error encoding response", err)
		return
	}
}

type yoloHandler struct {
	handleFunc func(types.Task) (bool, error)
}

func (h yoloHandler) Handle(t types.Task) (bool, error) {
	return h.handleFunc(t)
}

var subscribeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	queueName := strings.Split(r.URL.Path, "/")[2]

	handler := yoloHandler{
		handleFunc: func(task kewpie.Task) (bool, error) {
			sendPayload(w, r, task)
			return false, nil
		},
	}

	if err := queue.Pop(r.Context(), queueName, handler); err != nil {
		errRes(w, r, http.StatusInternalServerError, "Error popping job from queue", err)
		return
	}
})

var purgeHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	queueName := strings.Split(r.URL.Path, "/")[2]

	match := r.URL.Query().Get("matching")
	if match != "" {
		if err := queue.PurgeMatching(r.Context(), queueName, match); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error purging queue", err)
			return
		}
	} else {
		if err := queue.Purge(r.Context(), queueName); err != nil {
			errRes(w, r, http.StatusInternalServerError, "Error purging queue", err)
			return
		}
	}

	sendPayload(w, r, kewpie.Task{})
})

var healthHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(currentVersion))
	return
})

var notImplementedHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
	w.Write([]byte("Not implemented"))
	return
})

var notFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not found"))
	return
})

func errRes(w http.ResponseWriter, r *http.Request, status int, message string, err error) {
	fmt.Println("WARN sending error to client", status, message, err)

	errors := []map[string]string{}
	errors = append(errors, map[string]string{
		"detail": message,
	})

	response := jsonAPIPayload{
		Errors: errors,
	}

	w.WriteHeader(status)
	if r.Header.Get("Accept") == "application/vnd.api+json" {
		w.Header().Set("Content-Type", "application/vnd.api+json")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}
	json.NewEncoder(w).Encode(response)
}

func getVal(input []string, i int) string {
	if len(input)-1 < i {
		return ""
	}
	return input[i]
}

func decodeForm(input url.Values) ([]kewpie.Task, error) {
	tasks := []kewpie.Task{}

	for index, body := range input["body"] {
		task := kewpie.Task{}

		delay := getVal(input["delay"], index)
		if delay != "" {
			parsed, err := time.ParseDuration(delay)
			if err != nil {
				return tasks, fmt.Errorf("Delay is not a valid duration, eg: 1s %s", err.Error())
			}
			task.Delay = parsed
		}

		runAt := getVal(input["run_at"], index)
		if runAt != "" {
			parsed, err := time.Parse(time.RFC3339, runAt)
			if err != nil {
				return tasks, fmt.Errorf("Run At is not a valid RFC3339 string eg: 2006-01-02T15:04:05Z07:00 %s", err.Error())
			}
			task.RunAt = parsed
		}

		task.Body = body
		task.NoExpBackoff = getVal(input["no_exp_backoff"], index) == "true"

		tasks = append(tasks, task)
	}
	return tasks, nil
}

type jsonAPIPayload struct {
	Errors []map[string]string `json:"errors"`
	Data   kewpie.Task         `json:"data"`
	Meta   map[string]string   `json:"meta"`
}

type jsonAPIManyPayload struct {
	Errors []map[string]string `json:"errors"`
	Data   []kewpie.Task       `json:"data"`
	Meta   map[string]string   `json:"meta"`
}
