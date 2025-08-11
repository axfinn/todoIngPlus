package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/email"
	"github.com/axfinn/todoIngPlus/backend-go/internal/observability"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
)

// CreateReminder 创建提醒
func (d *ReminderDeps) CreateReminder(w http.ResponseWriter, r *http.Request) {
	writeErr := func(status int, code, msg string) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"code": code, "message": msg})
	}
	userID := GetUserID(r)
	if userID == "" {
		writeErr(http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeErr(http.StatusBadRequest, "user_id", "Invalid user ID (legacy account). Please re-register to use events/reminders.")
		return
	}

	// 读取 body 以便宽松解析
	rawBody, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if strings.TrimSpace(string(rawBody)) == "" {
		writeErr(http.StatusBadRequest, "empty_body", "Request body is empty")
		return
	}

	var req models.CreateReminderRequest
	// 严格解析
	strictErr := func() error {
		dec := json.NewDecoder(strings.NewReader(string(rawBody)))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&req); err != nil {
			return err
		}
		if dec.More() {
			return errors.New("multiple JSON objects")
		}
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
		if v, ok := m["event_id"].(string); ok {
			req.EventID, _ = primitive.ObjectIDFromHex(v)
		} else if v, ok := m["eventId"].(string); ok {
			req.EventID, _ = primitive.ObjectIDFromHex(v)
		}
		if v, ok := m["advance_days"].(float64); ok {
			req.AdvanceDays = int(v)
		} else if v, ok := m["advanceDays"].(float64); ok {
			req.AdvanceDays = int(v)
		}
		if v, ok := m["reminder_times"].([]interface{}); ok {
			for _, it := range v {
				if s, ok2 := it.(string); ok2 {
					req.ReminderTimes = append(req.ReminderTimes, s)
				}
			}
		} else if v, ok := m["reminderTimes"].([]interface{}); ok {
			for _, it := range v {
				if s, ok2 := it.(string); ok2 {
					req.ReminderTimes = append(req.ReminderTimes, s)
				}
			}
		}
		if v, ok := m["reminder_type"].(string); ok {
			req.ReminderType = v
		} else if v, ok := m["reminderType"].(string); ok {
			req.ReminderType = v
		}
		if v, ok := m["custom_message"].(string); ok {
			req.CustomMessage = v
		} else if v, ok := m["customMessage"].(string); ok {
			req.CustomMessage = v
		}
		observability.LogWarn("CreateReminder strict decode failed, fallback used err=%v", strictErr)
	}

	// 字段校验 (absolute_times 或 reminder_times 二选一至少一个)
	if req.EventID.IsZero() {
		writeErr(http.StatusBadRequest, "field_event_id", "Event ID is required")
		return
	}
	if len(req.ReminderTimes) == 0 && len(req.AbsoluteTimes) == 0 {
		writeErr(http.StatusBadRequest, "field_times", "Either reminder_times or absolute_times is required")
		return
	}
	if req.ReminderType == "" {
		writeErr(http.StatusBadRequest, "field_reminder_type", "Reminder type is required")
		return
	}

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
	reminder, err := reminderService.CreateReminder(context.Background(), objectID, req)
	if err != nil {
		if err.Error() == "event not found" {
			writeErr(http.StatusBadRequest, "event_not_found", "Event not found")
			return
		}
		writeErr(http.StatusInternalServerError, "create_failed", fmt.Sprintf("Failed to create reminder: %v", err))
		return
	}
	observability.LogInfo("CreateReminder success id=%s user=%s event=%s times=%d", reminder.ID.Hex(), userID, reminder.EventID.Hex(), len(reminder.ReminderTimes))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(reminder)
}

// truncateBody 辅助
func truncateBody(b []byte) string {
	s := string(b)
	if len(s) > 500 {
		return s[:500] + "..."
	}
	return s
}

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

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
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

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
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

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
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

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
	response, err := reminderService.ListReminders(context.Background(), objectID, page, pageSize, activeOnly)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list reminders: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// ListRemindersSimple 简化列表（无分页，快速） /api/reminders/simple?active_only=true
func (d *ReminderDeps) ListRemindersSimple(w http.ResponseWriter, r *http.Request) {
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
	activeOnly := r.URL.Query().Get("active_only") == "true"
	limit := 50
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if v, err := strconv.Atoi(ls); err == nil && v > 0 && v <= 100 {
			limit = v
		}
	}
	svc := services.NewReminderService(repository.NewReminderRepository(d.DB))
	list, err := svc.ListSimpleReminders(r.Context(), objectID, activeOnly, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to list simple reminders: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"reminders": list, "count": len(list)})
}

