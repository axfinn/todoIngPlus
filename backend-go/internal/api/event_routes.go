package api

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

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/axfinn/todoIng/backend-go/internal/services"
)

// EventDeps 事件相关依赖
type EventDeps struct {
	DB *mongo.Database
}

// ReminderDeps 提醒相关依赖
type ReminderDeps struct {
	DB *mongo.Database
}

// DashboardDeps 看板相关依赖
type DashboardDeps struct {
	DB *mongo.Database
}

// CreateEvent 创建事件
func (d *EventDeps) CreateEvent(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	event, err := eventService.CreateEvent(context.Background(), objectID, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create event: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
}

// GetEvent 获取单个事件
func (d *EventDeps) GetEvent(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	event, err := eventService.GetEvent(context.Background(), objectID, eventID)
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
func (d *EventDeps) UpdateEvent(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	event, err := eventService.UpdateEvent(context.Background(), objectID, eventID, req)
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
func (d *EventDeps) DeleteEvent(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	err = eventService.DeleteEvent(context.Background(), objectID, eventID)
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
func (d *EventDeps) ListEvents(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	response, err := eventService.ListEvents(context.Background(), objectID, page, pageSize, eventType, startDate, endDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUpcomingEvents 获取即将到来的事件
func (d *EventDeps) GetUpcomingEvents(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	events, err := eventService.GetUpcomingEvents(context.Background(), objectID, days)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get upcoming events: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"days":   days,
	})
}

// GetCalendarEvents 获取日历视图事件
func (d *EventDeps) GetCalendarEvents(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	calendar, err := eventService.GetCalendarEvents(context.Background(), objectID, year, month)
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
func (d *EventDeps) SearchEvents(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(d.DB)
	events, err := eventService.SearchEvents(context.Background(), objectID, keyword, limit)
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

// SetupEventRoutes 设置事件路由
func SetupEventRoutes(r *mux.Router, deps *EventDeps) {
	s := r.PathPrefix("/api/events").Subrouter()
	
	// 事件管理路由
	s.Handle("", Auth(http.HandlerFunc(deps.ListEvents))).Methods(http.MethodGet)
	s.Handle("", Auth(http.HandlerFunc(deps.CreateEvent))).Methods(http.MethodPost)
	s.Handle("/{id}", Auth(http.HandlerFunc(deps.GetEvent))).Methods(http.MethodGet)
	s.Handle("/{id}", Auth(http.HandlerFunc(deps.UpdateEvent))).Methods(http.MethodPut)
	s.Handle("/{id}", Auth(http.HandlerFunc(deps.DeleteEvent))).Methods(http.MethodDelete)
	
	// 特殊路由
	s.Handle("/upcoming", Auth(http.HandlerFunc(deps.GetUpcomingEvents))).Methods(http.MethodGet)
	s.Handle("/calendar", Auth(http.HandlerFunc(deps.GetCalendarEvents))).Methods(http.MethodGet)
	s.Handle("/search", Auth(http.HandlerFunc(deps.SearchEvents))).Methods(http.MethodGet)
}
