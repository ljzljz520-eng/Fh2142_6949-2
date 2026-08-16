package review

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

type UpdateStage string

const (
	StageAfterLoad  UpdateStage = "after_load"
	StageAfterStore UpdateStage = "after_store"
)

var ErrInvalidConfirmation = errors.New("record, operator and content are required")

type UpdateEvent struct {
	RecordID string
	Operator string
	Stage    UpdateStage
}

type SyncPoint func(UpdateEvent)

type Confirmation struct {
	Operator string `json:"operator"`
	Content  string `json:"content"`
}

type Summary struct {
	RecordID      string         `json:"recordId"`
	Confirmations []Confirmation `json:"confirmations"`
}

type record struct {
	ID            string
	Confirmations map[string]string
}

type Service struct {
	mu        sync.RWMutex
	records   map[string]record
	syncPoint SyncPoint
}

func NewService(syncPoint SyncPoint) *Service {
	return &Service{records: make(map[string]record), syncPoint: syncPoint}
}

func (s *Service) Confirm(recordID, operator, content string) (Summary, error) {
	recordID = strings.TrimSpace(recordID)
	operator = strings.TrimSpace(operator)
	content = strings.TrimSpace(content)
	if recordID == "" || operator == "" || content == "" {
		return Summary{}, ErrInvalidConfirmation
	}

	// Load phase: read the current record under a read lock. The after-load sync
	// point below deliberately lets another operator confirm the same record
	// between this load and the store, so the loaded snapshot may be stale by the
	// time we persist. The store must therefore merge this operator's change into
	// the latest state instead of overwriting the record with the stale snapshot.
	s.mu.RLock()
	current, ok := s.records[recordID]
	next := record{ID: recordID, Confirmations: make(map[string]string)}
	if ok {
		for name, value := range current.Confirmations {
			next.Confirmations[name] = value
		}
	}
	s.mu.RUnlock()

	s.notify(UpdateEvent{RecordID: recordID, Operator: operator, Stage: StageAfterLoad})

	next.Confirmations[operator] = content

	// Store phase: re-read the latest state under a write lock and fold in any
	// confirmations another operator added between our load and store. Only our
	// own entry keeps the value we just confirmed; every other entry is taken
	// from the latest state, so concurrent confirmations are never dropped.
	s.mu.Lock()
	if latest, latestOK := s.records[recordID]; latestOK {
		for name, value := range latest.Confirmations {
			if name == operator {
				continue
			}
			next.Confirmations[name] = value
		}
	}
	s.records[recordID] = next
	s.mu.Unlock()
	s.notify(UpdateEvent{RecordID: recordID, Operator: operator, Stage: StageAfterStore})

	return makeSummary(next), nil
}

func (s *Service) Summary(recordID string) Summary {
	s.mu.RLock()
	current, ok := s.records[recordID]
	s.mu.RUnlock()
	if !ok {
		return Summary{RecordID: recordID, Confirmations: []Confirmation{}}
	}
	return makeSummary(current)
}

func (s *Service) notify(event UpdateEvent) {
	if s.syncPoint != nil {
		s.syncPoint(event)
	}
}

func makeSummary(value record) Summary {
	confirmations := make([]Confirmation, 0, len(value.Confirmations))
	for operator, content := range value.Confirmations {
		confirmations = append(confirmations, Confirmation{Operator: operator, Content: content})
	}
	sort.Slice(confirmations, func(i, j int) bool {
		return confirmations[i].Operator < confirmations[j].Operator
	})
	return Summary{RecordID: value.ID, Confirmations: confirmations}
}
