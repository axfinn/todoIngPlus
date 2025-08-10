package notifications

import (
	"context"
	"sync"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Hub 负责在单实例内广播通知 (SSE)
type Hub struct {
	mu   sync.RWMutex
	subs map[primitive.ObjectID]map[chan models.Notification]struct{}
}

func NewHub() *Hub {
	return &Hub{subs: make(map[primitive.ObjectID]map[chan models.Notification]struct{})}
}

// Subscribe 返回 channel, context 取消时自动解除订阅
func (h *Hub) Subscribe(ctx context.Context, userID primitive.ObjectID) chan models.Notification {
	ch := make(chan models.Notification, 10)
	h.mu.Lock()
	if _, ok := h.subs[userID]; !ok {
		h.subs[userID] = make(map[chan models.Notification]struct{})
	}
	h.subs[userID][ch] = struct{}{}
	h.mu.Unlock()
	go func() { <-ctx.Done(); h.Unsubscribe(userID, ch) }()
	return ch
}

// Unsubscribe 移除 channel
func (h *Hub) Unsubscribe(userID primitive.ObjectID, ch chan models.Notification) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.subs[userID]; ok {
		if _, ok2 := m[ch]; ok2 {
			delete(m, ch)
			close(ch)
		}
		if len(m) == 0 {
			delete(h.subs, userID)
		}
	}
}

// Broadcast 发送一条通知
func (h *Hub) Broadcast(n models.Notification) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if m, ok := h.subs[n.UserID]; ok {
		for ch := range m {
			select {
			case ch <- n:
			default:
			}
		}
	}
}
