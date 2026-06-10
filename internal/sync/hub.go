package sync

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Event is a basic sync payload envelope used by websocket subscribers.
type Event struct {
	Sequence  int64     `json:"sequence"`
	Type      string    `json:"type"`
	Payload   any       `json:"payload,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

type HubStats struct {
	ConnectedClients int   `json:"connected_clients"`
	PublishedTotal   int64 `json:"published_total"`
	DroppedTotal     int64 `json:"dropped_total"`
	LastSequence     int64 `json:"last_sequence"`
	HistoryDepth     int   `json:"history_depth"`
}

const defaultHubHistoryLimit = 1024

func NewEvent(eventType string, payload any) Event {
	return Event{
		Type:      eventType,
		Payload:   payload,
		Timestamp: time.Now().UTC(),
	}
}

// Hub is an in-memory pub-sub broker for sync websocket clients.
type Hub struct {
	mu           sync.RWMutex
	subscribers  map[chan Event]struct{}
	history      []Event
	historyLimit int
	sequence     atomic.Int64
	published    atomic.Int64
	dropped      atomic.Int64
}

func NewHub() *Hub {
	return &Hub{
		subscribers:  make(map[chan Event]struct{}),
		historyLimit: defaultHubHistoryLimit,
	}
}

func (h *Hub) Subscribe(buffer int) (<-chan Event, func()) {
	if buffer <= 0 {
		buffer = 8
	}

	ch := make(chan Event, buffer)

	h.mu.Lock()
	h.subscribers[ch] = struct{}{}
	h.mu.Unlock()

	unsubscribe := func() {
		h.mu.Lock()
		if _, exists := h.subscribers[ch]; exists {
			delete(h.subscribers, ch)
			close(ch)
		}
		h.mu.Unlock()
	}

	return ch, unsubscribe
}

func (h *Hub) Publish(event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	if event.Sequence <= 0 {
		event.Sequence = h.sequence.Add(1)
	} else {
		h.advanceSequence(event.Sequence)
	}

	h.published.Add(1)

	h.mu.Lock()
	h.appendHistory(event)
	for ch := range h.subscribers {
		select {
		case ch <- event:
		default:
			// Drop when subscriber is slow to keep broadcaster non-blocking.
			h.dropped.Add(1)
		}
	}
	h.mu.Unlock()
}

func (h *Hub) appendHistory(event Event) {
	if h.historyLimit <= 0 {
		return
	}

	h.history = append(h.history, event)
	if len(h.history) <= h.historyLimit {
		return
	}

	overflow := len(h.history) - h.historyLimit
	copy(h.history, h.history[overflow:])
	h.history = h.history[:h.historyLimit]
}

func (h *Hub) ReplaySince(sequence int64, limit int) []Event {
	events, _ := h.ReplaySinceWithMetadata(sequence, limit)
	return events
}

func (h *Hub) ReplaySinceWithMetadata(sequence int64, limit int) ([]Event, bool) {
	if sequence < 0 {
		sequence = 0
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.history) == 0 {
		return nil, false
	}

	result := make([]Event, 0, len(h.history))
	for _, event := range h.history {
		if event.Sequence > sequence {
			result = append(result, event)
		}
	}

	truncated := false
	if limit > 0 && len(result) > limit {
		result = result[:limit]
		truncated = true
	}

	cloned := make([]Event, len(result))
	copy(cloned, result)
	return cloned, truncated
}

func (h *Hub) advanceSequence(sequence int64) {
	for {
		current := h.sequence.Load()
		if sequence <= current {
			return
		}
		if h.sequence.CompareAndSwap(current, sequence) {
			return
		}
	}
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subscribers)
}

// sortEventsBySequence sorts a slice of Events by Sequence ascending.
// Used when merging events-table replay with hub-history replay.
func sortEventsBySequence(events []Event) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Sequence < events[j].Sequence
	})
}

func (h *Hub) Stats() HubStats {
	h.mu.RLock()
	historyDepth := len(h.history)
	h.mu.RUnlock()

	return HubStats{
		ConnectedClients: h.ClientCount(),
		PublishedTotal:   h.published.Load(),
		DroppedTotal:     h.dropped.Load(),
		LastSequence:     h.sequence.Load(),
		HistoryDepth:     historyDepth,
	}
}
