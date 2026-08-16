package challenge

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Difficulty string

const (
	DifficultySimple Difficulty = "simple"
	DifficultyMedium Difficulty = "medium"
)

type PromptMode string

const (
	PromptChinese   PromptMode = "chinese"
	PromptScrambled PromptMode = "scrambled"
)

var (
	ErrChallengeNotFound = errors.New("challenge not found")
	ErrInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidElapsed    = errors.New("elapsed milliseconds must be non-negative")
)

type Word struct {
	ID         string     `json:"id"`
	English    string     `json:"english"`
	Chinese    string     `json:"chinese"`
	Scrambled  string     `json:"scrambled"`
	Difficulty Difficulty `json:"difficulty"`
}

type Prompt struct {
	ChallengeID string     `json:"challengeId"`
	WordID      string     `json:"wordId"`
	Difficulty  Difficulty `json:"difficulty"`
	Chinese     string     `json:"chinese"`
	Scrambled   string     `json:"scrambled"`
}

type Answer struct {
	ChallengeID string `json:"challengeId"`
	Value       string `json:"answer"`
	ElapsedMS   int64  `json:"elapsedMs"`
}

type Result struct {
	ID          string     `json:"id"`
	ChallengeID string     `json:"challengeId"`
	WordID      string     `json:"wordId"`
	Difficulty  Difficulty `json:"difficulty"`
	Chinese     string     `json:"chinese"`
	Expected    string     `json:"expected"`
	Answer      string     `json:"answer"`
	Correct     bool       `json:"correct"`
	ElapsedMS   int64      `json:"elapsedMs"`
	Score       int        `json:"score"`
}

type Stats struct {
	Attempts       int   `json:"attempts"`
	Correct        int   `json:"correct"`
	Wrong          int   `json:"wrong"`
	TotalScore     int   `json:"totalScore"`
	TotalElapsedMS int64 `json:"totalElapsedMs"`
}

type Service struct {
	mu       sync.RWMutex
	words    []Word
	byID     map[string]Word
	history  []Result
	sequence uint64
}

func NewService(words []Word) *Service {
	copyWords := append([]Word(nil), words...)
	byID := make(map[string]Word, len(copyWords))
	for _, word := range copyWords {
		byID[word.ID] = word
	}
	return &Service{words: copyWords, byID: byID}
}

func FixtureWords() []Word {
	return []Word{
		{ID: "simple-apple", English: "apple", Chinese: "苹果", Scrambled: "paple", Difficulty: DifficultySimple},
		{ID: "simple-book", English: "book", Chinese: "书", Scrambled: "okob", Difficulty: DifficultySimple},
		{ID: "simple-water", English: "water", Chinese: "水", Scrambled: "tawer", Difficulty: DifficultySimple},
		{ID: "simple-family", English: "family", Chinese: "家庭", Scrambled: "mifaly", Difficulty: DifficultySimple},
		{ID: "medium-journey", English: "journey", Chinese: "旅程", Scrambled: "rujenoy", Difficulty: DifficultyMedium},
		{ID: "medium-library", English: "library", Chinese: "图书馆", Scrambled: "rbliary", Difficulty: DifficultyMedium},
		{ID: "medium-courage", English: "courage", Chinese: "勇气", Scrambled: "uocareg", Difficulty: DifficultyMedium},
		{ID: "medium-mountain", English: "mountain", Chinese: "山脉", Scrambled: "tanumino", Difficulty: DifficultyMedium},
	}
}

func (s *Service) Prompts(difficulty Difficulty) ([]Prompt, error) {
	if difficulty != DifficultySimple && difficulty != DifficultyMedium {
		return nil, ErrInvalidDifficulty
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	prompts := make([]Prompt, 0, len(s.words))
	for _, word := range s.words {
		if word.Difficulty != difficulty {
			continue
		}
		prompts = append(prompts, Prompt{
			ChallengeID: word.ID,
			WordID:      word.ID,
			Difficulty:  word.Difficulty,
			Chinese:     word.Chinese,
			Scrambled:   word.Scrambled,
		})
	}
	return prompts, nil
}

func (s *Service) Submit(answer Answer) (Result, error) {
	if answer.ElapsedMS < 0 {
		return Result{}, ErrInvalidElapsed
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	word, ok := s.byID[answer.ChallengeID]
	if !ok {
		return Result{}, ErrChallengeNotFound
	}
	s.sequence++
	normalized := strings.ToLower(strings.TrimSpace(answer.Value))
	correct := normalized == word.English
	score := 0
	if correct {
		score = 10
		if word.Difficulty == DifficultyMedium {
			score = 20
		}
	}
	resultID := uuid.NewSHA1(uuid.NameSpaceOID, []byte(fmt.Sprintf("%s:%s:%d", word.ID, normalized, s.sequence))).String()
	result := Result{
		ID:          resultID,
		ChallengeID: word.ID,
		WordID:      word.ID,
		Difficulty:  word.Difficulty,
		Chinese:     word.Chinese,
		Expected:    word.English,
		Answer:      strings.TrimSpace(answer.Value),
		Correct:     correct,
		ElapsedMS:   answer.ElapsedMS,
		Score:       score,
	}
	s.history = append(s.history, result)
	return result, nil
}

func (s *Service) History() []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Result(nil), s.history...)
}

func (s *Service) WrongWords() []Result {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wrong := make([]Result, 0)
	for _, result := range s.history {
		if !result.Correct {
			wrong = append(wrong, result)
		}
	}
	sort.SliceStable(wrong, func(i, j int) bool {
		return wrong[i].ID < wrong[j].ID
	})
	return wrong
}

func (s *Service) Stats() Stats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	stats := Stats{Attempts: len(s.history)}
	for _, result := range s.history {
		if result.Correct {
			stats.Correct++
		} else {
			stats.Wrong++
		}
		stats.TotalScore += result.Score
		stats.TotalElapsedMS += result.ElapsedMS
	}
	return stats
}
