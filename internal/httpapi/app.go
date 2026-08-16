package httpapi

import (
	"encoding/json"
	"errors"
	"io/fs"
	"net/http"
	"strings"

	"spellingchallenge/internal/challenge"
	"spellingchallenge/internal/review"
	webassets "spellingchallenge/web"
)

type App struct {
	challenges *challenge.Service
	reviews    *review.Service
	handler    http.Handler
}

func New(challenges *challenge.Service, reviews *review.Service) *App {
	app := &App{challenges: challenges, reviews: reviews}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/challenges", app.listChallenges)
	mux.HandleFunc("POST /api/answers", app.submitAnswer)
	mux.HandleFunc("GET /api/history", app.history)
	mux.HandleFunc("GET /api/wrong-words", app.wrongWords)
	mux.HandleFunc("GET /api/stats", app.stats)
	mux.HandleFunc("POST /api/reviews/{recordID}/confirmations", app.confirm)
	mux.HandleFunc("GET /api/reviews/{recordID}", app.reviewSummary)
	staticFiles, err := fs.Sub(webassets.Files, ".")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(staticFiles)))
	app.handler = mux
	return app
}

func NewFixture(syncPoint review.SyncPoint) *App {
	return New(challenge.NewService(challenge.FixtureWords()), review.NewService(syncPoint))
}

func (a *App) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	a.handler.ServeHTTP(response, request)
}

func (a *App) listChallenges(response http.ResponseWriter, request *http.Request) {
	difficulty := challenge.Difficulty(request.URL.Query().Get("difficulty"))
	if difficulty == "" {
		difficulty = challenge.DifficultySimple
	}
	prompts, err := a.challenges.Prompts(difficulty)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	writeJSON(response, http.StatusOK, prompts)
}

func (a *App) submitAnswer(response http.ResponseWriter, request *http.Request) {
	var input challenge.Answer
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	result, err := a.challenges.Submit(input)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, challenge.ErrChallengeNotFound) {
			status = http.StatusNotFound
		}
		writeError(response, status, err)
		return
	}
	writeJSON(response, http.StatusCreated, result)
}

func (a *App) history(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, a.challenges.History())
}

func (a *App) wrongWords(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, a.challenges.WrongWords())
}

func (a *App) stats(response http.ResponseWriter, _ *http.Request) {
	writeJSON(response, http.StatusOK, a.challenges.Stats())
}

func (a *App) confirm(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Operator string `json:"operator"`
		Content  string `json:"content"`
	}
	if err := decodeJSON(request, &input); err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	summary, err := a.reviews.Confirm(request.PathValue("recordID"), input.Operator, input.Content)
	if err != nil {
		writeError(response, http.StatusBadRequest, err)
		return
	}
	writeJSON(response, http.StatusOK, summary)
}

func (a *App) reviewSummary(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusOK, a.reviews.Summary(request.PathValue("recordID")))
}

func decodeJSON(request *http.Request, target any) error {
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func writeJSON(response http.ResponseWriter, status int, value any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	_ = json.NewEncoder(response).Encode(value)
}

func writeError(response http.ResponseWriter, status int, err error) {
	writeJSON(response, status, map[string]string{"error": strings.TrimSpace(err.Error())})
}
