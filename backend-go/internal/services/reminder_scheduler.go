package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/email"
	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	"github.com/axfinn/todoIngPlus/backend-go/internal/repository"
	nHub "github.com/axfinn/todoIngPlus/backend-go/internal/notifications"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

// ReminderScheduler 提醒调度器
type ReminderScheduler struct {
	db              *mongo.Database
	reminderService *ReminderService
	eventService    *EventService
	ticker          *time.Ticker
	stopChan        chan bool
	running         bool
	notificationSvc *NotificationService
	hub             *nHub.Hub
	eventRepo       repository.EventRepository
}

// NewReminderScheduler 创建提醒调度器
func NewReminderScheduler(db *mongo.Database, hub *nHub.Hub) *ReminderScheduler {
	eventRepo := repository.NewEventRepository(db)
	reminderRepo := repository.NewReminderRepository(db)
	return &ReminderScheduler{
		db:              db,
		reminderService: NewReminderService(reminderRepo),
		eventService:    NewEventService(eventRepo),
		eventRepo:       eventRepo,
		stopChan:        make(chan bool),
		running:         false,
		notificationSvc: NewNotificationService(db),
		hub:             hub,
	}
}

// Start 启动调度器
func (s *ReminderScheduler) Start() {
	if s.running {
		log.Println("Reminder scheduler is already running")
		return
	}

	s.running = true
	s.ticker = time.NewTicker(1 * time.Minute) // 每分钟检查一次

	log.Println("Reminder scheduler started")

	go func() {
		for {
			select {
			case <-s.ticker.C:
				s.checkAndSendReminders()
			case <-s.stopChan:
				s.ticker.Stop()
				s.running = false
				log.Println("Reminder scheduler stopped")
				return
			}
		}
	}()
}

// TriggerOnce 手动触发一次检查（异步）
func (s *ReminderScheduler) TriggerOnce() {
	if !s.running { // 若未启动仍允许单次执行
		go s.checkAndSendReminders()
		return
	}
	go s.checkAndSendReminders()
}

// Stop 停止调度器
func (s *ReminderScheduler) Stop() {
	if !s.running {
		return
	}
	s.stopChan <- true
}

// checkAndSendReminders 检查并发送提醒
func (s *ReminderScheduler) checkAndSendReminders() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 1) 事件到点系统时间线记录（精度: 分钟）
	now := time.Now()
	from := now.Add(-1 * time.Minute)
	to := now
	events, errEv := s.eventRepo.ListStartingWindow(ctx, from, to)
	if errEv == nil {
		ecs := NewEventCommentService(s.db)
		for _, ev := range events {
			if ev.LastTriggeredAt != nil && now.Sub(*ev.LastTriggeredAt) < 55*time.Second {
				continue
			}
			content := fmt.Sprintf("系统: 事件开始触发 - %s", ev.Title)
			if c, err := ecs.AddComment(ctx, ev.UserID, ev.ID, models.CreateEventCommentRequest{Content: content, Type: "system", Meta: map[string]string{"kind": "event_start"}}); err != nil {
				log.Printf("timeline event_start comment err: %v", err)
			} else if s.hub != nil {
				// 推送通知用于前端实时刷新（事件时间线）
				n := models.Notification{UserID: ev.UserID, Type: "timeline_event", Message: content, CreatedAt: time.Now(), EventID: &ev.ID, Metadata: map[string]interface{}{"comment_id": c.ID.Hex(), "kind": "event_start"}}
				s.hub.Broadcast(n)
			}
			_ = s.eventRepo.MarkTriggered(ctx, ev.ID, now)
		}
	}

	// 2) 获取待发送的提醒
	pendingReminders, err := s.reminderService.GetPendingReminders(ctx)
	if err != nil {
		log.Printf("Failed to get pending reminders: %v", err)
		return
	}

	if len(pendingReminders) == 0 {
		return
	}

	log.Printf("Found %d pending reminders", len(pendingReminders))

	for _, reminderWithEvent := range pendingReminders {
		if err := s.sendReminder(ctx, reminderWithEvent); err != nil {
			log.Printf("Failed to send reminder %s: %v", reminderWithEvent.ID.Hex(), err)
			continue
		}

		// 发送成功后写入事件时间线系统记录
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("timeline comment panic: %v", r)
				}
			}()
			ecs := NewEventCommentService(s.db)
			content := fmt.Sprintf("系统: 已发送提醒 (%s) - %s", reminderWithEvent.Reminder.ReminderType, reminderWithEvent.Event.Title)
			if c, err := ecs.AddComment(ctx, reminderWithEvent.Reminder.UserID, reminderWithEvent.Event.ID, models.CreateEventCommentRequest{Content: content, Type: "system", Meta: map[string]string{"reminder_id": reminderWithEvent.ID.Hex()}}); err != nil {
				log.Printf("failed to append system comment for event %s: %v", reminderWithEvent.Event.ID.Hex(), err)
			} else if s.hub != nil {
				n := models.Notification{UserID: reminderWithEvent.Reminder.UserID, Type: "timeline_event", Message: content, CreatedAt: time.Now(), EventID: &reminderWithEvent.Event.ID, Metadata: map[string]interface{}{"comment_id": c.ID.Hex(), "reminder_id": reminderWithEvent.ID.Hex(), "kind": "reminder_sent"}}
				s.hub.Broadcast(n)
			}
		}()

		// 标记提醒已发送
		if err := s.reminderService.MarkReminderSent(ctx, reminderWithEvent.ID); err != nil {
			log.Printf("Failed to mark reminder as sent %s: %v", reminderWithEvent.ID.Hex(), err)
		}
	}
}

