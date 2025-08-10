package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/axfinn/todoIng/backend-go/internal/observability"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/axfinn/todoIng/backend-go/internal/services"
)

// CreateReminder 创建提醒
func (d *ReminderDeps) CreateReminder(w http.ResponseWriter, r *http.Request) {
	writeErr := func(status int, code, msg string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
	}
	userID := GetUserID(r)
	if userID == "" { writeErr(http.StatusUnauthorized, "unauthorized", "Unauthorized"); return }
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil { writeErr(http.StatusBadRequest, "user_id", "Invalid user ID (legacy account). Please re-register to use events/reminders."); return }

	// 读取 body 以便宽松解析
	rawBody, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if strings.TrimSpace(string(rawBody)) == "" { writeErr(http.StatusBadRequest, "empty_body", "Request body is empty"); return }

	var req models.CreateReminderRequest
	// 严格解析
	strictErr := func() error {
		dec := json.NewDecoder(strings.NewReader(string(rawBody)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil { return err }
		if dec.More() { return errors.New("multiple JSON objects") }
		return nil
	}()
	if strictErr != nil {
		// 宽松解析: 支持 eventId / reminderTimes / reminder_type / reminderType
		var m map[string]interface{}
		if err2 := json.Unmarshal(rawBody, &m); err2 != nil {
			observability.LogWarn("CreateReminder decode fail strict=%v loose=%v body=%s", strictErr, err2, truncateBody(rawBody))
			writeErr(http.StatusBadRequest, "json_decode", "Invalid request body: "+strictErr.Error())
			return
		}
		// 映射
		if v, ok := m["event_id"].(string); ok { req.EventID, _ = primitive.ObjectIDFromHex(v) } else if v, ok := m["eventId"].(string); ok { req.EventID, _ = primitive.ObjectIDFromHex(v) }
		if v, ok := m["advance_days"].(float64); ok { req.AdvanceDays = int(v) } else if v, ok := m["advanceDays"].(float64); ok { req.AdvanceDays = int(v) }
		if v, ok := m["reminder_times"].([]interface{}); ok { for _, it := range v { if s, ok2 := it.(string); ok2 { req.ReminderTimes = append(req.ReminderTimes, s) } } } else if v, ok := m["reminderTimes"].([]interface{}); ok { for _, it := range v { if s, ok2 := it.(string); ok2 { req.ReminderTimes = append(req.ReminderTimes, s) } } }
		if v, ok := m["reminder_type"].(string); ok { req.ReminderType = v } else if v, ok := m["reminderType"].(string); ok { req.ReminderType = v }
		if v, ok := m["custom_message"].(string); ok { req.CustomMessage = v } else if v, ok := m["customMessage"].(string); ok { req.CustomMessage = v }
		observability.LogWarn("CreateReminder strict decode failed, fallback used err=%v", strictErr)
	}

	// 字段校验
	if req.EventID.IsZero() { writeErr(http.StatusBadRequest, "field_event_id", "Event ID is required"); return }
	if len(req.ReminderTimes) == 0 { writeErr(http.StatusBadRequest, "field_reminder_times", "At least one reminder time is required"); return }
	if req.ReminderType == "" { writeErr(http.StatusBadRequest, "field_reminder_type", "Reminder type is required"); return }

	reminderService := services.NewReminderService(d.DB)
	reminder, err := reminderService.CreateReminder(context.Background(), objectID, req)
	if err != nil {
		if err.Error() == "event not found" { writeErr(http.StatusBadRequest, "event_not_found", "Event not found"); return }
		writeErr(http.StatusInternalServerError, "create_failed", fmt.Sprintf("Failed to create reminder: %v", err))
		return
	}
	observability.LogInfo("CreateReminder success id=%s user=%s event=%s times=%d", reminder.ID.Hex(), userID, reminder.EventID.Hex(), len(reminder.ReminderTimes))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(reminder)
}

// truncateBody 辅助
func truncateBody(b []byte) string { s := string(b); if len(s) > 500 { return s[:500] + "..." }; return s }

// GetReminder 获取单个提醒
func (d *ReminderDeps) GetReminder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID (legacy account). Please re-register to use events/reminders.", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	reminderService := services.NewReminderService(d.DB)
	reminder, err := reminderService.GetReminder(context.Background(), objectID, reminderID)
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
func (d *ReminderDeps) UpdateReminder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID (legacy account). Please re-register to use events/reminders.", http.StatusBadRequest)
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

	reminderService := services.NewReminderService(d.DB)
	reminder, err := reminderService.UpdateReminder(context.Background(), objectID, reminderID, req)
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
func (d *ReminderDeps) DeleteReminder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID (legacy account). Please re-register to use events/reminders.", http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}

	reminderService := services.NewReminderService(d.DB)
	err = reminderService.DeleteReminder(context.Background(), objectID, reminderID)
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
func (d *ReminderDeps) ListReminders(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID (legacy account). Please re-register to use events/reminders.", http.StatusBadRequest)
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

	reminderService := services.NewReminderService(d.DB)
	response, err := reminderService.ListReminders(context.Background(), objectID, page, pageSize, activeOnly)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list reminders: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetUpcomingReminders 获取即将到来的提醒
func (d *ReminderDeps) GetUpcomingReminders(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
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

	reminderService := services.NewReminderService(d.DB)
	reminders, err := reminderService.GetUpcomingReminders(context.Background(), objectID, hours)
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

// SnoozeReminder 暂停提醒
func (d *ReminderDeps) SnoozeReminder(w http.ResponseWriter, r *http.Request) {
	writeErr := func(status int, code, msg string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
	}
	userID := GetUserID(r)
	if userID == "" { writeErr(http.StatusUnauthorized, "unauthorized", "Unauthorized"); return }
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil { writeErr(http.StatusBadRequest, "user_id", "Invalid user ID"); return }

	vars := mux.Vars(r)
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil { writeErr(http.StatusBadRequest, "reminder_id", "Invalid reminder ID"); return }

	var req struct { SnoozeMinutes int `json:"snooze_minutes"` }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.SnoozeMinutes = 60 // fallback
	}
	if req.SnoozeMinutes <= 0 { req.SnoozeMinutes = 60 }

	reminderService := services.NewReminderService(d.DB)
	if err := reminderService.SnoozeReminder(context.Background(), objectID, reminderID, req.SnoozeMinutes); err != nil {
		if err.Error() == "reminder not found" { writeErr(http.StatusNotFound, "reminder_not_found", "Reminder not found"); return }
		writeErr(http.StatusInternalServerError, "snooze_failed", fmt.Sprintf("Failed to snooze reminder: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{ "message": "Reminder snoozed successfully", "snooze_minutes": req.SnoozeMinutes })
}

// SetupReminderRoutes 设置提醒路由
func SetupReminderRoutes(r *mux.Router, deps *ReminderDeps) {
	s := r.PathPrefix("/api/reminders").Subrouter()
	
	// 提醒管理路由
	s.Handle("", Auth(http.HandlerFunc(deps.ListReminders))).Methods(http.MethodGet)
	s.Handle("", Auth(http.HandlerFunc(deps.CreateReminder))).Methods(http.MethodPost)
	s.Handle("/{id}", Auth(http.HandlerFunc(deps.GetReminder))).Methods(http.MethodGet)
	s.Handle("/{id}", Auth(http.HandlerFunc(deps.UpdateReminder))).Methods(http.MethodPut)
	s.Handle("/{id}", Auth(http.HandlerFunc(deps.DeleteReminder))).Methods(http.MethodDelete)
	
	// 特殊路由
	s.Handle("/upcoming", Auth(http.HandlerFunc(deps.GetUpcomingReminders))).Methods(http.MethodGet)
	s.Handle("/{id}/snooze", Auth(http.HandlerFunc(deps.SnoozeReminder))).Methods(http.MethodPost)
}
