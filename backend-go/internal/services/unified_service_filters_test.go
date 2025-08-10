package services

import (
	"testing"
	"time"

	"github.com/axfinn/todoIng/backend-go/internal/models"
	"github.com/stretchr/testify/require"
)

// TestBuildUpcomingItems_TaskScenarios 验证任务过滤/排序/去重 & 逾期/无日期逻辑
func TestBuildUpcomingItems_TaskScenarios(t *testing.T) {
	now := time.Date(2025, 8, 10, 10, 0, 0, 0, time.UTC)
	end := now.Add(72 * time.Hour)

	// 构造：
	// 1. 优先任务 A (deadline 明天)
	// 2. 普通任务 B (scheduledDate 两天后)
	// 3. 逾期任务 C (deadline 昨天) —— 应保留
	// 4. 无日期任务 D (占位 now, Unscheduled)
	// 5. 重复 ID 在 priority 与 normal 中 (E) —— normal 应被去重
	tomorrow := now.Add(24 * time.Hour)
	twoDays := now.Add(48 * time.Hour)
	yesterday := now.Add(-24 * time.Hour)
	priority := []models.PriorityTask{
		{Task: models.Task{ID: "A", Title: "A", Deadline: &tomorrow}, PriorityScore: 10},
		{Task: models.Task{ID: "E", Title: "E", Deadline: &tomorrow}, PriorityScore: 5},
	}
	normal := []simpleTask{{ID: "B", Title: "B", ScheduledDate: &twoDays}, {ID: "C", Title: "C", Deadline: &yesterday}, {ID: "D", Title: "D", ScheduledDate: &now, Unscheduled: true}, {ID: "E", Title: "E", Deadline: &tomorrow}}

	items := buildUpcomingItems(now, end, nil, 0, nil, nil, priority, normal)
	// 期望包含 A,B,C,D (E 只出现一次且来自 priority)
	ids := map[string]int{}
	unscheduledCount := 0
	for _, it := range items {
		ids[it.ID]++
		if it.ID == "D" {
			require.True(t, it.IsUnscheduled, "D 应标记 is_unscheduled")
		}
		if it.IsUnscheduled {
			unscheduledCount++
		}
	}
	require.Equal(t, 1, ids["A"], "A once")
	require.Equal(t, 1, ids["B"], "B once")
	require.Equal(t, 1, ids["C"], "C once")
	require.Equal(t, 1, ids["D"], "D once")
	require.Equal(t, 1, ids["E"], "E 去重后一次")
	require.GreaterOrEqual(t, unscheduledCount, 1)
}