// sendReminder 发送提醒
func (s *ReminderScheduler) sendReminder(ctx context.Context, reminderWithEvent models.ReminderWithEvent) error {
	reminder := reminderWithEvent.Reminder
	event := reminderWithEvent.Event

	// 生成提醒消息
	message := s.generateReminderMessage(reminder, event)

	// 根据提醒类型发送
	switch reminder.ReminderType {
	case "app":
		return s.sendAppNotification(ctx, reminder.UserID, message, event)
	case "email":
		return s.sendEmailReminder(ctx, reminder.UserID, message, event)
	case "both":
		// 发送应用内通知
		if err := s.sendAppNotification(ctx, reminder.UserID, message, event); err != nil {
			log.Printf("Failed to send app notification: %v", err)
		}
		// 发送邮件提醒
		return s.sendEmailReminder(ctx, reminder.UserID, message, event)
	default:
		return fmt.Errorf("unknown reminder type: %s", reminder.ReminderType)
	}
}

// generateReminderMessage 生成提醒消息
func (s *ReminderScheduler) generateReminderMessage(reminder models.Reminder, event models.Event) string {
	if reminder.CustomMessage != "" {
		return reminder.CustomMessage
	}

	// 计算距离事件的时间
	now := time.Now()
	daysLeft := int(event.EventDate.Sub(now).Hours() / 24)

	var timeDesc string
	if daysLeft == 0 {
		timeDesc = "今天"
	} else if daysLeft == 1 {
		timeDesc = "明天"
	} else if daysLeft > 0 {
		timeDesc = fmt.Sprintf("%d天后", daysLeft)
	} else {
		timeDesc = "已过期"
	}

	eventTypeDesc := s.getEventTypeDescription(event.EventType)

	return fmt.Sprintf("🔔 %s提醒：%s (%s)", eventTypeDesc, event.Title, timeDesc)
}

// getEventTypeDescription 获取事件类型描述
func (s *ReminderScheduler) getEventTypeDescription(eventType string) string {
	switch eventType {
	case "birthday":
		return "生日"
	case "anniversary":
		return "纪念日"
	case "holiday":
		return "节日"
	case "meeting":
		return "会议"
	case "deadline":
		return "截止日期"
	case "custom":
		return "自定义事件"
	default:
		return "事件"
	}
}

