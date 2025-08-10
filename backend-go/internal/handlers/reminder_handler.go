package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/axfinn/todoIng/backend-go/internal/services"
)

// ReminderHandler 提醒处理器
type ReminderHandler struct {
	reminderService *services.ReminderService
	db              *mongo.Database
}

// NewReminderHandler 创建提醒处理器
func NewReminderHandler(db *mongo.Database) *ReminderHandler {
	return &ReminderHandler{
		reminderService: services.NewReminderService(db),
		db:              db,
	}
}

// CreateReminder 创建提醒
func (h *ReminderHandler) CreateReminder(w http.ResponseWriter, r *http.Request) {
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

	var req models.CreateReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 验证请求数据
	if req.EventID.IsZero() {
		http.Error(w, "Event ID is required", http.StatusBadRequest)
		return
	}
	if len(req.ReminderTimes) == 0 {
		http.Error(w, "At least one reminder time is required", http.StatusBadRequest)
		return
	}
	if req.ReminderType == "" {
		http.Error(w, "Reminder type is required", http.StatusBadRequest)
		return
	}

	reminder, err := h.reminderService.CreateReminder(context.Background(), objectID, req)
	if err != nil {
		if err.Error() == "event not found" {
			http.Error(w, "Event not found", http.StatusBadRequest)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to create reminder: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(reminder)
}

// GetReminder 获取单个提醒
func (h *ReminderHandler) GetReminder(w http.ResponseWriter, r *http.Request) {
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
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	reminder, err := h.reminderService.GetReminder(context.Background(), objectID, reminderID)
	if err != nil {
		if err.Error() == "reminder not found" {
			http.Error(w, "Reminder not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to get reminder: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminder)
}

// UpdateReminder 更新提醒
func (h *ReminderHandler) UpdateReminder(w http.ResponseWriter, r *http.Request) {
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
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateReminderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	reminder, err := h.reminderService.UpdateReminder(context.Background(), objectID, reminderID, req)
	if err != nil {
		if err.Error() == "reminder not found" {
			http.Error(w, "Reminder not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to update reminder: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(reminder)
}

// DeleteReminder 删除提醒
func (h *ReminderHandler) DeleteReminder(w http.ResponseWriter, r *http.Request) {
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
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	err = h.reminderService.DeleteReminder(context.Background(), objectID, reminderID)
	if err != nil {
		if err.Error() == "reminder not found" {
			http.Error(w, "Reminder not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to delete reminder: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListReminders 获取提醒列表
func (h *ReminderHandler) ListReminders(w http.ResponseWriter, r *http.Request) {
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

	activeOnly := query.Get("active_only") == "true"

	response, err := h.reminderService.ListReminders(context.Background(), objectID, page, pageSize, activeOnly)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list reminders: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUpcomingReminders 获取即将到来的提醒
func (h *ReminderHandler) GetUpcomingReminders(w http.ResponseWriter, r *http.Request) {
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

	hours := 24
	if hoursStr := r.URL.Query().Get("hours"); hoursStr != "" {
		if h, err := strconv.Atoi(hoursStr); err == nil && h > 0 && h <= 168 { // 最多一周
			hours = h
		}
	}

	reminders, err := h.reminderService.GetUpcomingReminders(context.Background(), objectID, hours)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get upcoming reminders: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"reminders": reminders,
		"hours":     hours,
	})
}

// SnoozeReminder (legacy) 暂停提醒 - 已由 internal/api/reminder_handlers.go 替代，保留仅兼容旧路由。
// TODO: remove after front-end fully migrated.
func (h *ReminderHandler) SnoozeReminder(w http.ResponseWriter, r *http.Request) {
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
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	var req struct {
		SnoozeMinutes int `json:"snooze_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// 使用默认值
		req.SnoozeMinutes = 60
	}

	if req.SnoozeMinutes <= 0 {
		req.SnoozeMinutes = 60
	}

	err = h.reminderService.SnoozeReminder(context.Background(), objectID, reminderID, req.SnoozeMinutes)
	if err != nil {
		if err.Error() == "reminder not found" {
			http.Error(w, "Reminder not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to snooze reminder: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":        "Reminder snoozed successfully",
		"snooze_minutes": req.SnoozeMinutes,
	})
}
