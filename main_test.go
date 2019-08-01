package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
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

	assert.Equal(t, http.StatusCreated, rr.Code)
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

	assert.Equal(t, http.StatusCreated, rr.Code)
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

	assert.Equal(t, http.StatusCreated, rr.Code)
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

	assert.Equal(t, http.StatusCreated, rr.Code)
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

	assert.Equal(t, http.StatusCreated, rr.Code)
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

	assert.Equal(t, http.StatusCreated, rr.Code)
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

func TestPurge(t *testing.T) {
	t.Parallel()

	fixture := kewpie.Task{
		Body: `{"hi": "` + uuid.NewV4().String() + `"}`,
	}

	payload, err := json.Marshal(fixture)
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "/queues/purgetest/publish", bytes.NewReader(payload))
	assert.Nil(t, err)

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	publishHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	res := kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, res.Body, fixture.Body)

	purgereq, err := http.NewRequest("GET", "/queues/purgetest/subscribe", nil)
	assert.Nil(t, err)

	purgerr := httptest.NewRecorder()
	purgeHandler(purgerr, purgereq)

	assert.Equal(t, http.StatusOK, purgerr.Code)
}

func TestPurgeMatching(t *testing.T) {
	t.Parallel()

	substr1 := uuid.NewV4().String()
	fixture := kewpie.Task{
		Body: `{"hi": "` + substr1 + `"}`,
	}

	substr2 := uuid.NewV4().String()
	fixture2 := kewpie.Task{
		Body: `{"hi": "` + substr2 + `"}`,
	}

	ctx := context.Background()
	assert.Nil(t, queue.Purge(ctx, "purgematchingtest"))
	assert.Nil(t, queue.Publish(ctx, "purgematchingtest", &fixture))
	assert.Nil(t, queue.Publish(ctx, "purgematchingtest", &fixture2))

	purgereq, err := http.NewRequest("POST", "/queues/purgematchingtest/purge?matching="+substr1, nil)
	assert.Nil(t, err)

	purgerr := httptest.NewRecorder()
	purgeHandler(purgerr, purgereq)

	assert.Equal(t, http.StatusOK, purgerr.Code)

	fired := false
	handler := yoloHandler{
		handleFunc: func(task kewpie.Task) (bool, error) {
			fired = true
			assert.True(t, strings.Contains(task.Body, substr2))
			assert.False(t, strings.Contains(task.Body, substr1))
			return false, nil
		},
	}

	assert.Nil(t, queue.Pop(ctx, "purgematchingtest", handler))
	assert.True(t, fired)
}

func TestGetVal(t *testing.T) {
	assert.Equal(t, "hai", getVal([]string{"hai"}, 0), "shouldn't be empty")
	assert.Equal(t, "", getVal([]string{"hai"}, 1), "off by one")
	assert.Equal(t, "", getVal([]string{"hai"}, 2), "off by two")
}

func TestPublishMany(t *testing.T) {
	t.Parallel()

	uniq1 := uuid.NewV4().String()
	uniq2 := uuid.NewV4().String()
	form := url.Values{
		"body": {
			`{"hi": "` + uniq1 + `"}`,
			`{"hi": "` + uniq2 + `"}`,
		},
	}

	req := &http.Request{
		Method: "POST",
		URL:    &url.URL{Path: "/queues/test/publish-many"},
		Form:   form,
	}

	rr := httptest.NewRecorder()
	publishManyHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	res := []kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Equal(t, len(res), 2)
	assert.Contains(t, res[0].Body, uniq1)
	assert.Contains(t, res[1].Body, uniq2)
}

func TestPublishManyJSON(t *testing.T) {
	t.Parallel()

	uniq1 := uuid.NewV4().String()
	uniq2 := uuid.NewV4().String()

	payload, err := json.Marshal([]kewpie.Task{
		kewpie.Task{
			Body: `{"hi": "` + uniq1 + `"}`,
		},
		kewpie.Task{
			Body: `{"hi": "` + uniq2 + `"}`,
		},
	})
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "/queues/test/publish-many", bytes.NewReader(payload))
	assert.Nil(t, err)

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	publishManyHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	res := []kewpie.Task{}
	assert.Nil(t, json.Unmarshal(rr.Body.Bytes(), &res))
	assert.Contains(t, res[0].Body, uniq1)
	assert.Contains(t, res[1].Body, uniq2)
}

func TestPublishManyJSONAPI(t *testing.T) {
	t.Parallel()

	uniq1 := uuid.NewV4().String()
	uniq2 := uuid.NewV4().String()

	payload, err := json.Marshal(jsonAPIManyPayload{
		Data: []kewpie.Task{
			kewpie.Task{
				Body: `{"hi": "` + uniq1 + `"}`,
			},
			kewpie.Task{
				Body: `{"hi": "` + uniq2 + `"}`,
			},
		},
	})
	assert.Nil(t, err)

	req, err := http.NewRequest("POST", "/queues/test/publish-many", bytes.NewReader(payload))
	assert.Nil(t, err)

	req.Header.Set("Content-Type", "application/vnd.api+json")
	req.Header.Set("Accept", "application/vnd.api+json")

	rr := httptest.NewRecorder()
	publishManyHandler(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	res := jsonAPIManyPayload{}
	huh := json.NewDecoder(rr.Body)
	assert.Nil(t, huh.Decode(&res))
	assert.Empty(t, res.Errors)
	assert.Contains(t, res.Data[0].Body, uniq1)
	assert.Contains(t, res.Data[1].Body, uniq2)
}
