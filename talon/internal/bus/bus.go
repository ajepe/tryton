package bus

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultTimeout     = 30 * time.Second
	DefaultHistorySize = 1024
)

type Message struct {
	ID        string                 `json:"id"`
	Channel   string                 `json:"channel"`
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

type envelope struct {
	seq int64
	msg Message
}

type Hub struct {
	mu          sync.Mutex
	nextSeq     int64
	historySize int
	channels    map[string][]envelope
	index       map[string]int64
	waiters     map[int64]chan struct{}
	nextWaiter  int64
	timeout     time.Duration
}

func NewHub() *Hub {
	return &Hub{
		historySize: DefaultHistorySize,
		channels:    map[string][]envelope{},
		index:       map[string]int64{},
		waiters:     map[int64]chan struct{}{},
		timeout:     DefaultTimeout,
	}
}

func (h *Hub) SetTimeout(d time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.timeout = d
}

func (h *Hub) Publish(channel string, msg Message) Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextSeq++
	seq := h.nextSeq
	if msg.ID == "" {
		msg.ID = newID()
	}
	if msg.Timestamp.IsZero() {
		msg.Timestamp = time.Now().UTC()
	}
	msg.Channel = channel
	h.index[msg.ID] = seq
	entries := append(h.channels[channel], envelope{seq: seq, msg: msg})
	if len(entries) > h.historySize {
		dropped := entries[0]
		entries = entries[1:]
		if h.index[dropped.msg.ID] == dropped.seq {
			delete(h.index, dropped.msg.ID)
		}
	}
	h.channels[channel] = entries
	for _, ch := range h.waiters {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
	return msg
}

func (h *Hub) Notify(channel string, data map[string]interface{}) Message {
	return h.Publish(channel, Message{Type: "notification", Data: data})
}

func (h *Hub) Poll(ctx context.Context, channels []string, lastID string, timeout time.Duration) ([]Message, error) {
	if timeout <= 0 {
		timeout = h.getTimeout()
	}
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()

	for {
		msgs := h.messages(channels, lastID)
		if len(msgs) > 0 {
			return msgs, nil
		}

		waiterID, signal := h.addWaiter()
		select {
		case <-ctx.Done():
			h.removeWaiter(waiterID)
			return nil, ctx.Err()
		case <-deadline.C:
			h.removeWaiter(waiterID)
			return nil, nil
		case <-signal:
			h.removeWaiter(waiterID)
		}
	}
}

func (h *Hub) messages(channels []string, lastID string) []Message {
	h.mu.Lock()
	defer h.mu.Unlock()
	after := int64(0)
	if lastID != "" {
		if seq, ok := h.index[lastID]; ok {
			after = seq
		}
	}
	result := []Message{}
	for _, channel := range channels {
		for _, entry := range h.channels[channel] {
			if entry.seq > after {
				result = append(result, entry.msg)
			}
		}
	}
	return result
}

func (h *Hub) getTimeout() time.Duration {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.timeout
}

func (h *Hub) addWaiter() (int64, chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextWaiter++
	id := h.nextWaiter
	ch := make(chan struct{}, 1)
	h.waiters[id] = ch
	return id, ch
}

func (h *Hub) removeWaiter(id int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.waiters, id)
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Channels    []string `json:"channels"`
		LastMessage string   `json:"last_message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if len(req.Channels) == 0 {
		http.Error(w, "channels required", http.StatusBadRequest)
		return
	}

	messages, err := h.Poll(r.Context(), req.Channels, req.LastMessage, 0)
	if err != nil {
		return
	}

	type item struct {
		Channel string  `json:"channel"`
		Message Message `json:"message"`
	}
	response := make([]item, 0, len(messages))
	for _, msg := range messages {
		response = append(response, item{Channel: msg.Channel, Message: msg})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString(b[:])
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return hex.EncodeToString(b[0:4]) + "-" +
		hex.EncodeToString(b[4:6]) + "-" +
		hex.EncodeToString(b[6:8]) + "-" +
		hex.EncodeToString(b[8:10]) + "-" +
		hex.EncodeToString(b[10:16])
}
