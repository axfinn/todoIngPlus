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
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/observability"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/services"
)

// EventDeps 事件相关依赖
type EventDeps struct {
	DB *mongo.Database
}

// --- 事件评论 / 时间线 Handlers ---
// CreateEventComment 创建评论
func (d *EventDeps) CreateEventComment(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "user_id", "Invalid user ID")
		return
	}
	vars := mux.Vars(r)
	evIDHex := vars["id"]
	evID, err := primitive.ObjectIDFromHex(evIDHex)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "event_id", "Invalid event ID")
		return
	}
	var req models.CreateEventCommentRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "decode", "Invalid JSON")
		return
	}
	svc := services.NewEventCommentService(d.DB)
	c, err := svc.AddComment(r.Context(), uid, evID, req)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "add_comment", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

// UpdateEventComment 更新普通评论（仅作者 & 10分钟内）
func (d *EventDeps) UpdateEventComment(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "user_id", "Invalid user ID")
		return
	}
	vars := mux.Vars(r)
	cidHex := vars["commentID"]
	cid, err := primitive.ObjectIDFromHex(cidHex)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "comment_id", "Invalid comment ID")
		return
	}
	var req models.UpdateEventCommentRequest
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<16)).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "decode", "Invalid JSON")
		return
	}
	svc := services.NewEventCommentService(d.DB)
	c, uerr := svc.UpdateComment(r.Context(), uid, cid, req)
	if uerr != nil {
		writeJSONError(w, http.StatusBadRequest, "update_comment", uerr.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(c)
}

// ListEventTimeline 获取事件时间线
func (d *EventDeps) ListEventTimeline(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	if _, err := primitive.ObjectIDFromHex(userID); err != nil {
		writeJSONError(w, http.StatusBadRequest, "user_id", "Invalid user ID")
		return
	}
	vars := mux.Vars(r)
	evID, err := primitive.ObjectIDFromHex(vars["id"])
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "event_id", "Invalid event ID")
		return
	}
	limit := 100
	if ls := r.URL.Query().Get("limit"); ls != "" {
		if l, e := strconv.Atoi(ls); e == nil && l > 0 && l <= 200 {
			limit = l
		}
	}
	var beforeOID *primitive.ObjectID
	if b := r.URL.Query().Get("before_id"); b != "" {
		if oid, e := primitive.ObjectIDFromHex(b); e == nil {
			beforeOID = &oid
		}
	}
	svc := services.NewEventCommentService(d.DB)
	items, err := svc.ListTimeline(r.Context(), primitive.NilObjectID, evID, limit, beforeOID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "timeline", err.Error())
		return
	}
	// 若无任何时间线记录，尝试生成一个系统“created”记录（防止前端完全空白 & 兼容历史数据）
	if len(items) == 0 && beforeOID == nil { // 仅第一页才 backfill
		// 读取事件创建时间
		var evDoc struct {
			ID        primitive.ObjectID `bson:"_id"`
			UserID    primitive.ObjectID `bson:"user_id"`
			CreatedAt time.Time          `bson:"created_at"`
			EventDate time.Time          `bson:"event_date"`
		}
		if e := d.DB.Collection("events").FindOne(r.Context(), bson.M{"_id": evID}).Decode(&evDoc); e == nil {
			// 插入一条 system comment
			newID := primitive.NewObjectID()
			_, _ = d.DB.Collection("event_comments").InsertOne(r.Context(), bson.M{
				"_id":        newID,
				"event_id":   evID,
				"user_id":    evDoc.UserID,
				"type":       "system",
				"content":    "created",
				"meta":       bson.M{"action": "create", "date": evDoc.EventDate.Format(time.RFC3339)},
				"created_at": evDoc.CreatedAt,
				"updated_at": evDoc.CreatedAt,
			})
			// 重新拉取
			items, _ = svc.ListTimeline(r.Context(), primitive.NilObjectID, evID, limit, beforeOID)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"items": items, "count": len(items)})
}

// DeleteEventComment 删除评论
func (d *EventDeps) DeleteEventComment(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	uid, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "user_id", "Invalid user ID")
		return
	}
	vars := mux.Vars(r)
	cidHex := vars["commentID"]
	cid, err := primitive.ObjectIDFromHex(cidHex)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "comment_id", "Invalid comment ID")
		return
	}
	svc := services.NewEventCommentService(d.DB)
	if err := svc.DeleteComment(r.Context(), uid, cid); err != nil {
		writeJSONError(w, http.StatusBadRequest, "delete_comment", err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ReminderDeps 提醒相关依赖
type ReminderDeps struct {
	DB *mongo.Database
}

// DashboardDeps 看板相关依赖
type DashboardDeps struct {
	DB *mongo.Database
}

// 通用 JSON 错误响应结构
type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func writeJSONError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiError{Code: code, Message: message})
}

