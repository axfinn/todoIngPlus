import React, { useState, useEffect, useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import api from '../config/api';
// import { useDispatch, useSelector } from 'react-redux';
// import { fetchUnifiedUpcoming } from '../features/unified/unifiedSlice';
// import type { AppDispatch } from '../app/store';
import { useGetUpcomingQuery } from '../features/unified/unifiedApi';
import DataState from '../components/DataState';
import useFocusHighlight from '../hooks/useFocusHighlight';

// Event (后端 events 返回的字段实际为 Event 模型，使用 id / event_date / recurrence_type 等；这里只抽取需要的)
interface EventOption {
  id: string; // primitive.ObjectID hex
  title: string;
  event_date: string; // ISO 时间
  event_type?: string;
}

// 后端 ReminderWithEvent => Reminder 字段 + event 嵌套
interface ReminderItem {
  id: string; // reminder id
  event_id: string; // 事件 ObjectID
  user_id: string;
  advance_days: number;
  reminder_times: string[]; // ["09:00","18:00"]
  reminder_type: 'app' | 'email' | 'both';
  custom_message?: string;
  is_active: boolean;
  last_sent?: string;
  next_send?: string;
  event?: {
    id: string;
    title: string;
    event_date: string;
  };
  _invalid_id?: boolean; // 标记后端返回的ID无效，用于显示提示
}

// UpcomingReminder (后台 upcoming 返回的结构)

// 创建提醒表单
interface CreateReminderForm {
  event_id: string;
  advance_days: number;
  reminder_times: string[]; // 允许多个 HH:MM
  reminder_type: 'app' | 'email' | 'both';
  custom_message: string;
}

const RemindersPage: React.FC = () => {
  const { t, i18n } = useTranslation();
  const [reminders, setReminders] = useState<ReminderItem[]>([]);
  // const dispatch: AppDispatch = useDispatch();
  // 通过 RTK Query 预热提醒 upcoming 缓存
  useGetUpcomingQuery({ sources: ['reminder'], hours: 24 });
  const [events, setEvents] = useState<EventOption[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [showCreateModal, setShowCreateModal] = useState(false);
  useFocusHighlight({ attrName: 'data-reminder-id' });
  
  // 表单状态
  const [formData, setFormData] = useState<CreateReminderForm>({
    event_id: '',
    advance_days: 0,
    reminder_times: [''],
    reminder_type: 'app',
    custom_message: ''
  });
  const [formErrors, setFormErrors] = useState<Record<string,string>>({});
  const [submitting, setSubmitting] = useState(false);
  const [creatingTest, setCreatingTest] = useState(false);

  const validate = (draft: CreateReminderForm) => {
    const errs: Record<string,string> = {};
    if (!draft.event_id) errs.event_id = t('reminders.validation.eventRequired','Event required');
    const times = draft.reminder_times.filter(ti => ti.trim());
    if (!times.length) errs.reminder_times = t('reminders.validation.timeRequired','At least one time');
    times.forEach(tm => { if (!/^\d{2}:\d{2}$/.test(tm)) errs.reminder_times = t('reminders.validation.timeFormat','Time HH:MM'); });
    if (draft.advance_days < 0) errs.advance_days = t('reminders.validation.advanceNonNegative','Advance days >=0');
    return errs;
  };
  const isFormValid = useMemo(()=> Object.keys(formErrors).length===0, [formErrors]);

  // 获取提醒列表
  const isValidHex24 = (s: string) => /^[0-9a-fA-F]{24}$/.test(s) && s !== '000000000000000000000000';

  const fetchReminders = async () => {
    setIsLoading(true);
    setError(null);
    try {
      // 优先调用简化接口，加快响应
      const response = await api.get('/reminders/simple');
      const list = response.data.reminders || response.data?.data || [];
      // 归一化 reminder_times / reminderTimes，防止 null 导致渲染 join 报错
      const normalized: ReminderItem[] = list.map((r: any) => {
        let times: string[] = [];
        if (Array.isArray(r.reminder_times)) times = r.reminder_times.filter(Boolean);
        else if (Array.isArray(r.reminderTimes)) times = r.reminderTimes.filter(Boolean);
        const rid = r.id || r._id || '';
        return {
          ...r,
          id: rid,
          reminder_times: times,
          // 向后兼容 event 嵌套结构字段名差异
          event: r.event ? {
            id: r.event.id || r.event._id,
            title: r.event.title,
            event_date: r.event.event_date || r.event.eventDate || r.event.EventDate
          } : r.event
        };
      });
      // 不再直接过滤无效ID，生成临时ID以便列表显示，便于调试后端问题
  const enriched: ReminderItem[] = normalized.map((r: any, idx: number) => {
        if (isValidHex24(r.id)) return r;
        // 尝试解析可能的 MongoDB Extended JSON 格式 { _id: { $oid: '...' } }
        const maybeOid = r._id?.$oid;
        if (maybeOid && isValidHex24(maybeOid)) return { ...r, id: maybeOid, _invalid_id: true };
        return {
          ...r,
          id: `temp-${idx}-${r.event_id || 'noevent'}`,
          _invalid_id: true
        };
      });
      if (enriched.some(r => r._invalid_id)) {
        console.warn('Some reminders have invalid IDs from backend, using temporary IDs');
      }
	setReminders(enriched);
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to fetch reminders');
    } finally {
      setIsLoading(false);
    }
  };

  // 旧的 thunk 预热已移除，使用 RTK Query 自动缓存

  // 获取事件列表（用于创建提醒时选择）
  const fetchEvents = async () => {
    try {
      const response = await api.get('/events/options');
      const listRaw = response.data?.data || response.data?.events || response.data || [];
      const list = Array.isArray(listRaw) ? listRaw : (listRaw.events || []);
      const mapped = list.map((e: any) => ({
        id: e.id || e._id || e.ID,
        title: e.title || e.Title || '(no title)',
        event_date: e.event_date || e.EventDate || e.eventDate || e.event_date_time || ''
  })).filter((e: EventOption) => e.id && /^[0-9a-fA-F]{24}$/.test(e.id)); // 过滤非法ID避免选择后提交报错
      setEvents(mapped);
    } catch (err: any) {
      console.error('Failed to fetch event options:', err);
      setEvents([]);
      setError(err.response?.data?.message || 'Failed to fetch events options');
    }
  };

  // 创建提醒
  const handleCreateReminder = async (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    const errs = validate(formData);
    setFormErrors(errs);
    if (Object.keys(errs).length) { setSubmitting(false); return; }
    
    try {
      if (!formData.event_id) throw new Error(t('reminders.validation.eventRequired','Event required'));
      if (!formData.reminder_times.filter(t=>t.trim()).length) throw new Error(t('reminders.validation.timeRequired','At least one time'));
      const payload = {
        event_id: formData.event_id, // 后端需要 ObjectID，这里直接 hex 字符串
        advance_days: formData.advance_days,
        reminder_times: formData.reminder_times.filter(t=>t.trim()),
        reminder_type: formData.reminder_type,
        custom_message: formData.custom_message || undefined
      };
      await api.post('/reminders', payload);
  await fetchReminders();
      setShowCreateModal(false);
      resetForm();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to create reminder');
    } finally {
      setSubmitting(false);
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
  // RTK Query 缓存自动刷新（可后续添加 invalidation 机制）
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to delete reminder');
    }
  };

  const handleToggleActive = async (reminderId: string) => {
    try {
      await api.post(`/reminders/${reminderId}/toggle_active`);
      await fetchReminders();
    } catch (err: any) {
      setError(err.response?.data?.message || 'Failed to toggle reminder');
    }
  };

  // 暂停提醒
  const handleSnoozeReminder = async (reminderId: string, minutes: number) => {
    if (!isValidHex24(reminderId)) {
      setError('Invalid reminder id (client)');
      return;
    }
    try {
      const payload = { snooze_minutes: minutes || 60 };
  await api.post(`/reminders/${reminderId}/snooze`, payload);
      // 后端返回新 next_send? (当前只 message, snooze_minutes) —— 可刷新列表
      await fetchReminders();
    } catch (err: any) {
      console.error('Snooze error', err);
      const msg = err.response?.data?.message || err.message || 'Failed to snooze reminder';
      setError(`Snooze failed: ${msg}`);
    }
  };

  // 重置表单
  const resetForm = () => {
    setFormData({
      event_id: '',
      advance_days: 0,
      reminder_times: [''],
      reminder_type: 'app',
      custom_message: ''
    });
  setFormErrors({});
  };

  // 预览防抖
  const schedulePreview = () => {};

  // 处理表单输入变化
  const handleInputChange = (e: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData(prev => {
      const next = { ...prev, [name]: name === 'advance_days' ? Number(value) : value } as CreateReminderForm;
      setFormErrors(validate(next));
      return next;
    });
  if (name === 'event_id') {
      // 若选择的事件不在当前 events 列表，提示错误（可能是加载失败或过期）
      if (value && !events.find(ev => ev.id === value)) {
        setFormErrors(fe => ({ ...fe, event_id: t('reminders.validation.eventInvalid','Event invalid or not loaded') }));
      }
    }
  };

  const handleTimeChange = (idx: number, value: string) => {
    setFormData(prev => {
      const times = [...prev.reminder_times]; times[idx] = value; const next = { ...prev, reminder_times: times };
      setFormErrors(validate(next)); return next;
    });
    schedulePreview();
  };
  const addTimeField = () => setFormData(prev => ({ ...prev, reminder_times: [...prev.reminder_times, ''] }));
  const removeTimeField = (idx: number) => setFormData(prev => ({ ...prev, reminder_times: prev.reminder_times.filter((_,i)=>i!==idx) }));

  // 快捷时间集合与解析
  const quickSets: Record<string,string[]> = {
    morning: ['09:00'],
    work: ['09:00','13:00','18:00'],
    evening: ['20:00'],
    hourly: Array.from({length:5},(_,i)=> `${(9+i).toString().padStart(2,'0')}:00`)
  };
  const applyQuick = (k: string) => { const arr = quickSets[k]; if (!arr) return; setFormData(p=>({...p, reminder_times: arr })); setFormErrors(validate({...formData, reminder_times: arr })); };
  const parseFreeTimes = (raw: string) => {
    const parts = raw.split(/[\s,;，；]+/).map(s=>s.trim()).filter(Boolean);
    const out: string[] = [];
    for (let p of parts) {
      if (/^\d{3}$/.test(p)) p = '0'+p;
      if (/^\d{4}$/.test(p)) p = p.slice(0,2)+':'+p.slice(2);
      if (/^\d{2}:\d{2}$/.test(p)) out.push(p);
    }
    if (out.length) { setFormData(prev=>({...prev, reminder_times: out })); setFormErrors(validate({...formData, reminder_times: out })); }
  };

  const createTestReminder = async () => {
    if (!formData.event_id) { setFormErrors(f=>({...f, event_id: t('reminders.validation.eventRequired','Event required')})); return; }
    setCreatingTest(true);
    try {
      await api.post('/reminders/test', { event_id: formData.event_id, message: formData.custom_message || '测试提醒', delay_seconds: 5 });
      await fetchReminders();
    } catch (e:any) {
      setError(e.response?.data?.message || 'Failed to create test reminder');
    } finally { setCreatingTest(false); }
  };

  // 格式化日期显示
  const formatDateTime = (dateTimeString: string) => {
    if (!dateTimeString) return '-';
    const date = new Date(dateTimeString);
    if (isNaN(date.getTime())) return dateTimeString; // 保留原始，避免抛错
    try {
      return date.toLocaleString(i18n.language);
    } catch {
      return date.toISOString();
    }
  };

  // 获取事件标题
  const getEventTitle = (eventId: string) => {
    const event = events.find(e => e.id === eventId);
  return event ? event.title : t('reminders.unknownEvent', 'Unknown Event');
  };

  // 初始化数据
  useEffect(() => {
  fetchReminders();
  fetchEvents();
  // RTK Query 缓存自动维护
  }, []);

  return (
    <div className="container-fluid mt-4 panel-wrap">
      <div className="panel-overlay" />
      <div className="panel-content">
      <div className="row mb-4">
        <div className="col-12 d-flex justify-content-between align-items-center">
          <h1 className="h2 text-primary mb-0">
            <i className="bi bi-bell me-2"></i>
            {t('reminders.title')}
          </h1>
          <div className="d-flex gap-2">
          <button className="btn btn-outline-secondary" onClick={fetchReminders} title={t('common.refresh','Refresh')}>
            <i className="bi bi-arrow-clockwise" />
          </button>
          <button className="btn btn-primary" onClick={() => setShowCreateModal(true)}>
            <i className="bi bi-plus-lg me-1"></i>{t('reminders.create')}
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
        data={reminders}
        emptyHint={<div className="text-center py-5 text-muted"><i className="bi bi-bell-slash display-6 d-block mb-2"></i>{t('reminders.noReminders')}</div>}
        skeleton={<div className="card"><div className="card-body"><div className="placeholder-wave">{Array.from({length:8}).map((_,i)=>(<div key={i} className="mb-3"><span className="placeholder col-4 me-2"></span><span className="placeholder col-2"></span></div>))}</div></div></div>}
      >
        {(list) => (
          <div className="card">
            <div className="card-header">
              <h5 className="card-title mb-0"><i className="bi bi-list me-2"></i>{t('reminders.all', 'All Reminders')}</h5>
            </div>
            <div className="card-body p-0">
              <div className="table-responsive">
                <table className="table table-hover mb-0">
                  <thead>
                    <tr>
                      <th>{t('reminders.message')}</th>
                      <th>{t('reminders.eventHeader', 'Event')}</th>
                      <th>{t('reminders.remindAt')}</th>
                      <th>{t('reminders.type')}</th>
                      <th>{t('reminders.status', 'Status')}</th>
                      <th>{t('common.actions')}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {list.map((reminder) => (
                      <tr key={reminder.id} data-reminder-id={reminder.id} className={reminder._invalid_id ? 'table-warning' : ''}>
                        <td>
                          {reminder.custom_message || '-'}
                          <div className="text-muted small">ID: {reminder.id}{reminder._invalid_id && <span className="ms-2 badge bg-warning text-dark">invalid</span>}</div>
                        </td>
                        <td>{reminder.event?.title || getEventTitle(reminder.event_id)}</td>
            <td>{Array.isArray(reminder.reminder_times) ? reminder.reminder_times.join(', ') : ''}</td>
                        <td>
                          <span className={`badge bg-${reminder.reminder_type === 'email' ? 'info' : reminder.reminder_type === 'both' ? 'success' : 'primary'}`}>
                            {t(`reminders.type_${reminder.reminder_type}`, reminder.reminder_type)}
                          </span>
                        </td>
                        <td>
                          {reminder.is_active ? (
                            <span className="badge bg-success">{t('reminders.active', 'Active')}</span>
                          ) : (
                            <span className="badge bg-secondary">{t('reminders.inactive', 'Inactive')}</span>
                          )}
                        </td>
                        <td>
                          <div className="btn-group" role="group">
                            <div className="btn-group" role="group">
                              <button className="btn btn-sm btn-outline-warning" onClick={() => handleSnoozeReminder(reminder.id, 15)} title={t('reminders.snoozeMinutes', { minutes: 15 })}>
                                <i className="bi bi-clock"/>
                              </button>
                              <button className="btn btn-sm btn-outline-warning" onClick={() => handleSnoozeReminder(reminder.id, 60)} title={t('reminders.snoozeMinutes', { minutes: 60 })}>
                                1h
                              </button>
                              <button className="btn btn-sm btn-outline-warning" onClick={() => handleSnoozeReminder(reminder.id, 180)} title={t('reminders.snoozeMinutes', { minutes: 180 })}>
                                3h
                              </button>
                            </div>
                            <button className="btn btn-sm btn-outline-secondary" onClick={() => handleToggleActive(reminder.id)} title={reminder.is_active ? t('reminders.deactivate','Deactivate') : t('reminders.activate','Activate')}>
                              {reminder.is_active ? <i className="bi bi-pause" /> : <i className="bi bi-play" />}
                            </button>
                            <button className="btn btn-sm btn-outline-danger" onClick={() => handleDeleteReminder(reminder.id)} title={t('reminders.deleteConfirm')}>
                              <i className="bi bi-trash"></i>
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        )}
      </DataState>

      {/* 创建提醒模态框 */}
      {showCreateModal && (
        <div className="modal fade show d-block" tabIndex={-1} style={{ backgroundColor: 'rgba(0,0,0,0.5)' }}>
          <div className="modal-dialog">
            <div className="modal-content">
              <div className="modal-header">
                <h5 className="modal-title">{t('reminders.createNew', 'Create New Reminder')}</h5>
                <button
                  type="button"
                  className="btn-close"
                  onClick={() => { setShowCreateModal(false); resetForm(); }}
                ></button>
              </div>
              <form onSubmit={handleCreateReminder}>
                <div className="modal-body">
                  <div className="mb-3">
                    <label htmlFor="event_id" className="form-label">{t('reminders.eventSelect', 'Event *')}</label>
                    <select
                      className="form-select"
                      id="event_id"
                      name="event_id"
                      value={formData.event_id}
                      onChange={handleInputChange}
                      required
                    >
                      <option value="">{t('reminders.selectEvent', 'Select an event')}</option>
                      {events.map((e) => (
                        <option key={e.id} value={e.id}>
                          {e.title} - {formatDateTime(e.event_date)}
                        </option>
                      ))}
                    </select>
                    {formErrors.event_id && <div className="text-danger small mt-1">{formErrors.event_id}</div>}
                    {!events.length && (<div className="form-text text-warning small">{t('reminders.noEventOptions','No events available or load failed')}</div>)}
                  </div>
                  <div className="mb-3">
                    <label htmlFor="custom_message" className="form-label">{t('reminders.customMessage', 'Custom Message')}</label>
                    <textarea
                      className="form-control"
                      id="custom_message"
                      name="custom_message"
                      rows={2}
                      value={formData.custom_message}
                      onChange={handleInputChange}
                      placeholder="Enter custom reminder message..."
                    />
                  </div>
                  <div className="mb-3">
                    <label htmlFor="advance_days" className="form-label">{t('reminders.advanceDays', 'Advance Days')}</label>
                    <input
                      type="number"
                      className="form-control"
                      id="advance_days"
                      name="advance_days"
                      value={formData.advance_days}
                      onChange={handleInputChange}
                      min={0}
                      max={365}
                    />
                  </div>
                  <div className="mb-3">
                    <label className="form-label">{t('reminders.reminderTime', 'Reminder Time (HH:MM)')}*</label>
                    {formData.reminder_times.map((tm,idx)=>(
                      <div key={idx} className="d-flex align-items-center mb-2 gap-2">
                        <input type="time" className="form-control" value={tm} onChange={e=>handleTimeChange(idx, e.target.value)} required />
                        {formData.reminder_times.length>1 && (
                          <button type="button" className="btn btn-outline-danger btn-sm" onClick={()=>removeTimeField(idx)} title={t('common.delete')}>
                            <i className="bi bi-x" />
                          </button>
                        )}
                      </div>
                    ))}
                    <div className="d-flex flex-wrap gap-2 mb-2">
                      <button type="button" className="btn btn-outline-secondary btn-sm" onClick={addTimeField}>{t('reminders.addTime','Add')}</button>
                      <button type="button" className="btn btn-outline-secondary btn-sm" onClick={()=>applyQuick('morning')}>09:00</button>
                      <button type="button" className="btn btn-outline-secondary btn-sm" onClick={()=>applyQuick('work')}>{t('reminders.quick.work','Work')}</button>
                      <button type="button" className="btn btn-outline-secondary btn-sm" onClick={()=>applyQuick('evening')}>20:00</button>
                      <button type="button" className="btn btn-outline-secondary btn-sm" onClick={()=>applyQuick('hourly')}>{t('reminders.quick.hourly','Hourly')}</button>
                      <div className="input-group input-group-sm" style={{maxWidth:220}}>
                        <input type="text" className="form-control" placeholder="0900,1400 1830" onBlur={e=>parseFreeTimes(e.target.value)} />
                        <span className="input-group-text">NL</span>
                      </div>
                    </div>
                    {formErrors.reminder_times && <div className="text-danger small mt-1">{formErrors.reminder_times}</div>}
                  </div>
                  <div className="mb-3">
                    <label htmlFor="reminder_type" className="form-label">{t('reminders.reminderType', 'Reminder Type')} *</label>
                    <select
                      className="form-select"
                      id="reminder_type"
                      name="reminder_type"
                      value={formData.reminder_type}
                      onChange={handleInputChange}
                      required
                    >
                      <option value="app">{t('reminders.app')}</option>
                      <option value="email">{t('reminders.email')}</option>
                      <option value="both">{t('reminders.both', 'Both')}</option>
                    </select>
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
                  <button type="button" className="btn btn-outline-warning" onClick={createTestReminder} disabled={creatingTest || !formData.event_id} title={t('reminders.testReminder','Create test reminder (5s)')}>
                    {creatingTest ? <span className="spinner-border spinner-border-sm"/> : <i className="bi bi-lightning"/>}
                  </button>
                  <button type="submit" className="btn btn-primary" disabled={submitting || !isFormValid}>
                    {submitting ? (<><span className="spinner-border spinner-border-sm me-2"/> {t('reminders.creating','Creating...')}</>) : t('reminders.create')}
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}
      </div>
    </div>
  );
};

export default RemindersPage;
