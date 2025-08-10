package api

import (
    "context"
    "encoding/json"
    "net/http"
    "time"

    "go.mongodb.org/mongo-driver/bson/primitive"
    "go.mongodb.org/mongo-driver/mongo"
    "github.com/gorilla/mux"

    "github.com/axfinn/todoIng/backend-go/internal/models"
    "github.com/axfinn/todoIng/backend-go/internal/services"
)

type UnifiedDeps struct { DB *mongo.Database }

// GetUpcomingUnified 获取统一聚合
func (d *UnifiedDeps) GetUpcomingUnified(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value("userID").(string)
    if !ok { http.Error(w, "User not authenticated", http.StatusUnauthorized); return }
    objectID, err := primitive.ObjectIDFromHex(userID)
    if err != nil { http.Error(w, "Invalid user ID", http.StatusBadRequest); return }

    parsed := parseUpcomingParams(r.URL.Query())
    svc := services.NewUnifiedService(d.DB)
    items, err := svc.GetUpcoming(context.Background(), objectID, parsed.Hours, parsed.Sources, parsed.Limit)
    if err != nil { http.Error(w, "Failed to get unified items: "+err.Error(), http.StatusInternalServerError); return }

    // 统计分布
    stats := &models.UnifiedUpcomingStats{}
    for _, it := range items {
        switch it.Source { case "task": stats.Tasks++; case "event": stats.Events++; case "reminder": stats.Reminders++; }
    }
    resp := models.UnifiedUpcomingResponse{Items: items, Hours: parsed.Hours, Total: len(items), ServerTimestamp: time.Now().Unix(), Stats: stats}
    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(resp)
}

// GetUnifiedCalendar 获取统一日历
func (d *UnifiedDeps) GetUnifiedCalendar(w http.ResponseWriter, r *http.Request) {
    userID, ok := r.Context().Value("userID").(string)
    if !ok { http.Error(w, "User not authenticated", http.StatusUnauthorized); return }
    objectID, err := primitive.ObjectIDFromHex(userID)
    if err != nil { http.Error(w, "Invalid user ID", http.StatusBadRequest); return }

    now := time.Now()
    parsedCal := parseCalendarParams(r.URL.Query(), now.Year(), int(now.Month()))

    svc := services.NewUnifiedService(d.DB)
    cal, err := svc.GetCalendar(context.Background(), objectID, parsedCal.Year, parsedCal.Month)
    if err != nil { http.Error(w, "Failed to get calendar: "+err.Error(), http.StatusInternalServerError); return }

    // 源过滤
    if len(parsedCal.Sources) > 0 {
        sf := map[string]bool{}
        for _, s := range parsedCal.Sources { sf[s] = true }
        for day, arr := range cal.Days {
            filtered := make([]models.UnifiedCalendarItem, 0, len(arr))
            for _, it := range arr { if sf[it.Source] { filtered = append(filtered, it) } }
            cal.Days[day] = filtered
        }
    }

    // limit 截断（按天遍历，保持天内顺序）
    if parsedCal.Limit > 0 {
        collected := 0
        for day := range cal.Days {
            items := cal.Days[day]
            if collected+len(items) <= parsedCal.Limit {
                collected += len(items)
                continue
            }
            remain := parsedCal.Limit - collected
            if remain < 0 { remain = 0 }
            if remain < len(items) { cal.Days[day] = items[:remain] }
            collected = parsedCal.Limit
            if collected >= parsedCal.Limit { break }
        }
    }

    // 重新计算 total & 服务器时间戳
    total := 0
    for _, arr := range cal.Days { total += len(arr) }
    cal.Total = total
    cal.ServerTimestamp = time.Now().Unix()

    w.Header().Set("Content-Type", "application/json")
    _ = json.NewEncoder(w).Encode(cal)
}

// SetupUnifiedRoutes 注册统一聚合路由
func SetupUnifiedRoutes(r *mux.Router, deps *UnifiedDeps) {
    s := r.PathPrefix("/api/unified").Subrouter()
    s.Handle("/upcoming", Auth(http.HandlerFunc(deps.GetUpcomingUnified))).Methods(http.MethodGet)
    s.Handle("/calendar", Auth(http.HandlerFunc(deps.GetUnifiedCalendar))).Methods(http.MethodGet)
}
