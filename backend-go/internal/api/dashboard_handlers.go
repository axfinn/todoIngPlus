/*
 * @Author: error: error: git config user.name & please set dead value or install git && error: git config user.email & please set dead value or install git & please set dead value or install git
 * @Date: 2025-08-10 02:58:16
 * @LastEditors: error: error: git config user.name & please set dead value or install git && error: git config user.email & please set dead value or install git & please set dead value or install git
 * @LastEditTime: 2025-08-10 20:43:45
 * @FilePath: /todoIngPlus/backend-go/internal/api/dashboard_handlers.go
 * @Description: 这是默认设置,请设置`customMade`, 打开koroFileHeader查看配置 进行设置: https://github.com/OBKoro1/koro1FileHeader/wiki/%E9%85%8D%E7%BD%AE
 */
package api

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/axfinn/todoIng/backend-go/internal/services"
)

// GetDashboardData 获取仪表板数据
func (d *DashboardDeps) GetDashboardData(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	taskSortService := services.NewTaskSortService(d.DB)
	dashboardData, err := taskSortService.GetDashboardData(context.Background(), userObjID)
	if err != nil {
		http.Error(w, "Failed to get dashboard data: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboardData)
}

// GetPriorityTasks 按优先级获取任务列表
func (d *DashboardDeps) GetPriorityTasks(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	taskSortService := services.NewTaskSortService(d.DB)
	tasks, err := taskSortService.GetPriorityTasks(context.Background(), userObjID)
	if err != nil {
		http.Error(w, "Failed to get priority tasks: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tasks": tasks,
		"total": len(tasks),
	})
}

// UpdateTaskSortConfig 更新任务排序配置
func (d *DashboardDeps) UpdateTaskSortConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	var req models.UpdateTaskSortConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	taskSortService := services.NewTaskSortService(d.DB)
	config, err := taskSortService.UpdateTaskSortConfig(context.Background(), userObjID, req)
	if err != nil {
		http.Error(w, "Failed to update task sort config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// GetTaskSortConfig 获取任务排序配置
func (d *DashboardDeps) GetTaskSortConfig(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userID").(string)
	if !ok {
		http.Error(w, "User not authenticated", http.StatusUnauthorized)
		return
	}

	userObjID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		http.Error(w, "Invalid user ID", http.StatusBadRequest)
		return
	}

	taskSortService := services.NewTaskSortService(d.DB)
	config, err := taskSortService.GetTaskSortConfig(context.Background(), userObjID)
	if err != nil {
		http.Error(w, "Failed to get task sort config: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(config)
}

// SetupDashboardRoutes 设置仪表板相关路由
func SetupDashboardRoutes(r *mux.Router, deps *DashboardDeps) {
	dashboard := r.PathPrefix("/api/dashboard").Subrouter()

	// 应用认证中间件
	dashboard.Use(Auth)

	// 仪表板数据
	dashboard.HandleFunc("", deps.GetDashboardData).Methods("GET")
	dashboard.HandleFunc("/", deps.GetDashboardData).Methods("GET")

	// 按优先级获取任务
	dashboard.HandleFunc("/tasks", deps.GetPriorityTasks).Methods("GET")

	// 任务排序配置
	dashboard.HandleFunc("/config", deps.GetTaskSortConfig).Methods("GET")
	dashboard.HandleFunc("/config", deps.UpdateTaskSortConfig).Methods("PUT")
}
