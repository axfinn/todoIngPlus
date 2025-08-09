package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/axfinn/todoIng/backend-go/internal/services"
)

// DashboardHandler 看板处理器
type DashboardHandler struct {
	taskSortService *services.TaskSortService
	db              *mongo.Database
}

// NewDashboardHandler 创建看板处理器
func NewDashboardHandler(db *mongo.Database) *DashboardHandler {
	return &DashboardHandler{
		taskSortService: services.NewTaskSortService(db),
		db:              db,
	}
}

// GetPriorityTasks 获取优先任务
func (h *DashboardHandler) GetPriorityTasks(w http.ResponseWriter, r *http.Request) {
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

	tasks, err := h.taskSortService.GetPriorityTasks(context.Background(), objectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get priority tasks: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// GetSmartTaskList 获取智能排序任务列表
func (h *DashboardHandler) GetSmartTaskList(w http.ResponseWriter, r *http.Request) {
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

	includeEvents := r.URL.Query().Get("include_events") == "true"

	tasks, err := h.taskSortService.GetSmartTaskList(context.Background(), objectID, includeEvents)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get smart task list: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks":          tasks,
		"total":          len(tasks),
		"include_events": includeEvents,
	})
}

// GetDashboardData 获取看板数据
func (h *DashboardHandler) GetDashboardData(w http.ResponseWriter, r *http.Request) {
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

	dashboard, err := h.taskSortService.GetDashboardData(context.Background(), objectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get dashboard data: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// GetTaskSortConfig 获取任务排序配置
func (h *DashboardHandler) GetTaskSortConfig(w http.ResponseWriter, r *http.Request) {
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

	config, err := h.taskSortService.GetTaskSortConfig(context.Background(), objectID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get sort config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// UpdateTaskSortConfig 更新任务排序配置
func (h *DashboardHandler) UpdateTaskSortConfig(w http.ResponseWriter, r *http.Request) {
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

	var req models.UpdateTaskSortConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	config, err := h.taskSortService.UpdateTaskSortConfig(context.Background(), objectID, req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update sort config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}
