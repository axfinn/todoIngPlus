import React, { useState, useEffect } from 'react';
import api from '../config/api';

// Reminder 类型定义
interface Reminder {
  _id: string;
  event_id: string;
  message: string;
  remind_at: string;
  type: 'email' | 'app';
  is_sent: boolean;
  sent_at?: string;
  user_id: string;
  created_at: string;
  updated_at: string;
}

// Event 类型定义（用于下拉选择）
interface Event {
  _id: string;
  title: string;
  start_time: string;
}

// 创建提醒的表单数据
interface CreateReminderForm {
  event_id: string;
  message: string;
  remind_at: string;
  type: 'email' | 'app';
}

const RemindersPage: React.FC = () => {
  const [reminders, setReminders] = useState<Reminder[]>([]);
  const [upcomingReminders, setUpcomingReminders] = useState<Reminder[]>([]);
  const [events, setEvents] = useState<Event[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  
  // 表单状态
  const [formData, setFormData] = useState<CreateReminderForm>({
    event_id: '',
    message: '',
    remind_at: '',
    type: 'app',
  });

  // 获取提醒列表
  const fetchReminders = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await api.get('/reminders');
      setReminders(response.data.reminders || response.data);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch reminders');
    } finally {
      setIsLoading(false);
    }
  };

  // 获取即将发送的提醒
  const fetchUpcomingReminders = async () => {
    try {
      const response = await api.get('/reminders/upcoming?hours=24');
      setUpcomingReminders(response.data.reminders || response.data);
    } catch (err: any) {
      console.error('Failed to fetch upcoming reminders:', err);
    }
  };

  // 获取事件列表（用于创建提醒时选择）
  const fetchEvents = async () => {
    try {
      const response = await api.get('/events');
      setEvents(response.data.events || response.data);
    } catch (err: any) {
      console.error('Failed to fetch events:', err);
    }
  };

  // 创建提醒
  const handleCreateReminder = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);
    
    try {
      await api.post('/reminders', formData);
      await fetchReminders();
      await fetchUpcomingReminders();
      setShowCreateModal(false);
      resetForm();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to create reminder');
    } finally {
      setIsLoading(false);
    }
  };

  // 删除提醒
  const handleDeleteReminder = async (reminderId: string) => {
    if (!window.confirm('Are you sure you want to delete this reminder?')) {
      return;
    }

    try {
      await api.delete(`/reminders/${reminderId}`);
      await fetchReminders();
      await fetchUpcomingReminders();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to delete reminder');
    }
  };

  // 暂停提醒
  const handleSnoozeReminder = async (reminderId: string, minutes: number) => {
    try {
      await api.post(`/reminders/${reminderId}/snooze`, { minutes });
      await fetchReminders();
      await fetchUpcomingReminders();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to snooze reminder');
    }
  };

  // 重置表单
  const resetForm = () => {
    setFormData({
      event_id: '',
      message: '',
      remind_at: '',
      type: 'app',
    });
  };

  // 处理表单输入变化
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => ({ ...prev, [name]: value }));
  };

  // 格式化日期显示
  const formatDateTime = (dateTimeString: string) => {
    const date = new Date(dateTimeString);
    return date.toLocaleString();
  };

  // 获取事件标题
  const getEventTitle = (eventId: string) => {
    const event = events.find(e => e._id === eventId);
    return event ? event.title : 'Unknown Event';
  };

  // 初始化数据
  useEffect(() => {
    fetchReminders();
    fetchUpcomingReminders();
    fetchEvents();
  }, []);

  return (
    <div className="container-fluid mt-4">
      <div className="row">
        <div className="col-12">
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h2 text-primary">
              <i className="bi bi-bell me-2"></i>
              Reminders
            </h1>
            <button
              className="btn btn-primary"
              onClick={() => setShowCreateModal(true)}
            >
              <i className="bi bi-plus-lg me-1"></i>
              Create Reminder
            </button>
          </div>

          {error && (
            <div className="alert alert-danger alert-dismissible fade show" role="alert">
              {error}
              <button
                type="button"
                className="btn-close"
                onClick={() => setError(null)}
              ></button>
            </div>
          )}
        </div>
      </div>

      <div className="row">
        {/* 即将发送的提醒 */}
        <div className="col-md-4 mb-4">
          <div className="card h-100">
            <div className="card-header bg-warning text-white">
              <h5 className="card-title mb-0">
                <i className="bi bi-alarm me-2"></i>
                Upcoming Reminders (24h)
              </h5>
            </div>
            <div className="card-body">
              {upcomingReminders.length === 0 ? (
                <p className="text-muted">No upcoming reminders</p>
              ) : (
                upcomingReminders.map((reminder) => (
                  <div key={reminder._id} className="border-bottom pb-2 mb-2">
                    <h6 className="mb-1">{reminder.message}</h6>
                    <small className="text-muted d-block">
                      Event: {getEventTitle(reminder.event_id)}
                    </small>
                    <small className="text-muted d-block">
                      <i className="bi bi-clock me-1"></i>
                      {formatDateTime(reminder.remind_at)}
                    </small>
                    <small className="text-muted d-block">
                      <i className="bi bi-tag me-1"></i>
                      {reminder.type}
                    </small>
                    <div className="mt-2">
                      <button
                        className="btn btn-sm btn-outline-warning me-1"
                        onClick={() => handleSnoozeReminder(reminder._id, 15)}
                      >
                        Snooze 15m
                      </button>
                      <button
                        className="btn btn-sm btn-outline-warning"
                        onClick={() => handleSnoozeReminder(reminder._id, 60)}
                      >
                        Snooze 1h
                      </button>
                    </div>
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        {/* 提醒列表 */}
        <div className="col-md-8 mb-4">
          <div className="card">
            <div className="card-header">
              <h5 className="card-title mb-0">
                <i className="bi bi-list me-2"></i>
                All Reminders
              </h5>
            </div>
            <div className="card-body">
              {isLoading ? (
                <div className="text-center py-4">
                  <div className="spinner-border text-primary" role="status">
                    <span className="visually-hidden">Loading...</span>
                  </div>
                </div>
              ) : reminders.length === 0 ? (
                <div className="text-center py-4">
                  <i className="bi bi-bell-slash display-4 text-muted mb-3"></i>
                  <p className="text-muted">No reminders found</p>
                </div>
              ) : (
                <div className="table-responsive">
                  <table className="table table-hover">
                    <thead>
                      <tr>
                        <th>Message</th>
                        <th>Event</th>
                        <th>Remind At</th>
                        <th>Type</th>
                        <th>Status</th>
                        <th>Actions</th>
                      </tr>
                    </thead>
                    <tbody>
                      {reminders.map((reminder) => (
                        <tr key={reminder._id}>
                          <td>{reminder.message}</td>
                          <td>{getEventTitle(reminder.event_id)}</td>
                          <td>{formatDateTime(reminder.remind_at)}</td>
                          <td>
                            <span className={`badge bg-${reminder.type === 'email' ? 'info' : 'primary'}`}>
                              {reminder.type}
                            </span>
                          </td>
                          <td>
                            {reminder.is_sent ? (
                              <span className="badge bg-success">
                                <i className="bi bi-check me-1"></i>
                                Sent {reminder.sent_at && `at ${formatDateTime(reminder.sent_at)}`}
                              </span>
                            ) : (
                              <span className="badge bg-secondary">Pending</span>
                            )}
                          </td>
                          <td>
                            <div className="btn-group" role="group">
                              {!reminder.is_sent && (
                                <>
                                  <button
                                    className="btn btn-sm btn-outline-warning"
                                    onClick={() => handleSnoozeReminder(reminder._id, 15)}
                                    title="Snooze 15 minutes"
                                  >
                                    <i className="bi bi-clock"></i>
                                  </button>
                                </>
                              )}
                              <button
                                className="btn btn-sm btn-outline-danger"
                                onClick={() => handleDeleteReminder(reminder._id)}
                                title="Delete reminder"
                              >
                                <i className="bi bi-trash"></i>
                              </button>
                            </div>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* 创建提醒模态框 */}
      {showCreateModal && (
        <div className="modal fade show d-block" tabIndex={-1} style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
          <div className="modal-dialog">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">Create New Reminder</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => { setShowCreateModal(false); resetForm(); }}
                ></button>
              </div>
              <form onSubmit={handleCreateReminder}>
                <div className="modal-body">
                  <div className="mb-3">
                    <label htmlFor="event_id" className="form-label">Event *</label>
                    <select
                      className="form-select"
                      id="event_id"
                      name="event_id"
                      value={formData.event_id}
                      onChange={handleInputChange}
                      required
                    >
                      <option value="">Select an event</option>
                      {events.map((event) => (
                        <option key={event._id} value={event._id}>
                          {event.title} - {formatDateTime(event.start_time)}
                        </option>
                      ))}
                    </select>
                  </div>
                  <div className="mb-3">
                    <label htmlFor="message" className="form-label">Reminder Message *</label>
                    <textarea
                      className="form-control"
                      id="message"
                      name="message"
                      rows={3}
                      value={formData.message}
                      onChange={handleInputChange}
                      placeholder="Enter reminder message..."
                      required
                    />
                  </div>
                  <div className="mb-3">
                    <label htmlFor="remind_at" className="form-label">Remind At *</label>
                    <input
                      type="datetime-local"
                      className="form-control"
                      id="remind_at"
                      name="remind_at"
                      value={formData.remind_at}
                      onChange={handleInputChange}
                      required
                    />
                  </div>
                  <div className="mb-3">
                    <label htmlFor="type" className="form-label">Reminder Type *</label>
                    <select
                      className="form-select"
                      id="type"
                      name="type"
                      value={formData.type}
                      onChange={handleInputChange}
                      required
                    >
                      <option value="app">App Notification</option>
                      <option value="email">Email</option>
                    </select>
                  </div>
                </div>
                <div className="modal-footer">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={() => { setShowCreateModal(false); resetForm(); }}
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={isLoading}
                  >
                    {isLoading ? (
                      <>
                        <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        Creating...
                      </>
                    ) : (
                      'Create Reminder'
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default RemindersPage;
