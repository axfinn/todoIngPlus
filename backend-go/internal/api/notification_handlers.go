package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/axfinn/todoIng/backend-go/internal/notifications"
	"github.com/axfinn/todoIng/backend-go/internal/services"
	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationDeps struct {
	DB      interface{}
	Service *services.NotificationService
	Hub     *notifications.Hub
}

// SetupNotificationRoutes 注册通知路由
func SetupNotificationRoutes(r *mux.Router, deps *NotificationDeps) {
	s := r.PathPrefix("/api/notifications").Subrouter()
	s.Handle("", Auth(http.HandlerFunc(deps.listNotifications))).Methods(http.MethodGet)
	s.Handle("/stream", Auth(http.HandlerFunc(deps.streamNotifications))).Methods(http.MethodGet)
	s.Handle("/{id}/read", Auth(http.HandlerFunc(deps.markRead))).Methods(http.MethodPost)
	s.Handle("/read_all", Auth(http.HandlerFunc(deps.markAllRead))).Methods(http.MethodPost)
	// 测试创建通知端点（开发使用）
	s.Handle("/test", Auth(http.HandlerFunc(deps.createTestNotification))).Methods(http.MethodPost)
}

func (d *NotificationDeps) createTestNotification(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	n, err := d.Service.Create(ctx, models.NotificationCreate{UserID: uid, Type: "test", Message: "测试通知 " + time.Now().Format("15:04:05"), Metadata: map[string]interface{}{"demo": true}})
	if err != nil {
		http.Error(w, "Failed to create test notification", http.StatusInternalServerError)
		return
	}
	if d.Hub != nil {
		d.Hub.Broadcast(n)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(n)
}

func (d *NotificationDeps) listNotifications(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	unreadOnly := q.Get("unread") == "true"
	limit := 50
	if ls := q.Get("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil {
			limit = v
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	list, err := d.Service.List(ctx, uid, unreadOnly, limit)
	if err != nil {
		http.Error(w, "Failed to list notifications", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"notifications": list})
}

func (d *NotificationDeps) streamNotifications(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	ctx := r.Context()
	ch := d.Hub.Subscribe(ctx, uid)
	_, _ = w.Write([]byte(":ok\n\n"))
	flusher.Flush()
	heartbeat := time.NewTicker(25 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case n, ok := <-ch:
			if !ok {
				return
			}
			b, _ := json.Marshal(n)
			_, _ = w.Write([]byte("event: notification\n"))
			_, _ = w.Write([]byte("data: "))
			_, _ = w.Write(b)
			_, _ = w.Write([]byte("\n\n"))
			flusher.Flush()
		case <-heartbeat.C:
			_, _ = w.Write([]byte(":hb\n\n"))
			flusher.Flush()
		}
	}
}

func (d *NotificationDeps) markRead(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	idStr := mux.Vars(r)["id"]
	nid, err := primitive.ObjectIDFromHex(idStr)
	if err != nil {
		http.Error(w, "Invalid notification ID", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	if err := d.Service.MarkRead(ctx, uid, nid); err != nil {
		http.Error(w, "Failed to mark read", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (d *NotificationDeps) markAllRead(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	cnt, err := d.Service.MarkAllRead(ctx, uid)
	if err != nil {
		http.Error(w, "Failed to mark all read", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"updated": cnt})
}
