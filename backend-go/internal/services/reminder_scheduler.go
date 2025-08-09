package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/axfinn/todoIng/backend-go/internal/models"
)

// ReminderScheduler 提醒调度器
type ReminderScheduler struct {
	db              *mongo.Database
	reminderService *ReminderService
	eventService    *EventService
	ticker          *time.Ticker
	stopChan        chan bool
	running         bool
}

// NewReminderScheduler 创建提醒调度器
func NewReminderScheduler(db *mongo.Database) *ReminderScheduler {
	return &ReminderScheduler{
		db:              db,
		reminderService: NewReminderService(db),
		eventService:    NewEventService(db),
		stopChan:        make(chan bool),
		running:         false,
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

	// 获取待发送的提醒
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
	// 这里可以实现WebSocket推送或者保存到通知表
	// 暂时只记录日志
	log.Printf("App notification for user %s: %s", userID.Hex(), message)
	
	// TODO: 实现WebSocket推送或保存到notifications集合
	// 可以考虑使用以下方式：
	// 1. WebSocket实时推送
	// 2. 保存到数据库的notifications集合，前端轮询
	// 3. 使用Redis发布订阅模式
	
	return nil
}

// sendEmailReminder 发送邮件提醒
func (s *ReminderScheduler) sendEmailReminder(ctx context.Context, userID primitive.ObjectID, message string, event models.Event) error {
	// 获取用户邮箱
	userEmail, err := s.getUserEmail(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user email: %w", err)
	}

	if userEmail == "" {
		return fmt.Errorf("user email not found")
	}

	// 生成邮件内容
	subject := fmt.Sprintf("TodoIng 提醒：%s", event.Title)
	body := s.generateEmailBody(message, event)

	// TODO: 集成现有的邮件服务
	// 这里应该调用现有的邮件发送服务
	log.Printf("Email reminder for %s: %s", userEmail, subject)
	log.Printf("Email body: %s", body)
	
	// 暂时只记录日志，实际部署时需要集成邮件服务
	// emailService := email.NewEmailService()
	// return emailService.SendEmail(userEmail, subject, body)
	
	return nil
}

// getUserEmail 获取用户邮箱
func (s *ReminderScheduler) getUserEmail(ctx context.Context, userID primitive.ObjectID) (string, error) {
	// TODO: 从用户集合中获取邮箱
	// 这里应该查询users集合获取用户邮箱
	
	// 暂时返回空字符串，实际部署时需要实现
	return "", nil
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
	_ = userColl // 避免未使用变量错误
	
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
