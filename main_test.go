package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	kewpie "github.com/davidbanham/kewpie_go"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func init() {
	if err := queue.Purge(context.Background(), "pubtest"); err != nil {
		log.Fatal(err)
	}
	if err := queue.Purge(context.Background(), "test"); err != nil {
		log.Fatal(err)
	}
}

func TestPublishDelay(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"body":  {`{"hi": "` + uuid.NewV4().String() + `"}`},
		"delay": {"10s"},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/queues/test/publish"},
		Form:   form,
	}

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.NotEmpty(t, res.ID)
	assert.True(t, res.RunAt.After(time.Now().Add(8*time.Second)))
}

func TestPublishRunAt(t *testing.T) {
	t.Parallel()

	runAt := time.Now().Add(1 * time.Minute).Format(time.RFC3339)

	form := url.Values{
		"body":   {`{"hi": "` + uuid.NewV4().String() + `"}`},
		"run_at": {runAt},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/queues/test/publish"},
		Form:   form,
	}

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, runAt, res.RunAt.Format(time.RFC3339))
}

func TestPublishNoExpBackoff(t *testing.T) {
	t.Parallel()

	form := url.Values{
		"body":           {`{"hi": "` + uuid.NewV4().String() + `"}`},
		"no_exp_backoff": {"true"},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/queues/test/publish"},
		Form:   form,
	}

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.True(t, res.NoExpBackoff)
}

func TestPublishJSON(t *testing.T) {
	t.Parallel()

	runAt := time.Now().Add(1 * time.Minute)

	payload, err := json.Marshal(kewpie.Task{
		Body:  `{"hi": "` + uuid.NewV4().String() + `"}`,
		RunAt: runAt,
	})
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "/queues/test/publish", bytes.NewReader(payload))
	assert.Nil(t, err)

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, runAt.Format(time.RFC3339), res.RunAt.Format(time.RFC3339))
}

func TestPublishJSONAPI(t *testing.T) {
	t.Parallel()

	runAt := time.Now().Add(1 * time.Minute)

	payload, err := json.Marshal(jsonAPIPayload{
		Data: kewpie.Task{
			Body:  `{"hi": "` + uuid.NewV4().String() + `"}`,
			RunAt: runAt,
		},
	})
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "/queues/test/publish", bytes.NewReader(payload))
	assert.Nil(t, err)

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := jsonAPIPayload{}
	huh := json.NewDecoder(rr.Body)
	assert.Nil(t, huh.Decode(&res))
	assert.Empty(t, res.Errors)
	assert.Equal(t, runAt.Format(time.RFC3339), res.Data.RunAt.Format(time.RFC3339))
}

func TestSubscribe(t *testing.T) {
	t.Parallel()

	fixture := kewpie.Task{
		Body: `{"hi": "` + uuid.NewV4().String() + `"}`,
	}

	payload, err := json.Marshal(fixture)
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "/queues/pubtest/publish", bytes.NewReader(payload))
	assert.Nil(t, err)

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, res.Body, fixture.Body)

	subreq, err := http.NewRequest("GET", "/queues/pubtest/subscribe", nil)
	assert.Nil(t, err)

	subrr := httptest.NewRecorder()
	subscribeHandler(subrr, subreq)

	assert.Equal(t, http.StatusOK, subrr.Code)

	subbed := kewpie.Task{}

	assert.Nil(t, json.Unmarshal(subrr.Body.Bytes(), &subbed))

	assert.Equal(t, res.Body, subbed.Body)
	assert.Equal(t, res.ID, subbed.ID)
}
