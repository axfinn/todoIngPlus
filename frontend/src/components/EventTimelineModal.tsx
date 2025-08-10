import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import api from '../config/api';

// 时间线条目类型
export interface EventTimelineItem {
  id: string;
  event_id: string;
  user_id: string;
  type: string; // comment / system / status_change
  content: string;
  meta?: Record<string, string>;
  created_at: string;
  updated_at: string;
}

interface Props {
  eventId: string;
  eventTitle: string;
  onClose: () => void;
}

const humanTime = (iso: string) => {
  const d = new Date(iso);
  const diffMs = Date.now() - d.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  if (diffSec < 60) return '刚刚';
  const diffMin = Math.floor(diffSec / 60);
  if (diffMin < 60) return diffMin + ' 分钟前';
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return diffHr + ' 小时前';
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return diffDay + ' 天前';
  return d.toLocaleString();
};

const systemLabel = (item: EventTimelineItem) => {
  if (item.meta?.kind === 'event_start') return '事件开始';
  if (item.meta?.reminder_id) return '提醒已发送';
  switch (item.type) {
    case 'status_change': return '状态变更';
    case 'system': return '系统';
    default: return '评论';
  }
};

const EventTimelineModal: React.FC<Props> = ({ eventId, eventTitle, onClose }) => {
  const [items, setItems] = useState<EventTimelineItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [input, setInput] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const listRef = useRef<HTMLDivElement | null>(null);

  const { t } = useTranslation();
  const fetchTimeline = useCallback(async (opts?: { more?: boolean }) => {
    if (loading || loadingMore) return;
    const isMore = !!opts?.more;
    isMore ? setLoadingMore(true) : setLoading(true);
    setError(null);
    try {
      const params = new URLSearchParams();
      params.set('limit', '50');
      if (isMore && items.length) {
        params.set('before_id', items[0].id); // 取最早的 id 做向前翻页
      }
      const res = await api.get(`/events/${eventId}/timeline?` + params.toString());
      const list: EventTimelineItem[] = res.data.items || [];
      if (isMore) {
        if (list.length === 0) setHasMore(false);
        setItems(prev => [...list, ...prev]); // 旧的在后，新加载的更早的在前
      } else {
        setItems(list);
        setHasMore(list.length === 50); // 粗略判断
        // 滚动到底部看到最新
        setTimeout(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight; }, 50);
      }
    } catch (e: any) {
      setError(e.response?.data?.message || e.message || '加载时间线失败');
    } finally {
      isMore ? setLoadingMore(false) : setLoading(false);
    }
  }, [eventId, items, loading, loadingMore]);

  useEffect(() => { fetchTimeline(); }, [fetchTimeline]);

  // 移除轮询：后续可与 SSE 对接（当前详情页已有 SSE，Modal 若需实时可在打开时订阅）以降低闪烁

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim()) return;
    setSubmitting(true);
    try {
      const res = await api.post(`/events/${eventId}/comments`, { content: input.trim(), type: 'comment' });
      const added: EventTimelineItem = res.data;
      setItems(prev => [...prev, added]);
      setInput('');
      setTimeout(() => { if (listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight; }, 30);
    } catch (e: any) {
      setError(e.response?.data?.message || '添加失败');
    } finally {
      setSubmitting(false);
    }
  };

  const handleDelete = async (id: string) => {
    if (!window.confirm(t('common.confirm') || '确认?')) return;
    try {
      await api.delete(`/events/comments/${id}`);
      setItems(prev => prev.filter(i => i.id !== id));
    } catch (e: any) {
      setError(e.response?.data?.message || (t('common.deleteFailed') as string) || '删除失败');
    }
  };

  // 编辑逻辑
  const [editingId, setEditingId] = useState<string|null>(null);
  const [editingVal, setEditingVal] = useState('');
  const startEdit = (item: EventTimelineItem) => { setEditingId(item.id); setEditingVal(item.content); };
  const cancelEdit = () => { setEditingId(null); setEditingVal(''); };
  const submitEdit = async () => {
    if(!editingId || !editingVal.trim()) { cancelEdit(); return; }
    try { const res = await api.put(`/events/comments/${editingId}`, { content: editingVal.trim() }); const updated:EventTimelineItem = res.data; setItems(prev=> prev.map(i=> i.id===editingId? {...i, content: updated.content, updated_at: updated.updated_at}: i)); }
    catch(e:any){ setError(e.response?.data?.message || (t('common.saveFailed') as string) || '保存失败'); }
    finally { cancelEdit(); }
  };

  const grouped = useMemo(() => items, [items]); // 保留扩展点

  return (
    <div className="modal fade show d-block" tabIndex={-1} style={{ background: 'rgba(0,0,0,.45)' }}>
      <div className="modal-dialog modal-lg modal-dialog-scrollable">
        <div className="modal-content">
          <div className="modal-header py-2">
            <h5 className="modal-title">
              <i className="bi bi-clock-history me-2 text-primary" />时间线 - {eventTitle}
            </h5>
            <button className="btn btn-sm btn-outline-secondary" onClick={() => fetchTimeline()} disabled={loading} title="刷新">
              <i className="bi bi-arrow-clockwise" />
            </button>
            <button type="button" className="btn-close ms-2" onClick={onClose}></button>
          </div>
          <div className="modal-body d-flex flex-column" style={{paddingTop: '0.5rem'}}>
            {error && (
              <div className="alert alert-danger py-2 small d-flex justify-content-between align-items-center">
                <span>{error}</span>
                <button className="btn-close" onClick={() => setError(null)}></button>
              </div>
            )}
            <div className="border rounded flex-grow-1 mb-3 position-relative" ref={listRef} style={{ overflowY: 'auto', maxHeight: '55vh', background: 'var(--bs-body-bg)' }}>
              {loading && (
                <div className="d-flex justify-content-center py-5 text-muted small">加载中...</div>
              )}
              {!loading && grouped.length === 0 && (
                <div className="d-flex justify-content-center py-5 text-muted small">暂无记录</div>
              )}
              {!loading && grouped.length > 0 && (
                <ul className="timeline list-unstyled m-0 p-3">
                  {hasMore && (
                    <li className="text-center mb-2">
                      <button className="btn btn-outline-secondary btn-sm" disabled={loadingMore} onClick={() => fetchTimeline({ more: true })}>
                        {loadingMore ? '加载...' : '加载更早'}
                      </button>
                    </li>
                  )}
                  {grouped.map(item => {
                    const isSystem = item.type !== 'comment';
                    const icon = item.meta?.kind==='event_start'? 'bi-flag-fill': item.meta?.reminder_id? 'bi-bell-fill': isSystem? 'bi-gear-fill': 'bi-chat-dots';
                    return (
                      <li key={item.id} className="d-flex position-relative ps-4 pb-3 timeline-item">
                        <span className={"position-absolute top-0 start-0 translate-middle-y badge rounded-pill " + (isSystem ? 'bg-secondary' : 'bg-primary')} style={{ left: 0, top: '0.9rem' }}>&nbsp;</span>
                        <div className="flex-grow-1">
                          <div className="d-flex justify-content-between align-items-start gap-2">
                            <div className="small fw-semibold text-truncate d-flex align-items-center gap-1" style={{maxWidth:'60%'}}><i className={`bi ${icon}`}></i>{systemLabel(item)}</div>
                            <div className="text-muted small" title={new Date(item.created_at).toLocaleString()}>{humanTime(item.created_at)}</div>
                          </div>
                          {editingId===item.id ? (
                            <div className="mt-2">
                              <textarea className="form-control form-control-sm mb-2" rows={2} value={editingVal} onChange={e=> setEditingVal(e.target.value)} />
                              <div className="d-flex gap-2">
                                <button className="btn btn-primary btn-sm" onClick={submitEdit}>{t('common.save')||'保存'}</button>
                                <button className="btn btn-secondary btn-sm" onClick={cancelEdit}>{t('common.cancel')||'取消'}</button>
                              </div>
                            </div>
                          ) : (
                            <div className={"mt-1 small " + (isSystem ? 'text-muted fst-italic' : '')} style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
                              {item.content || (isSystem ? '(无内容)' : '')}
                            </div>
                          )}
                          {!isSystem && editingId!==item.id && (
                            <div className="mt-1 d-flex gap-3">
                              <button className="btn btn-link btn-sm p-0" onClick={()=> startEdit(item)}>{t('common.edit')||'编辑'}</button>
                              <button className="btn btn-link btn-sm p-0 text-danger" onClick={() => handleDelete(item.id)}>{t('common.delete')||'删除'}</button>
                            </div>
                          )}
                        </div>
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
            <form onSubmit={handleAdd} className="border rounded p-2 bg-light">
              <div className="input-group">
                <textarea
                  className="form-control"
                  placeholder="添加评论 (Enter 发送, Shift+Enter 换行)"
                  value={input}
                  rows={2}
                  onChange={e => setInput(e.target.value)}
                  onKeyDown={e => {
                    if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); if (!submitting) handleAdd(e as any); }
                  }}
                />
                <button className="btn btn-primary" type="submit" disabled={!input.trim() || submitting}>{submitting ? '发送中...' : '发送'}</button>
              </div>
            </form>
          </div>
        </div>
      </div>
      <style>{`
        .timeline { position: relative; }
        .timeline:before { content:''; position:absolute; left:10px; top:0; bottom:0; width:2px; background: var(--bs-border-color); }
        .timeline-item:last-child { padding-bottom:0; }
        .timeline-item:hover { background: rgba(0,0,0,0.015); border-radius:4px; }
      `}</style>
    </div>
  );
};

export default EventTimelineModal;
