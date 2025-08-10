import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useSelector } from 'react-redux';
import type { RootState } from '../app/store';
import { useTranslation } from 'react-i18next';
import { useParams, Link, useNavigate } from 'react-router-dom';
import api from '../config/api';
import SeverityBadge from '../components/SeverityBadge';

interface EventDetail {
  id: string;
  title: string;
  description: string;
  event_type: string;
  event_date: string;
  importance_level: number;
  location?: string;
  recurrence_type: string;
  is_all_day: boolean;
  created_at: string;
  updated_at: string;
}

interface TimelineItem {
  id: string;
  event_id: string;
  user_id: string;
  type: string;
  content: string;
  meta?: Record<string,string>;
  created_at: string;
  updated_at: string;
}

const humanTime = (iso: string) => {
  const d = new Date(iso); const diff = (Date.now()-d.getTime())/1000;
  if (diff < 60) return '刚刚';
  const m = Math.floor(diff/60); if (m<60) return m+' 分钟前';
  const h = Math.floor(m/60); if (h<24) return h+' 小时前';
  const day = Math.floor(h/24); if (day<7) return day+' 天前';
  return d.toLocaleString();
};
const systemLabel = (it:TimelineItem) => {
  if (it.meta?.kind === 'event_start') return '事件开始';
  if (it.meta?.reminder_id) return '提醒已发送';
  switch(it.type){ case 'system': return '系统'; case 'status_change': return '状态变更'; default: return '评论'; }
};

