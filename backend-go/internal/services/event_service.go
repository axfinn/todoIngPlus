package services

import (
	"context"
	"errors"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// EventService 精简：仅组合仓储，不直接使用 mongo/bson
// 复杂推进 / 级联 / 聚合 已下沉到 repository.EventRepository

type EventService struct{ repo repository.EventRepository }

func NewEventService(repo repository.EventRepository) *EventService { return &EventService{repo: repo} }

// CreateEvent 构造并插入
func (s *EventService) CreateEvent(ctx context.Context, userID primitive.ObjectID, req models.CreateEventRequest) (*models.Event, error) {
	if s.repo == nil {
		return nil, errors.New("event repo nil")
	}
	e := &models.Event{
		ID:               primitive.NewObjectID(),
		UserID:           userID,
		Title:            req.Title,
		Description:      req.Description,
		EventType:        req.EventType,
		EventDate:        req.EventDate,
		RecurrenceType:   req.RecurrenceType,
		RecurrenceConfig: req.RecurrenceConfig,
		ImportanceLevel:  req.ImportanceLevel,
		Tags:             req.Tags,
		Location:         req.Location,
		IsAllDay:         req.IsAllDay,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if e.EventType == "" {
		e.EventType = "custom"
	}
	if e.RecurrenceType == "" {
		e.RecurrenceType = "none"
	}
	if e.ImportanceLevel == 0 {
		e.ImportanceLevel = 3
	}
	if e.Tags == nil {
		e.Tags = []string{}
	}
	if err := s.repo.Insert(ctx, e); err != nil {
		return nil, err
	}
	return e, nil
}

func (s *EventService) GetEvent(ctx context.Context, userID, eventID primitive.ObjectID) (*models.Event, error) {
	return s.repo.FindByID(ctx, userID, eventID)
}

func (s *EventService) UpdateEvent(ctx context.Context, userID, eventID primitive.ObjectID, req models.UpdateEventRequest) (*models.Event, error) {
	set := map[string]interface{}{}
	if req.Title != nil {
		set["title"] = *req.Title
	}
	if req.Description != nil {
		set["description"] = *req.Description
	}
	if req.EventType != nil {
		set["event_type"] = *req.EventType
	}
	if req.EventDate != nil {
		set["event_date"] = *req.EventDate
	}
	if req.RecurrenceType != nil {
		set["recurrence_type"] = *req.RecurrenceType
	}
	if req.RecurrenceConfig != nil {
		set["recurrence_config"] = req.RecurrenceConfig
	}
	if req.ImportanceLevel != nil {
		set["importance_level"] = *req.ImportanceLevel
	}
	if req.Tags != nil {
		set["tags"] = req.Tags
	}
	if req.Location != nil {
		set["location"] = *req.Location
	}
	if req.IsAllDay != nil {
		set["is_all_day"] = *req.IsAllDay
	}
	if req.IsActive != nil {
		set["is_active"] = *req.IsActive
	}
	if len(set) == 0 {
		return s.GetEvent(ctx, userID, eventID)
	}
	return s.repo.UpdateFields(ctx, userID, eventID, set)
}

func (s *EventService) DeleteEvent(ctx context.Context, userID, eventID primitive.ObjectID) error { return s.repo.Delete(ctx, userID, eventID) }

func (s *EventService) AdvanceEvent(ctx context.Context, userID, eventID primitive.ObjectID, reason string) (*models.Event, error) { return s.repo.Advance(ctx, userID, eventID, reason) }

func (s *EventService) ListEvents(ctx context.Context, userID primitive.ObjectID, page, pageSize int, eventType string, startDate, endDate *time.Time) (*models.EventListResponse, error) {
	return s.repo.ListPaged(ctx, userID, page, pageSize, eventType, startDate, endDate)
}

func (s *EventService) GetUpcomingEvents(ctx context.Context, userID primitive.ObjectID, days int) ([]models.Event, error) {
	return s.repo.ListUpcoming(ctx, userID, days)
}

func (s *EventService) GetCalendarEvents(ctx context.Context, userID primitive.ObjectID, year, month int) (map[string][]models.Event, error) {
	list, err := s.repo.CalendarRange(ctx, userID, year, month)
	if err != nil {
		return nil, err
	}
	m := make(map[string][]models.Event)
	for _, ev := range list {
		k := ev.EventDate.Format("2006-01-02")
		m[k] = append(m[k], ev)
	}
	return m, nil
}

func (s *EventService) SearchEvents(ctx context.Context, userID primitive.ObjectID, keyword string, limit int) ([]models.Event, error) {
	return s.repo.Search(ctx, userID, keyword, limit)
}
