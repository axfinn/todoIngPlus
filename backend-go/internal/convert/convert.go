package convert

import (
	"fmt"
	"time"

	"github.com/axfinn/todoIngPlus/backend-go/internal/models"
	pb "github.com/axfinn/todoIngPlus/backend-go/pkg/api/v1"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// 这样可以保持现有代码正常工作

// UserToProto 将内部用户模型转换为 protobuf 模型
func UserToProto(user *models.User) *pb.User {
	if user == nil {
		return nil
	}

	return &pb.User{
		Id:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: timestamppb.New(user.CreatedAt),
		UpdatedAt: timestamppb.New(user.CreatedAt), // 使用 CreatedAt 因为模型中没有 UpdatedAt
	}
}

// ProtoToUser 将 protobuf 用户模型转换为内部模型
func ProtoToUser(pbUser *pb.User) *models.User {
	if pbUser == nil {
		return nil
	}

	user := &models.User{
		Username: pbUser.Username,
		Email:    pbUser.Email,
	}

	if pbUser.CreatedAt != nil {
		user.CreatedAt = pbUser.CreatedAt.AsTime()
	}

	return user
}

// TaskStatusToProto 将内部任务状态转换为 protobuf 状态
func TaskStatusToProto(status string) pb.TaskStatus {
	switch status {
	case "todo":
		return pb.TaskStatus_TASK_STATUS_TODO
	case "in_progress":
		return pb.TaskStatus_TASK_STATUS_IN_PROGRESS
	case "done":
		return pb.TaskStatus_TASK_STATUS_DONE
	default:
		return pb.TaskStatus_TASK_STATUS_UNSPECIFIED
	}
}

// ProtoToTaskStatus 将 protobuf 状态转换为内部任务状态
func ProtoToTaskStatus(status pb.TaskStatus) string {
	switch status {
	case pb.TaskStatus_TASK_STATUS_TODO:
		return "todo"
	case pb.TaskStatus_TASK_STATUS_IN_PROGRESS:
		return "in_progress"
	case pb.TaskStatus_TASK_STATUS_DONE:
		return "done"
	default:
		return "todo"
	}
}

// TaskPriorityToProto 将内部任务优先级转换为 protobuf 优先级
func TaskPriorityToProto(priority string) pb.TaskPriority {
	switch priority {
	case "low":
		return pb.TaskPriority_TASK_PRIORITY_LOW
	case "medium":
		return pb.TaskPriority_TASK_PRIORITY_MEDIUM
	case "high":
		return pb.TaskPriority_TASK_PRIORITY_HIGH
	default:
		return pb.TaskPriority_TASK_PRIORITY_UNSPECIFIED
	}
}

// ProtoToTaskPriority 将 protobuf 优先级转换为内部任务优先级
func ProtoToTaskPriority(priority pb.TaskPriority) string {
	switch priority {
	case pb.TaskPriority_TASK_PRIORITY_LOW:
		return "low"
	case pb.TaskPriority_TASK_PRIORITY_MEDIUM:
		return "medium"
	case pb.TaskPriority_TASK_PRIORITY_HIGH:
		return "high"
	default:
		return "medium"
	}
}

// TaskComment conversions (HTTP -> Proto)
func taskCommentsToProto(cs []models.Comment) []*pb.TaskComment {
	out := make([]*pb.TaskComment, 0, len(cs))
	for _, c := range cs {
		out = append(out, &pb.TaskComment{Text: c.Text, CreatedBy: c.CreatedBy, CreatedAt: timestamppb.New(c.CreatedAt)})
	}
	return out
}

// TaskToProto 将内部任务模型转换为 protobuf 模型 (扩展字段)
func TaskToProto(task *models.Task) *pb.Task {
	if task == nil {
		return nil
	}

	var deadlineTS *timestamppb.Timestamp
	if task.Deadline != nil && !task.Deadline.IsZero() {
		deadlineTS = timestamppb.New(*task.Deadline)
	}
	var scheduledTS *timestamppb.Timestamp
	if task.ScheduledDate != nil && !task.ScheduledDate.IsZero() {
		scheduledTS = timestamppb.New(*task.ScheduledDate)
	}

	return &pb.Task{
		Id:            task.ID,
		Title:         task.Title,
		Description:   task.Description,
		Status:        TaskStatusToProto(task.Status),
		Priority:      TaskPriorityToProto(task.Priority),
		Deadline:      deadlineTS,
		ScheduledDate: scheduledTS,
		UserId:        task.CreatedBy, // 使用 CreatedBy 作为 UserId
		CreatedAt:     timestamppb.New(task.CreatedAt),
		UpdatedAt:     timestamppb.New(task.UpdatedAt),
		Assignee: func() string {
			if task.Assignee != nil {
				return *task.Assignee
			}
			return ""
		}(),
		Comments: taskCommentsToProto(task.Comments),
	}
}

// ReportTypeToProto 将内部报表类型转换为 protobuf 类型
func ReportTypeToProto(reportType string) pb.ReportType {
	switch reportType {
	case "daily":
		return pb.ReportType_REPORT_TYPE_DAILY
	case "weekly":
		return pb.ReportType_REPORT_TYPE_WEEKLY
	case "monthly":
		return pb.ReportType_REPORT_TYPE_MONTHLY
	default:
		return pb.ReportType_REPORT_TYPE_UNSPECIFIED
	}
}

// ProtoToReportType 将 protobuf 类型转换为内部报表类型
func ProtoToReportType(reportType pb.ReportType) string {
	switch reportType {
	case pb.ReportType_REPORT_TYPE_DAILY:
		return "daily"
	case pb.ReportType_REPORT_TYPE_WEEKLY:
		return "weekly"
	case pb.ReportType_REPORT_TYPE_MONTHLY:
		return "monthly"
	default:
		return "daily"
	}
}

// ReportToProto 将内部报表模型转换为 protobuf 模型
func ReportToProto(report *models.Report) *pb.Report {
	if report == nil {
		return nil
	}

	stats := &pb.ReportStats{
		TotalTasks:      int32(report.Statistics.TotalTasks),
		CompletedTasks:  int32(report.Statistics.CompletedTasks),
		InProgressTasks: int32(report.Statistics.InProgressTasks),
		PendingTasks:    int32(report.Statistics.TotalTasks - report.Statistics.CompletedTasks - report.Statistics.InProgressTasks),
		CompletionRate:  float64(report.Statistics.CompletionRate),
		OverdueTasks:    int32(report.Statistics.OverdueTasks),
	}

	return &pb.Report{
		Id:        report.ID,
		Title:     report.Title,
		Type:      ReportTypeToProto(report.Type),
		StartDate: timestamppb.New(report.CreatedAt), // 使用 CreatedAt 作为开始时间
		EndDate:   timestamppb.New(report.UpdatedAt), // 使用 UpdatedAt 作为结束时间
		UserId:    report.UserID,
		CreatedAt: timestamppb.New(report.CreatedAt),
		Stats:     stats,
		Period:    report.Period,
		Content:   report.Content,
		PolishedContent: func() string {
			if report.PolishedContent != nil {
				return *report.PolishedContent
			}
			return ""
		}(),
	}
}

// TaskComment conversions (Proto -> Model)
func protoTaskCommentsToModel(cs []*pb.TaskComment) []models.Comment {
	out := make([]models.Comment, 0, len(cs))
	for _, c := range cs {
		if c == nil {
			continue
		}
		createdAt := time.Now()
		if c.CreatedAt != nil {
			createdAt = c.CreatedAt.AsTime()
		}
		out = append(out, models.Comment{Text: c.Text, CreatedBy: c.CreatedBy, CreatedAt: createdAt})
	}
	return out
}

// ProtoToTask 增强: 反向填充新字段
func ProtoToTask(pbTask *pb.Task) *models.Task {
	if pbTask == nil {
		return nil
	}
	m := &models.Task{
		Title:       pbTask.Title,
		Description: pbTask.Description,
		Status:      ProtoToTaskStatus(pbTask.Status),
		Priority:    ProtoToTaskPriority(pbTask.Priority),
		CreatedBy:   pbTask.UserId,
		Comments:    protoTaskCommentsToModel(pbTask.Comments),
	}
	if pbTask.Deadline != nil {
		d := pbTask.Deadline.AsTime()
		m.Deadline = &d
	}
	if pbTask.ScheduledDate != nil {
		sd := pbTask.ScheduledDate.AsTime()
		m.ScheduledDate = &sd
	}
	if pbTask.CreatedAt != nil {
		m.CreatedAt = pbTask.CreatedAt.AsTime()
	}
	if pbTask.UpdatedAt != nil {
		m.UpdatedAt = pbTask.UpdatedAt.AsTime()
	}
	if pbTask.Assignee != "" {
		a := pbTask.Assignee
		m.Assignee = &a
	}
	return m
}

// ProtoToReport 增强: 新增字段映射
func ProtoToReport(p *pb.Report) *models.Report {
	if p == nil {
		return nil
	}
	r := &models.Report{
		ID:      p.Id,
		UserID:  p.UserId,
		Type:    ProtoToReportType(p.Type),
		Title:   p.Title,
		Period:  p.Period,
		Content: p.Content,
	}
	if p.PolishedContent != "" {
		pc := p.PolishedContent
		r.PolishedContent = &pc
	}
	if p.StartDate != nil {
		r.CreatedAt = p.StartDate.AsTime()
	}
	if p.EndDate != nil {
		r.UpdatedAt = p.EndDate.AsTime()
	}
	// Stats -> Statistics
	if p.Stats != nil {
		r.Statistics.TotalTasks = int(p.Stats.TotalTasks)
		r.Statistics.CompletedTasks = int(p.Stats.CompletedTasks)
		r.Statistics.InProgressTasks = int(p.Stats.InProgressTasks)
		r.Statistics.OverdueTasks = int(p.Stats.OverdueTasks)
		r.Statistics.CompletionRate = int(p.Stats.CompletionRate)
	}
	return r
}

// EventType <-> Proto
func EventTypeToProto(t string) pb.EventType {
	switch t {
	case "birthday":
		return pb.EventType_EVENT_TYPE_BIRTHDAY
	case "anniversary":
		return pb.EventType_EVENT_TYPE_ANNIVERSARY
	case "holiday":
		return pb.EventType_EVENT_TYPE_HOLIDAY
	case "custom":
		return pb.EventType_EVENT_TYPE_CUSTOM
	case "meeting":
		return pb.EventType_EVENT_TYPE_MEETING
	case "deadline":
		return pb.EventType_EVENT_TYPE_DEADLINE
	default:
		return pb.EventType_EVENT_TYPE_UNSPECIFIED
	}
}
func ProtoToEventType(t pb.EventType) string {
	switch t {
	case pb.EventType_EVENT_TYPE_BIRTHDAY:
		return "birthday"
	case pb.EventType_EVENT_TYPE_ANNIVERSARY:
		return "anniversary"
	case pb.EventType_EVENT_TYPE_HOLIDAY:
		return "holiday"
	case pb.EventType_EVENT_TYPE_CUSTOM:
		return "custom"
	case pb.EventType_EVENT_TYPE_MEETING:
		return "meeting"
	case pb.EventType_EVENT_TYPE_DEADLINE:
		return "deadline"
	default:
		return "custom"
	}
}

// RecurrenceType <-> Proto
func RecurrenceTypeToProto(t string) pb.RecurrenceType {
	switch t {
	case "none":
		return pb.RecurrenceType_RECURRENCE_TYPE_NONE
	case "yearly":
		return pb.RecurrenceType_RECURRENCE_TYPE_YEARLY
	case "monthly":
		return pb.RecurrenceType_RECURRENCE_TYPE_MONTHLY
	case "weekly":
		return pb.RecurrenceType_RECURRENCE_TYPE_WEEKLY
	case "daily":
		return pb.RecurrenceType_RECURRENCE_TYPE_DAILY
	default:
		return pb.RecurrenceType_RECURRENCE_TYPE_UNSPECIFIED
	}
}
func ProtoToRecurrenceType(t pb.RecurrenceType) string {
	switch t {
	case pb.RecurrenceType_RECURRENCE_TYPE_NONE:
		return "none"
	case pb.RecurrenceType_RECURRENCE_TYPE_YEARLY:
		return "yearly"
	case pb.RecurrenceType_RECURRENCE_TYPE_MONTHLY:
		return "monthly"
	case pb.RecurrenceType_RECURRENCE_TYPE_WEEKLY:
		return "weekly"
	case pb.RecurrenceType_RECURRENCE_TYPE_DAILY:
		return "daily"
	default:
		return "none"
	}
}

// Event conversions
func EventToProto(e *models.Event) *pb.Event {
	if e == nil {
		return nil
	}
	var last *timestamppb.Timestamp
	if e.LastTriggeredAt != nil {
		last = timestamppb.New(*e.LastTriggeredAt)
	}
	cfg := map[string]string{}
	for k, v := range e.RecurrenceConfig {
		cfg[k] = stringify(v)
	}
	return &pb.Event{
		Id: e.ID.Hex(), UserId: e.UserID.Hex(), Title: e.Title, Description: e.Description,
		EventType: EventTypeToProto(e.EventType), EventDate: timestamppb.New(e.EventDate),
		RecurrenceType: RecurrenceTypeToProto(e.RecurrenceType), RecurrenceConfig: cfg,
		ImportanceLevel: int32(e.ImportanceLevel), Tags: e.Tags, Location: e.Location,
		IsAllDay: e.IsAllDay, CreatedAt: timestamppb.New(e.CreatedAt), UpdatedAt: timestamppb.New(e.UpdatedAt),
		IsActive: e.IsActive, LastTriggeredAt: last,
	}
}

func ProtoToEvent(p *pb.Event) *models.Event {
	if p == nil {
		return nil
	}
	uid, _ := primitive.ObjectIDFromHex(p.UserId)
	id, _ := primitive.ObjectIDFromHex(p.Id)
	var lt *time.Time
	if p.LastTriggeredAt != nil {
		t := p.LastTriggeredAt.AsTime()
		lt = &t
	}
	cfg := map[string]interface{}{}
	for k, v := range p.RecurrenceConfig {
		cfg[k] = v
	}
	return &models.Event{
		ID: id, UserID: uid, Title: p.Title, Description: p.Description,
		EventType: ProtoToEventType(p.EventType), EventDate: p.EventDate.AsTime(),
		RecurrenceType: ProtoToRecurrenceType(p.RecurrenceType), RecurrenceConfig: cfg,
		ImportanceLevel: int(p.ImportanceLevel), Tags: p.Tags, Location: p.Location,
		IsAllDay: p.IsAllDay, CreatedAt: p.CreatedAt.AsTime(), UpdatedAt: p.UpdatedAt.AsTime(),
		IsActive: p.IsActive, LastTriggeredAt: lt,
	}
}

// ReminderType <-> Proto
func ReminderTypeToProto(t string) pb.ReminderType {
	switch t {
	case "app":
		return pb.ReminderType_REMINDER_TYPE_APP
	case "email":
		return pb.ReminderType_REMINDER_TYPE_EMAIL
	case "both":
		return pb.ReminderType_REMINDER_TYPE_BOTH
	default:
		return pb.ReminderType_REMINDER_TYPE_UNSPECIFIED
	}
}
func ProtoToReminderType(t pb.ReminderType) string {
	switch t {
	case pb.ReminderType_REMINDER_TYPE_APP:
		return "app"
	case pb.ReminderType_REMINDER_TYPE_EMAIL:
		return "email"
	case pb.ReminderType_REMINDER_TYPE_BOTH:
		return "both"
	default:
		return "app"
	}
}

// Reminder conversions
func ReminderToProto(r *models.Reminder) *pb.Reminder {
	if r == nil {
		return nil
	}
	var last, next *timestamppb.Timestamp
	if r.LastSent != nil {
		last = timestamppb.New(*r.LastSent)
	}
	if r.NextSend != nil {
		next = timestamppb.New(*r.NextSend)
	}
	abs := make([]*timestamppb.Timestamp, 0, len(r.AbsoluteTimes))
	for _, t := range r.AbsoluteTimes {
		abs = append(abs, timestamppb.New(t))
	}
	return &pb.Reminder{Id: r.ID.Hex(), EventId: r.EventID.Hex(), UserId: r.UserID.Hex(), AdvanceDays: int32(r.AdvanceDays),
		ReminderTimes: r.ReminderTimes, AbsoluteTimes: abs, ReminderType: ReminderTypeToProto(r.ReminderType),
		CustomMessage: r.CustomMessage, IsActive: r.IsActive, LastSent: last, NextSend: next, CreatedAt: timestamppb.New(r.CreatedAt), UpdatedAt: timestamppb.New(r.UpdatedAt)}
}

// Notification conversions
func NotificationToProto(n *models.Notification) *pb.Notification {
	if n == nil {
		return nil
	}
	var readAt *timestamppb.Timestamp
	if n.ReadAt != nil {
		readAt = timestamppb.New(*n.ReadAt)
	}
	meta := map[string]string{}
	for k, v := range n.Metadata {
		meta[k] = stringify(v)
	}
	var eventID string
	if n.EventID != nil {
		eventID = n.EventID.Hex()
	}
	return &pb.Notification{Id: n.ID.Hex(), Type: n.Type, Message: n.Message, EventId: eventID, IsRead: n.ReadAt != nil, ReadAt: readAt, CreatedAt: timestamppb.New(n.CreatedAt), Metadata: meta}
}

func ProtoToNotificationCreate(req *pb.CreateNotificationRequest, user primitive.ObjectID) models.NotificationCreate {
	var eid *primitive.ObjectID
	if req.EventId != "" {
		if oid, err := primitive.ObjectIDFromHex(req.EventId); err == nil {
			eid = &oid
		}
	}
	return models.NotificationCreate{UserID: user, Type: req.Type, Message: req.Message, EventID: eid, Metadata: map[string]interface{}{}}
}

// stringify helper
func stringify(v interface{}) string { return fmt.Sprintf("%v", v) }

// TODO: Unified / Dashboard conversions 后续继续添加

// TimeToString 将时间转换为字符串（保持与现有API兼容）
func TimeToString(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
