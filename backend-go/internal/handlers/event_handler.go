package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
)

// EventHandler 事件处理器
type EventHandler struct {
	eventService *services.EventService
	db           *mongo.Database
}

// NewEventHandler 创建事件处理器
func NewEventHandler(db *mongo.Database) *EventHandler {
	return &EventHandler{
		eventService: services.NewEventService(db),
		db:           db,
	}
}

// CreateEvent 创建事件
func (h *EventHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req models.CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证请求数据
	if req.Title == "" {
		http.Error(w, "Title is required", http.StatusBadRequest)
		return
	}
	if req.EventType == "" {
		http.Error(w, "Event type is required", http.StatusBadRequest)
		return
	}
	if req.EventDate.IsZero() {
		http.Error(w, "Event date is required", http.StatusBadRequest)
		return
	}

	event, err := h.eventService.CreateEvent(context.Background(), objectID, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// GetEvent 获取单个事件
func (h *EventHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	eventIDStr := vars["id"]
	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	event, err := h.eventService.GetEvent(context.Background(), objectID, eventID)
	if err != nil {
		if err.Error() == "event not found" {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// UpdateEvent 更新事件
func (h *EventHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	eventIDStr := vars["id"]
	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	event, err := h.eventService.UpdateEvent(context.Background(), objectID, eventID, req)
	if err != nil {
		if err.Error() == "event not found" {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// DeleteEvent 删除事件
func (h *EventHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	eventIDStr := vars["id"]
	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	err = h.eventService.DeleteEvent(context.Background(), objectID, eventID)
	if err != nil {
		if err.Error() == "event not found" {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete event: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListEvents 获取事件列表
func (h *EventHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	// 解析查询参数
	query := r.URL.Query()

	page := 1
	if pageStr := query.Get("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := query.Get("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	eventType := query.Get("event_type")

	var startDate, endDate *time.Time
	if startDateStr := query.Get("start_date"); startDateStr != "" {
		if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &sd
		}
	}
	if endDateStr := query.Get("end_date"); endDateStr != "" {
		if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &ed
		}
	}

	response, err := h.eventService.ListEvents(context.Background(), objectID, page, pageSize, eventType, startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUpcomingEvents 获取即将到来的事件
func (h *EventHandler) GetUpcomingEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	days := 7
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 30 {
			days = d
		}
	}

	events, err := h.eventService.GetUpcomingEvents(context.Background(), objectID, days)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get upcoming events: %v", err), http.StatusInternalServerError)
		return
	}
	// 增强：为每个事件计算倒计时（秒）与剩余天数
	now := time.Now()
	type eventWithCountdown struct {
		models.Event
		CountdownSeconds int `json:"countdown_seconds"`
		DaysLeft         int `json:"days_left"`
	}
	res := make([]eventWithCountdown, 0, len(events))
	for _, ev := range events {
		secs := int(ev.EventDate.Sub(now).Seconds())
		days := int(ev.EventDate.Sub(now).Hours() / 24)
		res = append(res, eventWithCountdown{Event: ev, CountdownSeconds: secs, DaysLeft: days})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": res,
		"days":   days,
	})
}

// GetCalendarEvents 获取日历视图事件
func (h *EventHandler) GetCalendarEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	query := r.URL.Query()

	year := time.Now().Year()
	if yearStr := query.Get("year"); yearStr != "" {
		if y, err := strconv.Atoi(yearStr); err == nil && y >= 2020 && y <= 2050 {
			year = y
		}
	}

	month := int(time.Now().Month())
	if monthStr := query.Get("month"); monthStr != "" {
		if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
			month = m
		}
	}

	calendar, err := h.eventService.GetCalendarEvents(context.Background(), objectID, year, month)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get calendar events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"year":     year,
		"month":    month,
		"calendar": calendar,
	})
}

// SearchEvents 搜索事件
func (h *EventHandler) SearchEvents(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	keyword := r.URL.Query().Get("q")
	if keyword == "" {
		http.Error(w, "Query parameter 'q' is required", http.StatusBadRequest)
		return
	}

	limit := 20
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	events, err := h.eventService.SearchEvents(context.Background(), objectID, keyword, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to search events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"keyword": keyword,
		"events":  events,
		"total":   len(events),
	})
}
