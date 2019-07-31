package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	kewpie "github.com/davidbanham/kewpie_go"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

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
	taskPostHandler(rr, req)

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
	taskPostHandler(rr, req)

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
	taskPostHandler(rr, req)

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
	taskPostHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, runAt.Format(time.RFC3339), res.RunAt.Format(time.RFC3339))
}

// TODO post JSONAPI