// CreateEvent 创建事件 (支持多格式 event_date)
func (d *EventDeps) CreateEvent(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	if userID == "" {
		writeJSONError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	if ct := r.Header.Get("Content-Type"); ct != "" {
		lc := strings.ToLower(ct)
		if !strings.HasPrefix(lc, "application/json") {
			writeJSONError(w, http.StatusBadRequest, "content_type", "Unsupported Content-Type (expect application/json)")
			return
		}
	}
	objectID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "user_id", "Invalid user ID (legacy account). Please re-register to use events/reminders.")
		return
	}

	// 原始载荷用字符串接收 event_date 以便自定义解析
	var raw struct {
		Title            string                 `json:"title"`
		Description      string                 `json:"description"`
		EventType        string                 `json:"event_type"`
		EventDate        string                 `json:"event_date"`
		RawEventDate     string                 `json:"raw_event_date"`
		RecurrenceType   string                 `json:"recurrence_type"`
		RecurrenceConfig map[string]interface{} `json:"recurrence_config"`
		ImportanceLevel  int                    `json:"importance_level"`
		Tags             []string               `json:"tags"`
		Location         string                 `json:"location"`
		IsAllDay         bool                   `json:"is_all_day"`
	}
	bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	trimmed := strings.TrimSpace(string(bodyBytes))
	if trimmed == "" {
		writeJSONError(w, http.StatusBadRequest, "empty_body", "Request body is empty")
		return
	}

	// 尝试严格解析
	strictErr := func() error {
		dec := json.NewDecoder(strings.NewReader(trimmed))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&raw); err != nil {
			return err
		}
		if dec.More() {
			return errors.New("multiple JSON objects")
		}
		return nil
	}()

	if strictErr != nil {
		// 宽松回退：解析为 map 再映射字段，忽略未知字段
		var m map[string]interface{}
		if err2 := json.Unmarshal([]byte(trimmed), &m); err2 != nil {
			trunc := trimmed
			if len(trunc) > 500 {
				trunc = trunc[:500] + "..."
			}
			observability.LogWarn("CreateEvent decode both strict & loose fail strict=%v loose=%v body=%s", strictErr, err2, trunc)
			writeJSONError(w, http.StatusBadRequest, "json_decode", "Invalid request body: "+strictErr.Error())
			return
		}
		mapAssignStr := func(key string, dst *string) {
			if v, ok := m[key]; ok {
				if s, ok2 := v.(string); ok2 {
					*dst = s
				}
			}
		}
		mapAssignInt := func(key string, dst *int) {
			if v, ok := m[key]; ok {
				switch val := v.(type) {
				case float64:
					*dst = int(val)
				case int:
					*dst = val
				}
			}
		}
		mapAssignBool := func(key string, dst *bool) {
			if v, ok := m[key]; ok {
				if b, ok2 := v.(bool); ok2 {
					*dst = b
				}
			}
		}
		mapAssignStr("title", &raw.Title)
		mapAssignStr("description", &raw.Description)
		mapAssignStr("event_type", &raw.EventType)
		if raw.EventType == "" {
			mapAssignStr("eventType", &raw.EventType)
		}
		mapAssignStr("event_date", &raw.EventDate)
		if raw.EventDate == "" {
			mapAssignStr("eventDate", &raw.EventDate)
		}
		if raw.RawEventDate == "" {
			mapAssignStr("raw_event_date", &raw.RawEventDate)
		}
		mapAssignStr("recurrence_type", &raw.RecurrenceType)
		if raw.RecurrenceType == "" {
			mapAssignStr("recurrenceType", &raw.RecurrenceType)
		}
		if v, ok := m["recurrence_config"]; ok {
			if mm, ok2 := v.(map[string]interface{}); ok2 {
				raw.RecurrenceConfig = mm
			}
		}
		mapAssignInt("importance_level", &raw.ImportanceLevel)
		if raw.ImportanceLevel == 0 {
			mapAssignInt("importanceLevel", &raw.ImportanceLevel)
		}
		mapAssignStr("location", &raw.Location)
		mapAssignBool("is_all_day", &raw.IsAllDay)
		if !raw.IsAllDay {
			mapAssignBool("isAllDay", &raw.IsAllDay)
		}
		// tags 处理
		if v, ok := m["tags"]; ok {
			if arr, ok2 := v.([]interface{}); ok2 {
				for _, el := range arr {
					if s, ok3 := el.(string); ok3 {
						raw.Tags = append(raw.Tags, s)
					}
				}
			}
		}
		observability.LogWarn("CreateEvent strict decode failed, used fallback mapping error=%v", strictErr)
	}

	if raw.Title == "" {
		writeJSONError(w, http.StatusBadRequest, "field_title", "Title is required")
		return
	}
	if raw.EventType == "" {
		writeJSONError(w, http.StatusBadRequest, "field_event_type", "Event type is required")
		return
	}
	if raw.EventDate == "" && raw.RawEventDate != "" {
		raw.EventDate = raw.RawEventDate
	}
	if raw.EventDate == "" {
		writeJSONError(w, http.StatusBadRequest, "field_event_date", "Event date is required")
		return
	}

	parsedDate, err := parseFlexibleEventDate(raw.EventDate, raw.IsAllDay)
	if err != nil {
		if len(raw.EventDate) == 16 && raw.EventDate[10] == 'T' {
			if t2, e2 := time.ParseInLocation("2006-01-02T15:04", raw.EventDate, time.Local); e2 == nil {
				parsedDate = t2.UTC()
				err = nil
			}
		}
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "date_parse", "Invalid event_date format: "+err.Error())
			return
		}
	}

	req := models.CreateEventRequest{
		Title:            raw.Title,
		Description:      raw.Description,
		EventType:        raw.EventType,
		EventDate:        parsedDate,
		RecurrenceType:   raw.RecurrenceType,
		RecurrenceConfig: raw.RecurrenceConfig,
		ImportanceLevel:  raw.ImportanceLevel,
		Tags:             raw.Tags,
		Location:         raw.Location,
		IsAllDay:         raw.IsAllDay,
	}

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
	event, err := eventService.CreateEvent(context.Background(), objectID, req)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "create_failed", fmt.Sprintf("Failed to create event: %v", err))
		return
	}
	observability.LogInfo("CreateEvent success id=%s user=%s title=%s date=%s", event.ID.Hex(), userID, event.Title, event.EventDate.Format(time.RFC3339))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(event)
}