// sendAppNotification 发送应用内通知
func (s *ReminderScheduler) sendAppNotification(ctx context.Context, userID primitive.ObjectID, message string, event models.Event) error {
	log.Printf("App notification for user %s: %s", userID.Hex(), message)
	var eventID *primitive.ObjectID
	if !event.ID.IsZero() {
		eid := event.ID
		eventID = &eid
	}
	n, err := s.notificationSvc.Create(ctx, models.NotificationCreate{
		UserID:   userID,
		Type:     "reminder",
		Message:  message,
		EventID:  eventID,
		Metadata: map[string]interface{}{"event_title": event.Title, "event_type": event.EventType},
	})
	if err == nil && s.hub != nil {
		s.hub.Broadcast(n)
	}
	return err
}

// sendEmailReminder 发送邮件提醒
func (s *ReminderScheduler) sendEmailReminder(ctx context.Context, userID primitive.ObjectID, message string, event models.Event) error {
	userEmail, err := s.getUserEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}
	if userEmail == "" {
		return fmt.Errorf("user email not found")
	}
	subject := fmt.Sprintf("TodoIng 提醒：%s", event.Title)
	body := s.generateEmailBody(message, event)
	if err := email.SendGeneric(userEmail, subject, body); err != nil {
		// 不再模拟，直接返回错误（使用登录同一邮箱配置）
		log.Printf("Email send failed %s: %v", userEmail, err)
		return err
	}
	log.Printf("Email sent to %s subject=%s", userEmail, subject)
	if s.notificationSvc != nil && s.hub != nil {
		var eventID *primitive.ObjectID
		if !event.ID.IsZero() {
			eid := event.ID
			eventID = &eid
		}
		n, err2 := s.notificationSvc.Create(ctx, models.NotificationCreate{UserID: userID, Type: "email", Message: fmt.Sprintf("Email sent: %s", subject), EventID: eventID})
		if err2 == nil {
			s.hub.Broadcast(n)
		}
	}
	return nil
}

// getUserEmail 获取用户邮箱
func (s *ReminderScheduler) getUserEmail(ctx context.Context, userID primitive.ObjectID) (string, error) {
	users := s.db.Collection("users")
	var doc struct {
		Email string `bson:"email"`
	}
	err := users.FindOne(ctx, bson.M{"_id": userID}).Decode(&doc)
	if err != nil {
		return "", err
	}
	return doc.Email, nil
}

// generateEmailBody 生成邮件正文
func (s *ReminderScheduler) generateEmailBody(message string, event models.Event) string {
	return fmt.Sprintf(`
亲爱的用户，

%s

事件详情：
- 标题：%s
- 时间：%s
- 类型：%s
- 重要程度：%d/5

请及时关注相关事项。

---
TodoIng 任务管理系统
`, message, event.Title, event.EventDate.Format("2006-01-02 15:04"),
		s.getEventTypeDescription(event.EventType), event.ImportanceLevel)
}

// UpdateEventReminders 更新事件的提醒时间（当事件变更时调用）
func (s *ReminderScheduler) UpdateEventReminders(ctx context.Context, eventID primitive.ObjectID) error {
	// 获取事件信息
	userColl := s.db.Collection("users") // 临时使用，实际需要获取事件的用户ID
	_ = userColl                         // 避免未使用变量错误

	// TODO: 实现提醒时间更新逻辑
	// 1. 根据eventID获取所有相关提醒
	// 2. 重新计算每个提醒的next_send时间
	// 3. 更新数据库

	log.Printf("Updating reminders for event %s", eventID.Hex())
	return nil
}

// IsRunning 检查调度器是否在运行
func (s *ReminderScheduler) IsRunning() bool {
	return s.running
}

// GetStatus 获取调度器状态
func (s *ReminderScheduler) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":    s.running,
		"last_check": time.Now().Format("2006-01-02 15:04:05"),
	}
}