const EventDetailPage: React.FC = () => {
  const { id } = useParams();
  const { t } = useTranslation();
  const notifications = useSelector((s:RootState)=> s.notifications.items);
  const navigate = useNavigate();
  const [event, setEvent] = useState<EventDetail | null>(null);
  const [loadingEvent, setLoadingEvent] = useState(true);
  const [timeline, setTimeline] = useState<TimelineItem[]>([]);
  const [loadingTL, setLoadingTL] = useState(false); // 仅首次加载或显式全量刷新时使用
  const [refreshing, setRefreshing] = useState(false); // 局部静默刷新，不清空列表避免闪烁
  const [error, setError] = useState<string | null>(null);
  const [tlError, setTlError] = useState<string | null>(null);
  const [input, setInput] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const listRef = useRef<HTMLDivElement|null>(null);

  const fetchEvent = useCallback(async ()=>{
    if(!id) return; setLoadingEvent(true); setError(null);
    try { const res = await api.get(`/events/${id}`); setEvent(res.data); } catch(e:any){ setError(e.response?.data?.message||'加载事件失败'); }
    finally { setLoadingEvent(false); }
  },[id]);

  const fetchTimeline = useCallback(async (opts?:{more?:boolean; keepScroll?:boolean; silent?:boolean})=>{
    if(!id) return; if(loadingMore) return; // loadingTL 只用于初始/手动全量
    const isMore=!!opts?.more; const silent = !!opts?.silent; const initial = !isMore && timeline.length===0 && !silent;
    if(isMore) setLoadingMore(true); else if(initial) setLoadingTL(true); else if(!silent) setRefreshing(true);
    setTlError(null);
    try {
      const params = new URLSearchParams(); params.set('limit','60'); if(isMore && timeline.length) params.set('before_id', timeline[0].id);
      const res = await api.get(`/events/${id}/timeline?`+params.toString()); const list:TimelineItem[] = res.data.items||[];
      if(isMore){ if(list.length===0) setHasMore(false); setTimeline(prev=>[...list,...prev]); }
      else {
        // 如果是静默/刷新，做增量 diff，避免整列表替换引发闪烁
        if(timeline.length && (silent || !initial)) {
          const existingIds = new Set(timeline.map(i=>i.id));
            const toAppend = list.filter(i=> !existingIds.has(i.id));
            if(toAppend.length){ setTimeline(prev=> [...prev, ...toAppend]); }
            // 可选：如果服务器返回被裁剪(丢最老的)不处理，保持已有以减少 DOM 抖动
        } else {
          setTimeline(list);
          if(!opts?.keepScroll){ setTimeout(()=>{ if(listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight; }, 40);} }
        setHasMore(list.length===60);
      }
    }
    catch(e:any){ setTlError(e.response?.data?.message||'加载时间线失败'); }
    finally {
      if(isMore) setLoadingMore(false); else if(initial) setLoadingTL(false); else if(!silent) setRefreshing(false);
    }
  },[id, timeline, loadingMore]);

  useEffect(()=>{ fetchEvent(); fetchTimeline(); },[fetchEvent, fetchTimeline]);

  // 监听通知中 timeline_event 自动加载最新
  useEffect(()=> {
    if(!id) return; const latestTimelineNotif = notifications.find(n=> n.type==='timeline_event' && n.event_id===id);
    if(!latestTimelineNotif) return;
    // 静默增量刷新: 只获取并追加新项，不触发 loading 占位
    fetchTimeline({keepScroll:true, silent:true});
  }, [notifications, id, fetchTimeline]);

  // 移除定时轮询：靠 SSE 通知和手动刷新，减少重绘（避免与 backdrop-filter 叠加闪烁）

  const handleAdd = async (e:React.FormEvent) => {
    e.preventDefault(); if(!id || !input.trim()) return; setSubmitting(true);
    try { const res = await api.post(`/events/${id}/comments`, { content: input.trim(), type:'comment'}); const added:TimelineItem = res.data; setTimeline(prev=>[...prev, added]); setInput(''); setTimeout(()=>{ if(listRef.current) listRef.current.scrollTop = listRef.current.scrollHeight; }, 30); }
    catch(e:any){ setTlError(e.response?.data?.message||'添加失败'); }
    finally { setSubmitting(false); }
  };

  const handleDelete = async (cid:string) => { if(!window.confirm(t('common.confirm')||'确认?')) return; try { await api.delete(`/events/comments/${cid}`); setTimeline(prev=> prev.filter(i=> i.id!==cid)); } catch(e:any){ setTlError(e.response?.data?.message||t('common.deleteFailed')||'删除失败'); } };

  // 编辑流程
  const [editingId, setEditingId] = useState<string|null>(null);
  const [editingVal, setEditingVal] = useState('');
  const startEdit = (item:TimelineItem) => { setEditingId(item.id); setEditingVal(item.content); };
  const cancelEdit = () => { setEditingId(null); setEditingVal(''); };
  const submitEdit = async () => {
    if(!editingId || !editingVal.trim()) { cancelEdit(); return; }
    try { const res = await api.put(`/events/comments/${editingId}`, { content: editingVal.trim() }); const updated:TimelineItem = res.data; setTimeline(prev=> prev.map(i=> i.id===editingId? {...i, content: updated.content, updated_at: updated.updated_at}: i)); }
    catch(e:any){ setTlError(e.response?.data?.message||t('common.saveFailed')||'保存失败'); }
    finally { cancelEdit(); }
  };

  const countdown = useMemo(()=>{ if(!event) return ''; const diff = new Date(event.event_date).getTime() - Date.now(); const past = diff <0; const abs = Math.abs(diff); const d = Math.floor(abs/86400000); const h = Math.floor((abs%86400000)/3600000); const m = Math.floor((abs%3600000)/60000); if(d>0) return (past? '已开始 ':'倒计时 ')+ `${d}天${h}小时`; if(h>0) return (past? '已开始 ':'倒计时 ')+ `${h}小时${m}分`; const s=Math.floor((abs%60000)/1000); return (past? '已开始 ':'倒计时 ')+ `${m}分${s}秒`; },[event]);

  return (
    <div className="container-fluid mt-4 panel-wrap">
      <div className="panel-content">
        <div className="d-flex align-items-center mb-3 gap-2 flex-wrap">
          <button className="btn btn-outline-secondary btn-sm" onClick={()=> navigate(-1)}><i className="bi bi-arrow-left"/> 返回</button>
          <h2 className="h4 mb-0"><i className="bi bi-calendar-event me-2"/>事件详情</h2>
          {event && <SeverityBadge source="event" scheduledAt={event.event_date} importance={event.importance_level} showLabel={true} />}
          {event && <span className="badge bg-light text-dark border fw-normal">{countdown}</span>}
          <Link className="btn btn-outline-primary btn-sm" to="/events"><i className="bi bi-list-ul"/> 所有事件</Link>
        </div>
        {error && <div className="alert alert-danger py-2 small">{error}</div>}
        {loadingEvent && <div className="placeholder-wave"><div className="placeholder col-6 mb-2"/><div className="placeholder col-4"/></div>}
        {event && !loadingEvent && (
          <div className="card mb-4 shadow-sm">
            <div className="card-body">
              <h5 className="card-title mb-2">{event.title}</h5>
              {event.description && <p className="text-muted small mb-2" style={{whiteSpace:'pre-wrap'}}>{event.description}</p>}
              <div className="small text-secondary d-flex flex-wrap gap-3">
                <span><i className="bi bi-clock me-1"/> {new Date(event.event_date).toLocaleString()}</span>
                <span><i className="bi bi-tag me-1"/> {event.event_type}</span>
                {event.location && <span><i className="bi bi-geo-alt me-1"/> {event.location}</span>}
                {event.recurrence_type!=='none' && <span><i className="bi bi-arrow-repeat me-1"/> {event.recurrence_type}</span>}
              </div>
            </div>
          </div>
        )}
        <div className="d-flex align-items-center mb-2 gap-2">
          <h3 className="h6 mb-0"><i className="bi bi-clock-history me-2"/>时间线</h3>
          <button className="btn btn-sm btn-outline-secondary d-flex align-items-center gap-1" disabled={loadingTL||refreshing} onClick={()=> fetchTimeline()}>
            <i className={`bi ${refreshing? 'bi-arrow-repeat spin':'bi-arrow-clockwise'}`}/>{refreshing? '刷新中':''}
          </button>
        </div>
        {tlError && <div className="alert alert-warning py-2 small d-flex justify-content-between"><span>{tlError}</span><button className="btn-close" onClick={()=> setTlError(null)}/></div>}
        <div className="card mb-4">
          <div className="card-body p-0">
            <div ref={listRef} style={{maxHeight:'55vh', overflowY:'auto'}} className="timeline-scroll position-relative">
              {loadingTL && timeline.length===0 && <div className="py-5 text-center text-muted small">加载中...</div>}
              {!loadingTL && timeline.length===0 && <div className="py-5 text-center text-muted small">暂无记录</div>}
              {timeline.length>0 && (
                <ul className="event-timeline list-unstyled m-0 p-3">
                  {hasMore && <li className="text-center mb-2"><button className="btn btn-outline-secondary btn-sm" disabled={loadingMore} onClick={()=> fetchTimeline({more:true})}>{loadingMore?'加载...':'加载更早'}</button></li>}
                  {timeline.map(item=>{
                    const isSystem = item.type!=='comment';
                    const icon = item.meta?.kind==='event_start'? 'bi-flag-fill': item.meta?.reminder_id? 'bi-bell-fill': isSystem? 'bi-gear-fill': 'bi-chat-dots';
                    return (
                      <li key={item.id} className="timeline-entry position-relative ps-4 pb-4">
                        <span className={"timeline-dot position-absolute rounded-circle "+(isSystem? 'bg-secondary':'bg-primary')} />
                        <div className="d-flex justify-content-between align-items-start gap-2 flex-wrap">
                          <div className="small fw-semibold text-truncate d-flex align-items-center gap-1" style={{maxWidth:'60%'}}>
                            <i className={`bi ${icon}`}></i> {systemLabel(item)}
                          </div>
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
                          <div className={'mt-1 small '+(isSystem?'text-muted fst-italic':'')} style={{whiteSpace:'pre-wrap', wordBreak:'break-word'}}>{item.content || (isSystem? '(无内容)':'')}</div>
                        )}
                        {!isSystem && editingId!==item.id && (
                          <div className="mt-1 d-flex gap-3">
                            <button className="btn btn-link btn-sm p-0" onClick={()=> startEdit(item)}>{t('common.edit')||'编辑'}</button>
                            <button className="btn btn-link btn-sm p-0 text-danger" onClick={()=> handleDelete(item.id)}>{t('common.delete')||'删除'}</button>
                          </div>
                        )}
                      </li>
                    );
                  })}
                </ul>
              )}
            </div>
            <form onSubmit={handleAdd} className="border-top p-3 bg-light">
              <div className="input-group">
                <textarea className="form-control" rows={2} placeholder="添加评论 (Enter 发送, Shift+Enter 换行)" value={input} onChange={e=> setInput(e.target.value)} onKeyDown={e=> { if(e.key==='Enter' && !e.shiftKey){ e.preventDefault(); if(!submitting) handleAdd(e as any);} }} />
                <button className="btn btn-primary" disabled={!input.trim()||submitting}>{submitting? '发送中...':'发送'}</button>
              </div>
            </form>
          </div>
        </div>
      </div>
      <style>{`
        .event-timeline { position:relative; }
        .event-timeline:before { content:''; position:absolute; left:12px; top:0; bottom:0; width:2px; background: var(--bs-border-color); }
        .timeline-entry { background:transparent; }
        .timeline-entry:hover { background: rgba(0,0,0,0.02); border-radius:4px; }
        .timeline-dot { width:10px; height:10px; left:8px; top:9px; }
        .spin { animation: spin 1s linear infinite; }
        @keyframes spin { from { transform: rotate(0deg);} to { transform: rotate(360deg);} }
      `}</style>
    </div>
  );
};

export default EventDetailPage;