// parseFlexibleDate 支持多种输入格式：
//   - RFC3339 (含毫秒) 2025-08-10T12:30:00Z
//   - 2025-08-10T12:30  (datetime-local, 视为本地时区再转 UTC)
//   - 2025-08-10 12:30  (空格分隔)
//   - 2025-08-10        (仅日期, 若 isAllDay=true 则使用 00:00)
func parseFlexibleEventDate(s string, isAllDay bool) (time.Time, error) {
	// 支持：
	//  - RFC3339 / RFC3339Nano (含毫秒或纳秒)
	//  - YYYY-MM-DDTHH:MM (HTML datetime-local)
	//  - YYYY-MM-DD HH:MM (空格)
	//  - YYYY-MM-DD (仅日期, 根据 isAllDay 补 00:00)
	layouts := []string{time.RFC3339Nano, time.RFC3339, "2006-01-02T15:04", "2006-01-02 15:04"}
	if len(s) == 10 { // 只有日期
		if isAllDay {
			s = s + "T00:00"
		} else {
			s = s + "T00:00"
		}
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	// 再尝试加上本地时区解释 datetime-local
	if matchLen := len(s); matchLen == 16 { // 形如 2006-01-02T15:04
		if lt, err := time.ParseInLocation("2006-01-02T15:04", s, time.Local); err == nil {
			return lt.UTC(), nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported format (expect RFC3339 or 'YYYY-MM-DDTHH:MM')")
}

// GetEvent 获取单个事件
func (d *EventDeps) GetEvent(w http.ResponseWriter, r *http.Request) {
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
	eventIDStr := vars["id"]
	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
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

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
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

// AdvanceEvent 推进事件到下一周期或完成一次性事件
func (d *EventDeps) AdvanceEvent(w http.ResponseWriter, r *http.Request) {
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
	vars := mux.Vars(r)
	eventIDStr := vars["id"]
	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}
	// 可选解析 body reason
	var body struct {
		Reason string `json:"reason"`
	}
	if r.Body != nil {
		dec := json.NewDecoder(r.Body)
		dec.DisallowUnknownFields()
		_ = dec.Decode(&body) // 忽略解析错误（可能无 body）
	}
	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
	event, err := eventService.AdvanceEvent(context.Background(), objectID, eventID, body.Reason)
	if err != nil {
		if err.Error() == "event not found" {
			http.Error(w, "Event not found", http.StatusNotFound)
			return
		}
		http.Error(w, fmt.Sprintf("Failed to advance event: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event)
}

// DeleteEvent 删除事件
func (d *EventDeps) DeleteEvent(w http.ResponseWriter, r *http.Request) {
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
	eventIDStr := vars["id"]
	eventID, err := primitive.ObjectIDFromHex(eventIDStr)
	if err != nil {
		http.Error(w, "Invalid event ID", http.StatusBadRequest)
		return
	}

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
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

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
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

	days := 7
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 30 {
			days = d
		}
	}

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
	events, err := eventService.GetUpcomingEvents(context.Background(), objectID, days)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get upcoming events: %v", err), http.StatusInternalServerError)
		return
	}
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
func (d *EventDeps) GetCalendarEvents(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
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

// GetEventOptions 获取事件下拉选项（精简字段，用于提醒创建等）
func (d *EventDeps) GetEventOptions(w http.ResponseWriter, r *http.Request) {
	userID := GetUserID(r)
	observability.LogWarn("GetEventOptions enter remote=%s userID=%s", r.RemoteAddr, userID)
	if userID == "" {
		observability.LogWarn("GetEventOptions no user id ctx remote=%s", r.RemoteAddr)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	objectID, err := primitive.ObjectIDFromHex(userID)
	legacy := false
	if err != nil {
		// 兼容旧账号（非 ObjectID），返回空列表并标记 legacy_user，避免前端报错
		legacy = true
		observability.LogWarn("GetEventOptions legacy user id=%s", userID)
	}

	type option struct {
		ID         primitive.ObjectID `json:"id"`
		Title      string             `json:"title"`
		EventDate  time.Time          `json:"event_date"`
		EventType  string             `json:"event_type"`
		Importance int                `json:"importance_level"`
		IsAllDay   bool               `json:"is_all_day"`
	}
	opts := []option{}
	if !legacy {
		eventService := services.NewEventService(repository.NewEventRepository(d.DB))
		resp, err := eventService.ListEvents(context.Background(), objectID, 1, 200, "", nil, nil)
		if err != nil {
			observability.LogWarn("GetEventOptions list error user=%s err=%v", userID, err)
			http.Error(w, fmt.Sprintf("Failed to list events: %v", err), http.StatusInternalServerError)
			return
		}
		opts = make([]option, 0, len(resp.Events))
		for _, e := range resp.Events {
			opts = append(opts, option{ID: e.ID, Title: e.Title, EventDate: e.EventDate, EventType: e.EventType, Importance: e.ImportanceLevel, IsAllDay: e.IsAllDay})
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"data": opts, "events": opts, "total": len(opts), "legacy_user": legacy})
	observability.LogWarn("GetEventOptions success legacy=%v count=%d user=%s", legacy, len(opts), userID)
}

// SearchEvents 搜索事件
func (d *EventDeps) SearchEvents(w http.ResponseWriter, r *http.Request) {
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

	eventService := services.NewEventService(repository.NewEventRepository(d.DB))
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

	// 静态/特定功能路由需放在动态 {id} 之前，避免被错误匹配 (如 /options 被 /{id} 捕获)
	s.Handle("/upcoming", Auth(http.HandlerFunc(deps.GetUpcomingEvents))).Methods(http.MethodGet)
	s.Handle("/calendar", Auth(http.HandlerFunc(deps.GetCalendarEvents))).Methods(http.MethodGet)
	s.Handle("/options", Auth(http.HandlerFunc(deps.GetEventOptions))).Methods(http.MethodGet)
	s.Handle("/search", Auth(http.HandlerFunc(deps.SearchEvents))).Methods(http.MethodGet)

	// 事件列表与创建
	s.Handle("", Auth(http.HandlerFunc(deps.ListEvents))).Methods(http.MethodGet)
	s.Handle("", Auth(http.HandlerFunc(deps.CreateEvent))).Methods(http.MethodPost)

	// 使用正则限制 id 为 24 位 hex，防止 /options 等被误判
	s.Handle("/{id:[0-9a-fA-F]{24}}", Auth(http.HandlerFunc(deps.GetEvent))).Methods(http.MethodGet)
	s.Handle("/{id:[0-9a-fA-F]{24}}", Auth(http.HandlerFunc(deps.UpdateEvent))).Methods(http.MethodPut)
	s.Handle("/{id:[0-9a-fA-F]{24}}", Auth(http.HandlerFunc(deps.DeleteEvent))).Methods(http.MethodDelete)
	// 推进/完成
	s.Handle("/{id:[0-9a-fA-F]{24}}/advance", Auth(http.HandlerFunc(deps.AdvanceEvent))).Methods(http.MethodPost)

	// 评论 / 时间线 (挂在 /api/events/{id}/comments ... )
	s.Handle("/{id:[0-9a-fA-F]{24}}/comments", Auth(http.HandlerFunc(deps.CreateEventComment))).Methods(http.MethodPost)
	s.Handle("/{id:[0-9a-fA-F]{24}}/timeline", Auth(http.HandlerFunc(deps.ListEventTimeline))).Methods(http.MethodGet)
	// 删除评论单独路由（无需 eventID 冗余匹配）
	s.Handle("/comments/{commentID:[0-9a-fA-F]{24}}", Auth(http.HandlerFunc(deps.DeleteEventComment))).Methods(http.MethodDelete)
	// 更新评论
	s.Handle("/comments/{commentID:[0-9a-fA-F]{24}}", Auth(http.HandlerFunc(deps.UpdateEventComment))).Methods(http.MethodPut)
}
