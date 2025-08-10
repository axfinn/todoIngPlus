import React, { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import api from '../config/api';

// Event 类型定义
interface Event {
  _id: string;
  title: string;
  description: string;
  start_time: string;
  end_time: string;
  location?: string;
  recurrence?: {
    enabled: boolean;
    pattern: 'daily' | 'weekly' | 'monthly' | 'yearly';
    interval: number;
    end_date?: string;
  };
  user_id: string;
  created_at: string;
  updated_at: string;
}

// 创建事件的表单数据
interface CreateEventForm {
  title: string;
  description: string;
  start_time: string;
  end_time: string;
  location: string;
  recurrence_enabled: boolean;
  recurrence_pattern: 'daily' | 'weekly' | 'monthly' | 'yearly';
  recurrence_interval: number;
  recurrence_end_date: string;
}

const EventsPage: React.FC = () => {
  const { t } = useTranslation();
  const [events, setEvents] = useState<Event[]>([]);
  const [upcomingEvents, setUpcomingEvents] = useState<Event[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [editingEvent, setEditingEvent] = useState<Event | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  
  // 表单状态
  const [formData, setFormData] = useState<CreateEventForm>({
    title: '',
    description: '',
    start_time: '',
    end_time: '',
    location: '',
    recurrence_enabled: false,
    recurrence_pattern: 'daily',
    recurrence_interval: 1,
    recurrence_end_date: '',
  });

  // 获取事件列表
  const fetchEvents = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await api.get('/events');
      setEvents(response.data.events || response.data);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch events');
    } finally {
      setIsLoading(false);
    }
  };

  // 获取即将到来的事件
  const fetchUpcomingEvents = async () => {
    try {
      const response = await api.get('/events/upcoming?days=7');
      setUpcomingEvents(response.data.events || response.data);
    } catch (err: any) {
      console.error('Failed to fetch upcoming events:', err);
    }
  };

  // 创建事件
  const handleCreateEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsLoading(true);
    setError(null);
    
    try {
      const eventData: any = {
        title: formData.title,
        description: formData.description,
        start_time: formData.start_time,
        end_time: formData.end_time,
        location: formData.location || undefined,
      };

      if (formData.recurrence_enabled) {
        eventData.recurrence = {
          enabled: true,
          pattern: formData.recurrence_pattern,
          interval: formData.recurrence_interval,
          end_date: formData.recurrence_end_date || undefined,
        };
      }

      await api.post('/events', eventData);
      await fetchEvents();
      await fetchUpcomingEvents();
      setShowCreateModal(false);
      resetForm();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to create event');
    } finally {
      setIsLoading(false);
    }
  };

  // 删除事件
  const handleDeleteEvent = async (eventId: string) => {
    if (!window.confirm('Are you sure you want to delete this event?')) {
      return;
    }

    try {
      await api.delete(`/events/${eventId}`);
      await fetchEvents();
      await fetchUpcomingEvents();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to delete event');
    }
  };

  // 搜索事件
  const handleSearch = async () => {
    if (!searchQuery.trim()) {
      await fetchEvents();
      return;
    }

    setIsLoading(true);
    try {
      const response = await api.get(`/events/search?q=${encodeURIComponent(searchQuery)}`);
      setEvents(response.data.events || response.data);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to search events');
    } finally {
      setIsLoading(false);
    }
  };

  // 重置表单
  const resetForm = () => {
    setFormData({
      title: '',
      description: '',
      start_time: '',
      end_time: '',
      location: '',
      recurrence_enabled: false,
      recurrence_pattern: 'daily',
      recurrence_interval: 1,
      recurrence_end_date: '',
    });
    setEditingEvent(null);
  };

  // 处理表单输入变化
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value, type } = e.target;
    if (type === 'checkbox') {
      const checked = (e.target as HTMLInputElement).checked;
      setFormData(prev => ({ ...prev, [name]: checked }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
  };

  // 格式化日期显示
  const formatDateTime = (dateTimeString: string) => {
    const date = new Date(dateTimeString);
    return date.toLocaleString();
  };

  // 初始化数据
  useEffect(() => {
    fetchEvents();
    fetchUpcomingEvents();
  }, []);

  return (
    <div className="container-fluid mt-4">
      <div className="row">
        <div className="col-12">
          <div className="d-flex justify-content-between align-items-center mb-4">
            <h1 className="h2 text-primary">
              <i className="bi bi-calendar-event me-2"></i>
              Events & Calendar
            </h1>
            <button
              className="btn btn-primary"
              onClick={() => setShowCreateModal(true)}
            >
              <i className="bi bi-plus-lg me-1"></i>
              Create Event
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
        {/* 即将到来的事件 */}
        <div className="col-md-4 mb-4">
          <div className="card h-100">
            <div className="card-header bg-warning text-white">
              <h5 className="card-title mb-0">
                <i className="bi bi-clock me-2"></i>
                Upcoming Events (7 days)
              </h5>
            </div>
            <div className="card-body">
              {upcomingEvents.length === 0 ? (
                <p className="text-muted">No upcoming events</p>
              ) : (
                upcomingEvents.map((event) => (
                  <div key={event._id} className="border-bottom pb-2 mb-2">
                    <h6 className="mb-1">{event.title}</h6>
                    <small className="text-muted">
                      {formatDateTime(event.start_time)}
                    </small>
                    {event.location && (
                      <div className="small text-muted">
                        <i className="bi bi-geo-alt me-1"></i>
                        {event.location}
                      </div>
                    )}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>

        {/* 事件列表 */}
        <div className="col-md-8 mb-4">
          <div className="card">
            <div className="card-header">
              <div className="row align-items-center">
                <div className="col-md-6">
                  <h5 className="card-title mb-0">
                    <i className="bi bi-list me-2"></i>
                    All Events
                  </h5>
                </div>
                <div className="col-md-6">
                  <div className="input-group">
                    <input
                      type="text"
                      className="form-control"
                      placeholder="Search events..."
                      value={searchQuery}
                      onChange={(e) => setSearchQuery(e.target.value)}
                      onKeyPress={(e) => e.key === 'Enter' && handleSearch()}
                    />
                    <button className="btn btn-outline-secondary" onClick={handleSearch}>
                      <i className="bi bi-search"></i>
                    </button>
                  </div>
                </div>
              </div>
            </div>
            <div className="card-body">
              {isLoading ? (
                <div className="text-center py-4">
                  <div className="spinner-border text-primary" role="status">
                    <span className="visually-hidden">Loading...</span>
                  </div>
                </div>
              ) : events.length === 0 ? (
                <div className="text-center py-4">
                  <i className="bi bi-calendar-x display-4 text-muted mb-3"></i>
                  <p className="text-muted">No events found</p>
                </div>
              ) : (
                <div className="row">
                  {events.map((event) => (
                    <div key={event._id} className="col-lg-6 mb-3">
                      <div className="card border">
                        <div className="card-body">
                          <h6 className="card-title d-flex justify-content-between align-items-start">
                            {event.title}
                            <div className="dropdown">
                              <button
                                className="btn btn-sm btn-outline-secondary dropdown-toggle"
                                type="button"
                                data-bs-toggle="dropdown"
                              >
                                <i className="bi bi-three-dots"></i>
                              </button>
                              <ul className="dropdown-menu">
                                <li>
                                  <button
                                    className="dropdown-item"
                                    onClick={() => handleDeleteEvent(event._id)}
                                  >
                                    <i className="bi bi-trash me-2"></i>Delete
                                  </button>
                                </li>
                              </ul>
                            </div>
                          </h6>
                          <p className="card-text small">{event.description}</p>
                          <div className="small text-muted">
                            <div>
                              <i className="bi bi-calendar me-1"></i>
                              {formatDateTime(event.start_time)} - {formatDateTime(event.end_time)}
                            </div>
                            {event.location && (
                              <div>
                                <i className="bi bi-geo-alt me-1"></i>
                                {event.location}
                              </div>
                            )}
                            {event.recurrence?.enabled && (
                              <div>
                                <i className="bi bi-arrow-repeat me-1"></i>
                                Repeats {event.recurrence.pattern} (every {event.recurrence.interval})
                              </div>
                            )}
                          </div>
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* 创建事件模态框 */}
      {showCreateModal && (
        <div className="modal fade show d-block" tabIndex={-1} style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
          <div className="modal-dialog modal-lg">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">Create New Event</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => { setShowCreateModal(false); resetForm(); }}
                ></button>
              </div>
              <form onSubmit={handleCreateEvent}>
                <div className="modal-body">
                  <div className="row">
                    <div className="col-md-12 mb-3">
                      <label htmlFor="title" className="form-label">Title *</label>
                      <input
                        type="text"
                        className="form-control"
                        id="title"
                        name="title"
                        value={formData.title}
                        onChange={handleInputChange}
                        required
                      />
                    </div>
                    <div className="col-md-12 mb-3">
                      <label htmlFor="description" className="form-label">Description</label>
                      <textarea
                        className="form-control"
                        id="description"
                        name="description"
                        rows={3}
                        value={formData.description}
                        onChange={handleInputChange}
                      />
                    </div>
                    <div className="col-md-6 mb-3">
                      <label htmlFor="start_time" className="form-label">Start Time *</label>
                      <input
                        type="datetime-local"
                        className="form-control"
                        id="start_time"
                        name="start_time"
                        value={formData.start_time}
                        onChange={handleInputChange}
                        required
                      />
                    </div>
                    <div className="col-md-6 mb-3">
                      <label htmlFor="end_time" className="form-label">End Time *</label>
                      <input
                        type="datetime-local"
                        className="form-control"
                        id="end_time"
                        name="end_time"
                        value={formData.end_time}
                        onChange={handleInputChange}
                        required
                      />
                    </div>
                    <div className="col-md-12 mb-3">
                      <label htmlFor="location" className="form-label">Location</label>
                      <input
                        type="text"
                        className="form-control"
                        id="location"
                        name="location"
                        value={formData.location}
                        onChange={handleInputChange}
                      />
                    </div>
                    
                    {/* 重复设置 */}
                    <div className="col-md-12 mb-3">
                      <div className="form-check">
                        <input
                          className="form-check-input"
                          type="checkbox"
                          id="recurrence_enabled"
                          name="recurrence_enabled"
                          checked={formData.recurrence_enabled}
                          onChange={handleInputChange}
                        />
                        <label className="form-check-label" htmlFor="recurrence_enabled">
                          Repeat this event
                        </label>
                      </div>
                    </div>
                    
                    {formData.recurrence_enabled && (
                      <>
                        <div className="col-md-6 mb-3">
                          <label htmlFor="recurrence_pattern" className="form-label">Repeat Pattern</label>
                          <select
                            className="form-select"
                            id="recurrence_pattern"
                            name="recurrence_pattern"
                            value={formData.recurrence_pattern}
                            onChange={handleInputChange}
                          >
                            <option value="daily">Daily</option>
                            <option value="weekly">Weekly</option>
                            <option value="monthly">Monthly</option>
                            <option value="yearly">Yearly</option>
                          </select>
                        </div>
                        <div className="col-md-6 mb-3">
                          <label htmlFor="recurrence_interval" className="form-label">Every</label>
                          <input
                            type="number"
                            className="form-control"
                            id="recurrence_interval"
                            name="recurrence_interval"
                            min="1"
                            value={formData.recurrence_interval}
                            onChange={handleInputChange}
                          />
                        </div>
                        <div className="col-md-12 mb-3">
                          <label htmlFor="recurrence_end_date" className="form-label">End Date (optional)</label>
                          <input
                            type="date"
                            className="form-control"
                            id="recurrence_end_date"
                            name="recurrence_end_date"
                            value={formData.recurrence_end_date}
                            onChange={handleInputChange}
                          />
                        </div>
                      </>
                    )}
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
                      'Create Event'
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

export default EventsPage;