// ToggleReminderActive 反转激活状态
func (d *ReminderDeps) ToggleReminderActive(w http.ResponseWriter, r *http.Request) {
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
	ridStr := mux.Vars(r)["id"]
	rid, err := primitive.ObjectIDFromHex(ridStr)
	if err != nil {
		http.Error(w, "Invalid reminder ID", http.StatusBadRequest)
		return
	}
	svc := services.NewReminderService(repository.NewReminderRepository(d.DB))
	newVal, err := svc.ToggleReminderActive(r.Context(), objectID, rid)
	if err != nil {
		if err.Error() == "reminder not found" {
			http.Error(w, "Reminder not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to toggle: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"id": rid.Hex(), "is_active": newVal})
}

// PreviewReminder 预览提醒计划（不落库） POST /api/reminders/preview
// { "event_id": "...", "advance_days": 0, "reminder_times": ["09:00"] }
func (d *ReminderDeps) PreviewReminder(w http.ResponseWriter, r *http.Request) {
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
	var body struct {
		EventID       string   `json:"event_id"`
		AdvanceDays   int      `json:"advance_days"`
		ReminderTimes []string `json:"reminder_times"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	eid, err := primitive.ObjectIDFromHex(body.EventID)
	if err != nil {
		http.Error(w, "Invalid event_id", http.StatusBadRequest)
		return
	}
	svc := services.NewReminderService(repository.NewReminderRepository(d.DB))
	preview, err := svc.PreviewReminder(r.Context(), uid, eid, body.AdvanceDays, body.ReminderTimes)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to preview: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(preview)
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

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
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
	if userID == "" {
		writeErr(http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeErr(http.StatusBadRequest, "user_id", "Invalid user ID")
		return
	}

	vars := mux.Vars(r)
	reminderIDStr := vars["id"]
	reminderID, err := primitive.ObjectIDFromHex(reminderIDStr)
	if err != nil {
		writeErr(http.StatusBadRequest, "reminder_id", "Invalid reminder ID")
		return
	}

	var req struct {
		SnoozeMinutes int `json:"snooze_minutes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		req.SnoozeMinutes = 60 // fallback
	}
	if req.SnoozeMinutes <= 0 {
		req.SnoozeMinutes = 60
	}

	reminderService := services.NewReminderService(repository.NewReminderRepository(d.DB))
	if err := reminderService.SnoozeReminder(context.Background(), objectID, reminderID, req.SnoozeMinutes); err != nil {
		if err.Error() == "reminder not found" {
			writeErr(http.StatusNotFound, "reminder_not_found", "Reminder not found")
			return
		}
		writeErr(http.StatusInternalServerError, "snooze_failed", fmt.Sprintf("Failed to snooze reminder: %v", err))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"message": "Reminder snoozed successfully", "snooze_minutes": req.SnoozeMinutes})
}

// SetupReminderRoutes 设置提醒路由
func SetupReminderRoutes(r *mux.Router, deps *ReminderDeps) {
	s := r.PathPrefix("/api/reminders").Subrouter()

	// 先注册所有静态/特定前缀路由，避免被通配 {id} 吃掉
	s.Handle("/simple", Auth(http.HandlerFunc(deps.ListRemindersSimple))).Methods(http.MethodGet)
	s.Handle("/upcoming", Auth(http.HandlerFunc(deps.GetUpcomingReminders))).Methods(http.MethodGet)
	s.Handle("/test", Auth(http.HandlerFunc(deps.CreateTestReminder))).Methods(http.MethodPost)

	// 根路径列表 / 创建
	s.Handle("", Auth(http.HandlerFunc(deps.ListReminders))).Methods(http.MethodGet)
	s.Handle("", Auth(http.HandlerFunc(deps.CreateReminder))).Methods(http.MethodPost)
	s.Handle("/preview", Auth(http.HandlerFunc(deps.PreviewReminder))).Methods(http.MethodPost)

	// 使用 24 位十六进制正则确保只匹配合法 ObjectID
	idPattern := "{id:[0-9a-fA-F]{24}}"
	s.Handle("/"+idPattern, Auth(http.HandlerFunc(deps.GetReminder))).Methods(http.MethodGet)
	s.Handle("/"+idPattern, Auth(http.HandlerFunc(deps.UpdateReminder))).Methods(http.MethodPut)
	s.Handle("/"+idPattern, Auth(http.HandlerFunc(deps.DeleteReminder))).Methods(http.MethodDelete)
	s.Handle("/"+idPattern+"/snooze", Auth(http.HandlerFunc(deps.SnoozeReminder))).Methods(http.MethodPost)
	s.Handle("/"+idPattern+"/toggle_active", Auth(http.HandlerFunc(deps.ToggleReminderActive))).Methods(http.MethodPost)
}

// CreateTestReminder 直接创建一个立即或短延迟触发的测试提醒
func (d *ReminderDeps) CreateTestReminder(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user id", http.StatusBadRequest)
		return
	}
	var payload struct {
		EventID      string `json:"event_id"`
		Message      string `json:"message"`
		DelaySeconds int    `json:"delay_seconds"`
		SendEmail    *bool  `json:"send_email"` // 可选: 若提供并为 true 则强制发送邮件
	}
	_ = json.NewDecoder(r.Body).Decode(&payload)
	if payload.Message == "" {
		payload.Message = "测试提醒"
	}
	if payload.DelaySeconds < 0 {
		payload.DelaySeconds = 0
	}
	// 如果未指定事件，尝试选择最近的一个事件
	var eventID primitive.ObjectID
	if payload.EventID != "" {
		if oid, err := primitive.ObjectIDFromHex(payload.EventID); err == nil {
			eventID = oid
		}
	}
	svc := services.NewReminderService(repository.NewReminderRepository(d.DB))
	if eventID.IsZero() {
		// 找一个最近未来事件
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		cur, err := d.DB.Collection("events").Find(ctx, bson.M{"user_id": uid}, options.Find().SetSort(bson.D{{Key: "event_date", Value: 1}}).SetLimit(1))
		if err == nil {
			var ev models.Event
			if cur.Next(ctx) {
				_ = cur.Decode(&ev)
				eventID = ev.ID
			}
			cur.Close(ctx)
		}
		if eventID.IsZero() {
			http.Error(w, "No event available for test", http.StatusBadRequest)
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	reminder, evt, err := svc.CreateImmediateTestReminder(ctx, uid, eventID, payload.Message, payload.DelaySeconds)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create test reminder: %v", err), http.StatusInternalServerError)
		return
	}

	// 判断是否需要立即发送邮件：条件 1) 请求显式 send_email=true; 或 2) 默认策略: 测试提醒都即时尝试邮件
	wantEmail := true
	if payload.SendEmail != nil {
		wantEmail = *payload.SendEmail
	}

	emailSent := false
	var emailErr string
	if wantEmail {
		// 获取用户邮箱
		var userDoc struct {
			Email string `bson:"email"`
		}
		uCtx, uCancel := context.WithTimeout(ctx, 3*time.Second)
		defer uCancel()
		errFind := d.DB.Collection("users").FindOne(uCtx, bson.M{"_id": uid}).Decode(&userDoc)
		if errFind != nil || userDoc.Email == "" {
			emailErr = "user email not found"
		} else {
			// 构造邮件内容（复用 scheduler 风格）
			subject := fmt.Sprintf("TodoIng 测试提醒：%s", evt.Title)
			body := fmt.Sprintf("亲爱的用户,\n\n%s\n\n事件: %s\n时间: %s\n类型: %s\n\n--\nTodoIng 系统即时测试提醒\n",
				payload.Message,
				evt.Title,
				evt.EventDate.Format("2006-01-02 15:04"),
				evt.EventType,
			)
			if errSend := email.SendGeneric(userDoc.Email, subject, body); errSend != nil {
				emailErr = errSend.Error()
				log.Printf("CreateTestReminder immediate email failed user=%s email=%s err=%v", uid.Hex(), userDoc.Email, errSend)
			} else {
				emailSent = true
				log.Printf("CreateTestReminder immediate email sent user=%s to=%s subject=%s", uid.Hex(), userDoc.Email, subject)
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	resp := map[string]interface{}{
		"reminder":   reminder,
		"event":      evt,
		"email_sent": emailSent,
		"note":       "Test reminder created and immediate email attempted",
	}
	if emailErr != "" {
		resp["email_error"] = emailErr
	}
	_ = json.NewEncoder(w).Encode(resp)
}

// TestReminder 计算一个临时提醒的 next_send 不落库
