package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"sync"
	"testing"

	"spellingchallenge/internal/challenge"
	"spellingchallenge/internal/httpapi"
	"spellingchallenge/internal/review"
)

func TestChallengeHistoryWrongWordsAndStats(t *testing.T) {
	app := httpapi.NewFixture(nil)

	status, payload := perform(t, app, http.MethodGet, "/api/challenges?difficulty=simple", nil)
	if status != http.StatusOK {
		t.Fatalf("challenge status = %d, want %d", status, http.StatusOK)
	}
	var prompts []challenge.Prompt
	decode(t, payload, &prompts)
	if len(prompts) != 4 {
		t.Fatalf("simple prompts = %d, want 4", len(prompts))
	}

	status, payload = perform(t, app, http.MethodPost, "/api/answers", map[string]any{
		"challengeId": "simple-apple",
		"answer":      " Apple ",
		"elapsedMs":   1200,
	})
	if status != http.StatusCreated {
		t.Fatalf("correct submission status = %d, want %d", status, http.StatusCreated)
	}
	var correct challenge.Result
	decode(t, payload, &correct)
	if !correct.Correct || correct.Score != 10 || correct.ID == "" {
		t.Fatalf("correct result = %+v", correct)
	}

	status, payload = perform(t, app, http.MethodPost, "/api/answers", map[string]any{
		"challengeId": "medium-library",
		"answer":      "libary",
		"elapsedMs":   2800,
	})
	if status != http.StatusCreated {
		t.Fatalf("wrong submission status = %d, want %d", status, http.StatusCreated)
	}
	var wrong challenge.Result
	decode(t, payload, &wrong)
	if wrong.Correct || wrong.Expected != "library" || wrong.Score != 0 {
		t.Fatalf("wrong result = %+v", wrong)
	}

	_, payload = perform(t, app, http.MethodGet, "/api/history", nil)
	var history []challenge.Result
	decode(t, payload, &history)
	if len(history) != 2 || history[0].Expected != "apple" || history[1].Expected != "library" {
		t.Fatalf("history = %+v", history)
	}

	_, payload = perform(t, app, http.MethodGet, "/api/wrong-words", nil)
	var wrongWords []challenge.Result
	decode(t, payload, &wrongWords)
	if len(wrongWords) != 1 || wrongWords[0].Expected != "library" {
		t.Fatalf("wrong words = %+v", wrongWords)
	}

	_, payload = perform(t, app, http.MethodGet, "/api/stats", nil)
	var stats challenge.Stats
	decode(t, payload, &stats)
	if stats.Attempts != 2 || stats.Correct != 1 || stats.Wrong != 1 || stats.TotalScore != 10 || stats.TotalElapsedMS != 4000 {
		t.Fatalf("stats = %+v", stats)
	}
}

func TestMediumChallengeFixtureAndStaticInterface(t *testing.T) {
	app := httpapi.NewFixture(nil)
	status, payload := perform(t, app, http.MethodGet, "/api/challenges?difficulty=medium", nil)
	if status != http.StatusOK {
		t.Fatalf("medium challenge status = %d, want %d", status, http.StatusOK)
	}
	var prompts []challenge.Prompt
	decode(t, payload, &prompts)
	if len(prompts) != 4 || prompts[0].Chinese != "旅程" || prompts[0].Scrambled != "rujenoy" {
		t.Fatalf("medium prompts = %+v", prompts)
	}

	status, payload = perform(t, app, http.MethodGet, "/", nil)
	if status != http.StatusOK || !bytes.Contains(payload, []byte("单词拼写挑战")) {
		t.Fatalf("static interface status = %d, body = %q", status, payload)
	}
}

func TestConcurrentConfirmationsRemainInSummary(t *testing.T) {
	aliceLoaded := make(chan struct{})
	bobStored := make(chan struct{})
	hook := func(event review.UpdateEvent) {
		if event.Operator == "操作员甲" && event.Stage == review.StageAfterLoad {
			close(aliceLoaded)
			<-bobStored
		}
		if event.Operator == "操作员乙" && event.Stage == review.StageAfterStore {
			close(bobStored)
		}
	}
	app := httpapi.NewFixture(hook)

	var wait sync.WaitGroup
	wait.Add(1)
	var aliceStatus int
	go func() {
		defer wait.Done()
		aliceStatus, _ = perform(t, app, http.MethodPost, "/api/reviews/daily-review/confirmations", map[string]string{
			"operator": "操作员甲",
			"content":  "已确认英文拼写",
		})
	}()

	<-aliceLoaded
	bobStatus, _ := perform(t, app, http.MethodPost, "/api/reviews/daily-review/confirmations", map[string]string{
		"operator": "操作员乙",
		"content":  "已确认中文释义",
	})
	wait.Wait()
	if aliceStatus != http.StatusOK || bobStatus != http.StatusOK {
		t.Fatalf("confirmation statuses = %d and %d", aliceStatus, bobStatus)
	}

	status, payload := perform(t, app, http.MethodGet, "/api/reviews/daily-review", nil)
	if status != http.StatusOK {
		t.Fatalf("summary status = %d, want %d", status, http.StatusOK)
	}
	var summary review.Summary
	decode(t, payload, &summary)
	got := make([]string, 0, len(summary.Confirmations))
	for _, confirmation := range summary.Confirmations {
		got = append(got, confirmation.Operator+":"+confirmation.Content)
	}
	sort.Strings(got)
	want := []string{"操作员乙:已确认中文释义", "操作员甲:已确认英文拼写"}
	sort.Strings(want)
	if len(got) != len(want) {
		t.Fatalf("summary confirmations = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("summary confirmations = %v, want %v", got, want)
		}
	}
}

func perform(t *testing.T, handler http.Handler, method, path string, body any) (int, []byte) {
	t.Helper()
	var encoded []byte
	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response.Code, response.Body.Bytes()
}

func decode(t *testing.T, payload []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("decode %q: %v", payload, err)
	}
}
