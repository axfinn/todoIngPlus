package services

import (
	"context"
	"testing"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NOTE: This is a thin smoke test placeholder illustrating intended test structure.
// A real test would spin up a test Mongo (or mock collection). Here we only ensure method signature behaves with nil DB.
func TestComputeUpcomingSignature(t *testing.T) {
	svc := NewUnifiedService(nil)
	dummyUser := primitive.NewObjectID()
	ctx := context.Background()
	if _, _, err := svc.GetUpcoming(ctx, dummyUser, 24, []string{"event"}, 10); err == nil {
		// 期望返回 error，因为 db 为 nil
		// 如果不报错，说明签名或错误处理逻辑可能改变
		t.Errorf("expected error with nil db")
	}
}

// TestBuildUpcomingItems verifies filtering, sorting and limit logic of buildUpcomingItems helper.
func TestBuildUpcomingItems(t *testing.T) {
	now := time.Now()
	end := now.Add(48 * time.Hour)
	// craft events / reminders / priority / normal tasks with overlapping times
	evID := primitive.NewObjectID()
	events := []models.Event{{ID: evID, Title: "EventA", EventDate: now.Add(6 * time.Hour), ImportanceLevel: 3}}
	remID := primitive.NewObjectID()
	reminders := []models.UpcomingReminder{{ID: remID, Message: "ReminderA", ReminderAt: now.Add(12 * time.Hour), Importance: 2}}
	// priority tasks: one earlier, one later
	pt1 := models.PriorityTask{Task: models.Task{ID: "pt1", Title: "PT Early", Deadline: ptrTime(now.Add(3 * time.Hour))}, PriorityScore: 0.9}
	pt2 := models.PriorityTask{Task: models.Task{ID: "pt2", Title: "PT Later", Deadline: ptrTime(now.Add(30 * time.Hour))}, PriorityScore: 0.5}
	priority := []models.PriorityTask{pt2, pt1} // intentionally unsorted to test ordering
	normals := []simpleTask{{ID: "nt1", Title: "Normal", Deadline: ptrTime(now.Add(20 * time.Hour))}}

	// Case 1: no source filter, limit large
	items := buildUpcomingItems(now, end, nil, 0, events, reminders, priority, normals)
	if len(items) != 5 {
		t.Fatalf("expected 5 items got %d", len(items))
	}
	// ordering: by ScheduledAt asc -> PT Early (3h), Event(6h), Reminder(12h), Normal(20h), PT Later(30h)
	if items[0].ID != pt1.ID || items[1].ID != events[0].ID.Hex() || items[2].ID != reminders[0].ID.Hex() || items[3].ID != "nt1" || items[4].ID != pt2.ID {
		t.Errorf("unexpected ordering: %+v", items)
	}
	// Case 2: source filter only event
	itemsEv := buildUpcomingItems(now, end, []string{"event"}, 0, events, reminders, priority, normals)
	if len(itemsEv) != 1 || itemsEv[0].Source != "event" {
		t.Errorf("event filter failed: %+v", itemsEv)
	}
	// Case 3: limit truncation
	limited := buildUpcomingItems(now, end, nil, 3, events, reminders, priority, normals)
	if len(limited) != 3 {
		t.Errorf("limit not applied: %d", len(limited))
	}
	// Ensure first 3 keep ordering from full list
	if limited[0].ID != pt1.ID || limited[1].ID != events[0].ID.Hex() || limited[2].ID != reminders[0].ID.Hex() {
		t.Errorf("limit ordering mismatch: %+v", limited)
	}
}

func ptrTime(ti time.Time) *time.Time { return &ti }
