import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import api from '../config/api';
// import { useDispatch } from 'react-redux';
// import type { AppDispatch } from '../app/store';
import { useGetUpcomingQuery } from '../features/unified/unifiedApi';
import SeverityBadge from '../components/SeverityBadge';
import useFocusHighlight from '../hooks/useFocusHighlight';
import DataState from '../components/DataState';

// 后端 Event 模型核心字段（简化版）
interface EventItem {
  id: string;              // ObjectID
  title: string;
  description: string;
  event_type: string;      // birthday / anniversary / holiday / custom / meeting / deadline
  event_date: string;      // ISO 时间
  importance_level: number;
  location?: string;
  is_all_day: boolean;
  recurrence_type: string; // none / yearly / monthly / weekly / daily
  created_at?: string;
  updated_at?: string;
}

// 创建事件表单
interface CreateEventForm {
  title: string;
  description: string;
  event_date: string;          // datetime-local
  event_type: string;
  importance_level: number;
  location: string;
  is_all_day: boolean;
  recurrence_type: string;     // none / daily / weekly / monthly / yearly
}

const EventsPage: React.FC = () => {
  const { t } = useTranslation();
  const [events, setEvents] = useState<EventItem[]>([]);
  // const dispatch: AppDispatch = useDispatch(); // 事件后续若需全局状态再启用
  // 预热 unified 缓存（不单独展示列表）使用 RTK Query 自动缓存
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');

  // 使用通用 focus 高亮钩子
  useFocusHighlight({ attrName: 'data-event-id' });
  
  // 表单状态
  const initialForm: CreateEventForm = {
    title: '',
    description: '',
    event_date: '',
    event_type: 'custom',
    importance_level: 3,
    location: '',
    is_all_day: false,
    recurrence_type: 'none'
  };
  const [formData, setFormData] = useState<CreateEventForm>(initialForm);
  const [formErrors, setFormErrors] = useState<Record<string,string>>({});
  const [submitting, setSubmitting] = useState(false);

  const validate = (draft: CreateEventForm) => {
    const errs: Record<string,string> = {};
    if (!draft.title.trim()) errs.title = t('events.validation.titleRequired', 'Title is required');
    if (!draft.event_date) errs.event_date = t('events.validation.dateRequired', 'Event date required');
    if (draft.importance_level < 1 || draft.importance_level > 5) errs.importance_level = t('events.validation.importanceRange', 'Importance 1-5');
    return errs;
  };
  const isFormValid = useMemo(()=> Object.keys(formErrors).length===0, [formErrors]);

  // 获取事件列表
  const fetchEvents = async () => {
    setIsLoading(true);
    setError(null);
    try {
      const response = await api.get('/events');
      const list = response.data.events || [];
      // 将后端字段映射为前端使用
      const mapped = list.map((e: any) => ({
        id: e.id || e._id,
        title: e.title,
        description: e.description,
        event_type: e.event_type,
        event_date: e.event_date,
        importance_level: e.importance_level,
        location: e.location,
        is_all_day: e.is_all_day,
        recurrence_type: e.recurrence_type,
        created_at: e.created_at,
        updated_at: e.updated_at,
      }));
      setEvents(mapped);
  // focus 高亮逻辑已抽象至 hook
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch events');
    } finally {
      setIsLoading(false);
    }
  };

  // 预热 unified 缓存（不再单独展示）
  // 使用 RTK Query 自动请求 (events 7 天)
  useGetUpcomingQuery({ sources: ['event'], hours: 24 * 7 });

  // 创建事件
  const handleCreateEvent = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    const errs = validate(formData);
    setFormErrors(errs);
    if (Object.keys(errs).length) { setSubmitting(false); return; }
    
    try {
  if (!formData.event_date) throw new Error(t('events.validation.dateRequired', 'Event date required'));
      // datetime-local 不带时区，Go 默认 time.Time 反序列化需要 RFC3339 (含时区)。
      // 若用户输入形如 2025-08-10T12:30 则转为本地时间再序列化 ISO。
      let eventDateStr = formData.event_date; // 保留原始 datetime-local 由后端多格式解析
      const payload: any = {
        title: formData.title,
        description: formData.description,
        event_type: formData.event_type,
        event_date: eventDateStr,
        importance_level: formData.importance_level,
        location: formData.location || undefined,
        is_all_day: formData.is_all_day,
        recurrence_type: formData.recurrence_type,
        raw_event_date: formData.event_date,
      };
  await api.post('/events', payload);
  await fetchEvents();
      setShowCreateModal(false);
      resetForm();
    } catch (err: any) {
  const msg = err.response?.data?.message || err.message || 'Failed to create event';
  setError(msg);
    } finally {
  setSubmitting(false);
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
  // unified upcoming 缓存由 RTK Query 自动刷新（可后续添加乐观更新）
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
      const list = response.data.events || [];
      const mapped = list.map((e: any) => ({
        id: e.id || e._id,
        title: e.title,
        description: e.description,
        event_type: e.event_type,
        event_date: e.event_date,
        importance_level: e.importance_level,
        location: e.location,
        is_all_day: e.is_all_day,
        recurrence_type: e.recurrence_type,
        created_at: e.created_at,
        updated_at: e.updated_at,
      }));
      setEvents(mapped);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to search events');
    } finally {
      setIsLoading(false);
    }
  };

  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement>) => {
    const { name, value, type, checked } = e.target as any;
    if (type === 'checkbox') {
      setFormData(prev => ({ ...prev, [name]: checked }));
    } else if (name === 'importance_level') {
      setFormData(prev => ({ ...prev, [name]: Number(value) }));
    } else {
      setFormData(prev => ({ ...prev, [name]: value }));
    }
    // 实时校验
  setFormErrors(() => {
      const draft = { ...formData, [name]: type==='checkbox'? checked : (name==='importance_level'? Number(value): value) } as CreateEventForm;
      return validate(draft);
    });
  };

  const resetForm = () => setFormData(initialForm);

  // 格式化日期显示
  const formatDateTime = (dateTimeString: string) => {
    if (!dateTimeString) return '-';
    const date = new Date(dateTimeString);
    return date.toLocaleString();
  };

  // 初始化数据
  useEffect(() => {
    fetchEvents();
  // unified upcoming 缓存由 RTK Query 自动刷新
  }, []);

  return (
    <div className="container-fluid mt-4 panel-wrap">
      <div className="panel-overlay" />
      <div className="panel-content">
      <div className="row mb-4">
        <div className="col-12 d-flex justify-content-between align-items-center">
          <h1 className="h2 text-primary mb-0">
            <i className="bi bi-calendar-event me-2"></i>
            {t('events.title') || t('events.all') || 'Events'}
          </h1>
          <div className="d-flex gap-2">
            <div className="input-group">
              <input
                type="text"
                className="form-control"
                placeholder={t('events.searchPlaceholder') || 'Search events...'}
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                onKeyDown={(e) => e.key === 'Enter' && handleSearch()}
              />
              <button className="btn btn-outline-secondary" onClick={handleSearch} disabled={isLoading}>
                <i className="bi bi-search"></i>
              </button>
            </div>
            <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
              <i className="bi bi-plus-lg me-1"></i>
              {t('events.create') || 'Create'}
            </button>
          </div>
        </div>
      </div>

      {error && (
        <div className="alert alert-danger alert-dismissible fade show" role="alert">
          {error}
          <button type="button" className="btn-close" onClick={() => setError(null)}></button>
        </div>
      )}

      <DataState
        loading={isLoading}
        error={error}
        data={events}
        emptyHint={<div className="text-center py-5 text-muted">{t('events.noEvents')}</div>}
        skeleton={<div className="row">{Array.from({length:6}).map((_,i)=>(<div key={i} className="col-lg-6 mb-3"><div className="border rounded p-4 placeholder-wave" style={{height:160}}><span className="placeholder col-8 mb-3 d-block"></span><span className="placeholder col-6 mb-2 d-block"></span><span className="placeholder col-4 d-block"></span></div></div>))}</div>}
      >
        {(list) => (
          <div className="row">
            {list.map((event) => (
              <div key={event.id} data-event-id={event.id} className="col-lg-6 mb-3">
                <div className="card border h-100">
                  <div className="card-body d-flex flex-column">
                    <h6 className="card-title d-flex justify-content-between align-items-start">
                      <span className="me-2 flex-grow-1 text-truncate" title={event.title}>{event.title}</span>
                      <div className="d-flex align-items-center gap-2">
                        <SeverityBadge source="event" scheduledAt={event.event_date} importance={event.importance_level} showLabel={false} />
                        <div className="dropdown">
                          <button className="btn btn-sm btn-outline-secondary dropdown-toggle" type="button" data-bs-toggle="dropdown">
                            <i className="bi bi-three-dots"></i>
                          </button>
                          <ul className="dropdown-menu">
                            <li>
                              <button className="dropdown-item" onClick={() => handleDeleteEvent(event.id)}>
                                <i className="bi bi-trash me-2"></i>{t('common.delete')}
                              </button>
                            </li>
                          </ul>
                        </div>
                      </div>
                    </h6>
                    {event.description && <p className="card-text small mb-2">{event.description}</p>}
                    <div className="small text-muted mt-auto">
                      <div><i className="bi bi-calendar me-1"></i>{formatDateTime(event.event_date)}</div>
                      {event.location && (
                        <div><i className="bi bi-geo-alt me-1"></i>{event.location}</div>
                      )}
                      {event.recurrence_type !== 'none' && (
                        <div><i className="bi bi-arrow-repeat me-1"></i>{t('events.repeats', { pattern: t(`events.${event.recurrence_type}`) || event.recurrence_type, interval: '' }).replace(' (every )','')}</div>
                      )}
                    </div>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}
      </DataState>
  {/* 创建按钮已移动到顶部 */}

      {/* 创建事件模态框 */}
      {showCreateModal && (
        <div className="modal fade show d-block" tabIndex={-1} style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
          <div className="modal-dialog modal-lg">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">{t('events.createNew')}</h5>
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
                      <label htmlFor="title" className="form-label">{t('common.title') || 'Title'} *</label>
                      <input
                        type="text"
                        className="form-control"
                        id="title"
                        name="title"
                        value={formData.title}
                        onChange={handleInputChange}
                        required
                        aria-invalid={!!formErrors.title}
                        aria-describedby={formErrors.title? 'err-title': undefined}
                      />
                      {formErrors.title && <div id="err-title" className="text-danger small mt-1">{formErrors.title}</div>}
                    </div>
                    <div className="col-md-12 mb-3">
                      <label htmlFor="description" className="form-label">{t('dashboard.description') || 'Description'}</label>
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
                      <label htmlFor="event_date" className="form-label">{t('events.eventDate') || 'Event Date'} *</label>
                      <input
                        type="datetime-local"
                        className="form-control"
                        id="event_date"
                        name="event_date"
                        value={formData.event_date}
                        onChange={handleInputChange}
                        required
                        aria-invalid={!!formErrors.event_date}
                        aria-describedby={formErrors.event_date? 'err-event-date': undefined}
                      />
                      {formErrors.event_date && <div id="err-event-date" className="text-danger small mt-1">{formErrors.event_date}</div>}
                      {/* 快捷时间按钮 */}
                      <div className="mt-2 d-flex flex-wrap gap-2 small">
                        {(() => {
                          const now = new Date();
                          const pad = (n:number)=> n.toString().padStart(2,'0');
                          const toLocalInput = (d:Date)=> `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
                          const clone = (d:Date)=> new Date(d.getTime());
                          const startOfDay = (d:Date)=> new Date(d.getFullYear(), d.getMonth(), d.getDate());
                          const addDays = (d:Date,days:number)=>{const nd=clone(d); nd.setDate(nd.getDate()+days); return nd;};
                          const nextWeekday = (weekday:number)=>{ // 0=Sun
                            const nd = clone(now); let diff = (weekday - nd.getDay() + 7) % 7; if(diff===0) diff=7; nd.setDate(nd.getDate()+diff); return nd;
                          };
                          const presets = [
                            { label: t('quick.nowPlus1h') || '+1h', calc: ()=> { const d=clone(now); d.setHours(d.getHours()+1,0,0,0); return d; } },
                            { label: t('quick.tonight') || '今晚20:00', calc: ()=> { const d = startOfDay(now); d.setHours(20,0,0,0); if(d < now) d.setDate(d.getDate()+1); return d; } },
                            { label: t('quick.tomorrowMorning') || '明早09:00', calc: ()=> { const d = addDays(startOfDay(now),1); d.setHours(9,0,0,0); return d; } },
                            { label: t('quick.fridayEvening') || '周五18:00', calc: ()=> { const d = nextWeekday(5); d.setHours(18,0,0,0); return d; } },
                            { label: t('quick.nextMonday') || '下周一09:00', calc: ()=> { const d = nextWeekday(1); d.setHours(9,0,0,0); return d; } },
                          ];
                          return presets.map(p => (
                            <button key={p.label} type="button" className="btn btn-outline-secondary btn-sm"
                              onClick={()=> setFormData(prev=> ({...prev, event_date: toLocalInput(p.calc())}))}>{p.label}</button>
                          ));
                        })()}
                      </div>
                    </div>
                    <div className="col-md-6 mb-3">
                      <label htmlFor="event_type" className="form-label">{t('events.eventType') || 'Event Type'}</label>
                      <select
                        className="form-select"
                        id="event_type"
                        name="event_type"
                        value={formData.event_type}
                        onChange={handleInputChange}
                      >
                        <option value="custom">{t('events.custom') || 'custom'}</option>
                        <option value="birthday">{t('events.birthday') || 'birthday'}</option>
                        <option value="anniversary">{t('events.anniversary') || 'anniversary'}</option>
                        <option value="holiday">{t('events.holiday') || 'holiday'}</option>
                        <option value="meeting">{t('events.meeting') || 'meeting'}</option>
                        <option value="deadline">{t('events.deadline') || 'deadline'}</option>
                      </select>
                    </div>
                    <div className="col-md-4 mb-3">
                      <label htmlFor="importance_level" className="form-label">{t('events.importance') || 'Importance'}</label>
                      <input
                        type="number"
                        className="form-control"
                        id="importance_level"
                        name="importance_level"
                        min={1}
                        max={5}
                        value={formData.importance_level}
                        onChange={handleInputChange}
                        aria-invalid={!!formErrors.importance_level}
                        aria-describedby={formErrors.importance_level? 'err-importance': undefined}
                      />
                      {formErrors.importance_level && <div id="err-importance" className="text-danger small mt-1">{formErrors.importance_level}</div>}
                    </div>
                    <div className="col-md-4 mb-3 form-check" style={{paddingTop: '2.1rem'}}>
                      <input
                        type="checkbox"
                        className="form-check-input"
                        id="is_all_day"
                        name="is_all_day"
                        checked={formData.is_all_day}
                        onChange={handleInputChange}
                      />
                      <label className="form-check-label" htmlFor="is_all_day">{t('events.allDay') || 'All Day'}</label>
                    </div>
                    <div className="col-md-4 mb-3">
                      <label htmlFor="recurrence_type" className="form-label">{t('events.recurrence') || 'Recurrence'}</label>
                      <select
                        className="form-select"
                        id="recurrence_type"
                        name="recurrence_type"
                        value={formData.recurrence_type}
                        onChange={handleInputChange}
                      >
                        <option value="none">{t('events.none') || 'none'}</option>
                        <option value="daily">{t('events.daily')}</option>
                        <option value="weekly">{t('events.weekly')}</option>
                        <option value="monthly">{t('events.monthly')}</option>
                        <option value="yearly">{t('events.yearly')}</option>
                      </select>
                    </div>
                    <div className="col-md-12 mb-3">
                      <label htmlFor="location" className="form-label">{t('events.location')}</label>
                      <input
                        type="text"
                        className="form-control"
                        id="location"
                        name="location"
                        value={formData.location}
                        onChange={handleInputChange}
                      />
                    </div>
                  </div>
                </div>
                <div className="modal-footer">
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={() => { setShowCreateModal(false); resetForm(); }}
                  >
                    {t('common.cancel')}
                  </button>
                  <button
                    type="submit"
                    className="btn btn-primary"
                    disabled={submitting || !isFormValid}
                  >
                    {submitting ? (
                      <>
                        <span className="spinner-border spinner-border-sm me-2" role="status" aria-hidden="true"></span>
                        {t('events.creating')}
                      </>
                    ) : (
                      t('events.create')
                    )}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
      </div>{/* panel-content */}
    </div>
  );
};

export default EventsPage;
