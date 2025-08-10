import React, { useEffect, useState, useMemo, useRef } from 'react';
import { useTranslation } from 'react-i18next';
// import { fetchUnifiedUpcoming } from '../features/unified/unifiedSlice'; // 已迁移至 RTK Query
import { useGetUpcomingQuery, type UnifiedUpcomingItem } from '../features/unified/unifiedApi';
import SeverityBadge from '../components/SeverityBadge';
import DataState from '../components/DataState';
import UnifiedFilterBar, { type UnifiedFilterValues } from '../components/UnifiedFilterBar';
import { computeSeverity } from '../components/SeverityBadge';

const formatCountdown = (seconds: number) => {
  if (seconds < 0) return '-';
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  if (d > 0) return `${d}d ${h}h`;
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
};

const badgeClassFor = (secs: number, source: string) => {
  if (secs < 0) return 'bg-secondary';
  const h = secs / 3600;
  if (h <= 24) return 'bg-danger';
  if (h <= 72) return 'bg-warning text-dark';
  if (source === 'task') return 'bg-info text-dark';
  return 'bg-primary';
};

const severityThresholds = [0, 8, 14, 20]; // 与 computeSeverity 的 level 边界对应 (low/medium/high/critical)

const UnifiedBoardPage: React.FC = () => {
  const { t } = useTranslation();
  // 实时刷新使用 serverOffset 基础上动态计算，不再单独存 now (使用 Date.now())
  const [serverOffset, setServerOffset] = useState<number>(0); // 客户端时间 - 服务器时间 (ms)
  const dataArrivalRef = useRef<number>(0);

  const defaultFilters: UnifiedFilterValues = { sources:['task','event','reminder'], hours:24*7, limit: undefined, minSeverity:0 };
  const [filterState, setFilterState] = useState<UnifiedFilterValues>(defaultFilters);
  const [groupByDay, setGroupByDay] = useState<boolean>(false);
  const [compact, setCompact] = useState<boolean>(false);
  const [showDebug, setShowDebug] = useState<boolean>(false);
  const applyFilters = (vals: UnifiedFilterValues) => { setFilterState(vals); };
  const queryArgs = { hours: filterState.hours, sources: filterState.sources.length===3 ? undefined : filterState.sources, limit: filterState.limit } as const;
  const { data: upcomingData, isFetching: isLoading, error: loadErr, refetch } = useGetUpcomingQuery(queryArgs, { refetchOnMountOrArgChange: true });
  const items = upcomingData?.items || [];
  // 服务器时间偏差：首次拿到数据时计算 (一次即可，后续可根据 server_timestamp 变化更新)
  useEffect(()=>{
    if (upcomingData?.server_timestamp) {
      const clientNow = Date.now();
      dataArrivalRef.current = clientNow;
      const serverMs = upcomingData.server_timestamp * 1000;
      setServerOffset(clientNow - serverMs);
    }
  }, [upcomingData?.server_timestamp]);
  const error = loadErr ? (()=>{
    if (typeof (loadErr as any).status === 'number') {
      const errAny: any = loadErr;
      if (errAny?.data?.message) return errAny.data.message;
      return `HTTP ${errAny.status}`;
    }
    return 'Load failed';
  })() : null;
  useEffect(() => { /* 触发依赖 queryArgs 自动 */ }, [queryArgs]);
  // 可选的定期刷新倒计时 UI：通过 refetch 或强制渲染；此处略。

  // 预计算添加 severity 分数，避免多次 compute
  const enriched: (UnifiedUpcomingItem & { _severity: number })[] = useMemo(()=>{
    return items.map(it => {
      const sev = computeSeverity({
        source: it.source,
        scheduledAt: it.scheduled_at,
        importance: it.importance,
        priorityScore: it.priority_score
      });
      return { ...it, _severity: sev.score };
    });
  }, [items]);

  const filteredItems = enriched.filter(it => !filterState.minSeverity || it._severity >= filterState.minSeverity);

  // 衍生统计数据 (基于过滤 + 排序前)
  const derivedStats = useMemo(()=>{
    const total = filteredItems.length;
    const nowBase = Date.now() - serverOffset;
    let overdue = 0; let critical = 0; let within24h = 0;
    filteredItems.forEach(it => {
      const schedMs = new Date(it.scheduled_at).getTime();
      if (schedMs < nowBase) overdue++;
      if (it._severity >= 20) critical++; // 参照 severityThresholds
      if (schedMs - nowBase <= 24*3600*1000 && schedMs - nowBase >= 0) within24h++;
    });
    return { total, overdue, critical, within24h };
  }, [filteredItems, serverOffset]);

  // 排序
  type SortKey = 'scheduled_at' | 'severity';
  const [sortKey, setSortKey] = useState<SortKey>('scheduled_at');
  const [sortDir, setSortDir] = useState<'asc' | 'desc'>('asc');
  const toggleSort = (key: SortKey) => {
    setSortKey(prev => {
      if (prev !== key) { setSortDir('asc'); return key; }
      setSortDir(d => d === 'asc' ? 'desc' : 'asc');
      return key;
    });
  };
  const sortedItems = useMemo(()=>{
    const arr = [...filteredItems];
    arr.sort((a,b)=>{
      if (sortKey === 'scheduled_at') {
        const ta = new Date(a.scheduled_at).getTime();
        const tb = new Date(b.scheduled_at).getTime();
        return sortDir === 'asc' ? ta - tb : tb - ta;
      } else { // severity
        return sortDir === 'asc' ? a._severity - b._severity : b._severity - a._severity;
      }
    });
    return arr;
  }, [filteredItems, sortKey, sortDir]);

  // 分组 (仅在已排序后保持顺序) key: YYYY-MM-DD
  const grouped = useMemo(()=>{
    if (!groupByDay) return null;
    const map: Record<string, typeof sortedItems> = {};
    sortedItems.forEach(it => {
      const d = new Date(it.scheduled_at);
      const key = d.toISOString().slice(0,10);
      (map[key] ||= []).push(it);
    });
    // 维持日期顺序
    return Object.keys(map).sort().map(date => ({ date, items: map[date] }));
  }, [groupByDay, sortedItems]);

  const stats = upcomingData?.stats;

  const applySeverityShortcut = (score: number) => setFilterState(s => ({ ...s, minSeverity: score }));

  // 导出 CSV（基于当前可见排序后列表）
  const exportCSV = () => {
    const header = ['source','title','scheduled_at','countdown_seconds','days_left','severity_score','importance','priority_score'];
    const lines = [header.join(',')];
    const baseNow = Date.now() - serverOffset; // server 时间估算
    sortedItems.forEach(it => {
      const sched = new Date(it.scheduled_at).getTime();
      const countdown = Math.max(0, Math.floor((sched - baseNow)/1000));
      const row = [it.source, JSON.stringify(it.title||''), new Date(it.scheduled_at).toISOString(), String(countdown), String(it.days_left), String(it._severity), String(it.importance ?? ''), String(it.priority_score ?? '')];
      lines.push(row.join(','));
    });
    const blob = new Blob([lines.join('\n')], { type: 'text/csv;charset=utf-8;' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `unified_upcoming_${Date.now()}.csv`;
    a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <div className="container py-4 panel-wrap">
      <div className="panel-overlay" />
      <div className="panel-content">
      <div className="d-flex flex-wrap align-items-center gap-3 mb-3">
        <h2 className="mb-0">{t('unified.title')}</h2>
        <div className="ms-auto d-flex flex-wrap gap-2 align-items-center">
          <div className="small text-muted" title={`Server offset ≈ ${serverOffset}ms (client-server)`}>
            <i className="bi bi-clock-history me-1" />Δ {serverOffset >=0? '+':''}{Math.round(serverOffset/1000)}s
          </div>
          {stats && (
            <div className="small text-muted">
              <i className="bi bi-bar-chart me-1"/>
              tasks {stats.tasks} / events {stats.events} / reminders {stats.reminders}
            </div>
          )}
          <div className="btn-group btn-group-sm" role="group" aria-label="Hours shortcuts">
            {[24, 72, 168].map(h => (
              <button key={h} type="button" className={`btn ${filterState.hours===h? 'btn-primary':'btn-outline-secondary'}`} onClick={()=>setFilterState(s=>({...s, hours:h}))}>{h<=24? `${h}h`: `${Math.round(h/24)}d`}</button>
            ))}
          </div>
          <div className="btn-group btn-group-sm" role="group" aria-label="Severity shortcuts">
            {severityThresholds.map(th => (
              <button key={th} type="button" className={`btn ${filterState.minSeverity===th? 'btn-primary':'btn-outline-secondary'}`} onClick={()=>applySeverityShortcut(th)} title={`score ≥ ${th}`}>{th===0? 'ALL': th}</button>
            ))}
          </div>
          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=>setGroupByDay(g=>!g)} title="Group by day">
            <i className="bi bi-calendar3 me-1"/>{groupByDay? 'Ungroup':'Group'}
          </button>
          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=>setCompact(c=>!c)} title="Compact mode">
            <i className="bi bi-aspect-ratio me-1"/>{compact? 'Normal':'Compact'}
          </button>
          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={()=>setShowDebug(d=>!d)} title="Debug panel">
            <i className="bi bi-bug me-1"/>{showDebug? 'Debug-':'Debug+'}
          </button>
          <button type="button" className="btn btn-sm btn-outline-primary" onClick={()=>refetch()} disabled={isLoading}>
            <i className={`bi bi-arrow-repeat me-1 ${isLoading? 'spin':''}`}></i>{isLoading? t('common.loading')||'Loading': t('common.refresh')||'Refresh'}
          </button>
          <button type="button" className="btn btn-sm btn-outline-secondary" onClick={exportCSV} title="Export CSV">
            <i className="bi bi-download me-1"/>CSV
          </button>
          <button type="button" className="btn btn-sm btn-outline-danger" onClick={()=>setFilterState(defaultFilters)} title="Reset filters">
            <i className="bi bi-x-circle me-1"/>Reset
          </button>
        </div>
      </div>

      <UnifiedFilterBar initial={filterState} onApply={applyFilters} />
      <div className="row g-3 mb-3">
        <div className="col-auto">
          <div className="card shadow-sm border-0 bg-light small">
            <div className="card-body py-2 px-3">
              <div className="fw-semibold">Total</div>
              <div className="fs-5">{derivedStats.total}</div>
            </div>
          </div>
        </div>
        <div className="col-auto">
          <div className="card shadow-sm border-0 bg-light small">
            <div className="card-body py-2 px-3">
              <div className="fw-semibold text-danger">Overdue</div>
              <div className="fs-5">{derivedStats.overdue}</div>
            </div>
          </div>
        </div>
        <div className="col-auto">
          <div className="card shadow-sm border-0 bg-light small">
            <div className="card-body py-2 px-3">
              <div className="fw-semibold text-warning">24h</div>
              <div className="fs-5">{derivedStats.within24h}</div>
            </div>
          </div>
        </div>
        <div className="col-auto">
          <div className="card shadow-sm border-0 bg-light small">
            <div className="card-body py-2 px-3">
              <div className="fw-semibold text-primary">Critical</div>
              <div className="fs-5">{derivedStats.critical}</div>
            </div>
          </div>
        </div>
      </div>
      <DataState
        loading={isLoading}
        error={error}
        data={filteredItems}
        emptyHint={<div className="text-muted small p-4 border rounded bg-light">
          <div className="mb-2">{t('unified.empty')} / No upcoming items.</div>
          <ul className="mb-2">
            <li>扩大时间窗口 (当前 {filterState.hours}h)</li>
            <li>检查是否有任务 / 事件 / 提醒未设置时间</li>
            <li>降低最小严重度 (当前 ≥ {filterState.minSeverity})</li>
          </ul>
          <div className="d-flex gap-2 flex-wrap">
            {[24,72,168, 24*14].map(h=> <button key={h} className="btn btn-sm btn-outline-secondary" onClick={()=>setFilterState(s=>({...s, hours:h}))}>{h<=24? `${h}h`: `${Math.round(h/24)}d`}</button> )}
            <button className="btn btn-sm btn-outline-danger" onClick={()=>setFilterState(defaultFilters)}>Reset</button>
          </div>
        </div>}
        skeleton={
          <div className="placeholder-glow">
            <div className="table-responsive">
              <table className="table table-sm">
                <thead>
                  <tr>
                    <th style={{width:'6%'}}><span className="placeholder col-8" /></th>
                    <th style={{width:'34%'}}><span className="placeholder col-10" /></th>
                    <th style={{width:'22%'}}><span className="placeholder col-6" /></th>
                    <th style={{width:'14%'}}><span className="placeholder col-6" /></th>
                    <th style={{width:'12%'}}><span className="placeholder col-5" /></th>
                    <th style={{width:'12%'}}><span className="placeholder col-5" /></th>
                  </tr>
                </thead>
                <tbody>
                  {Array.from({length:6}).map((_,i)=>(
                    <tr key={i}>
                      <td><span className="badge bg-secondary placeholder col-8" style={{opacity:0.5}}>&nbsp;</span></td>
                      <td><span className="placeholder col-9" /></td>
                      <td><span className="placeholder col-7" /></td>
                      <td><span className="placeholder col-5" /></td>
                      <td><span className="placeholder col-4" /></td>
                      <td><span className="placeholder col-4" /></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </div>
        }
      >
  {()=> (
          <div className="table-responsive">
            <table className="table table-sm align-middle table-hover">
              <thead>
                <tr>
                  <th>{t('unified.type')}</th>
                  <th>{t('unified.titleCol')}</th>
                  <th role="button" onClick={()=>toggleSort('scheduled_at')} className="text-nowrap">{t('unified.schedule')} {sortKey==='scheduled_at' && (sortDir==='asc'? '▲':'▼')}</th>
                  <th>{t('unified.countdown')}</th>
                  <th>{t('unified.daysLeft')}</th>
                  <th role="button" onClick={()=>toggleSort('severity')} className="text-nowrap">{t('unified.severity', 'Severity')} {sortKey==='severity' && (sortDir==='asc'? '▲':'▼')}</th>
                </tr>
              </thead>
              <tbody>
                {!groupByDay && sortedItems.map((it: any) => {
                  const scheduled = new Date(it.scheduled_at);
                  const baseNow = Date.now() - serverOffset; // 校准
                  const diffSecs = Math.floor((scheduled.getTime() - baseNow)/1000);
                  const absSecs = Math.abs(diffSecs);
                  const countdownCls = badgeClassFor(absSecs, it.source);
                  const overdue = diffSecs < 0;
                  return (
                    <tr key={`${it.source}-${it.id}`} className={`${overdue? 'table-warning':''} position-relative ${compact? 'small':''}`}>
                      <td><span className="badge bg-secondary text-uppercase" title={it.source}>{it.source}</span></td>
                      <td className="position-relative">
                        {it.detail_url ? <a href={`${it.detail_url}?focus=${it.source_id || it.id}&source=${it.source}`} className="text-decoration-none">{it.title || '-'}</a> : (it.title || '-')}
                        <div className="row-actions small text-nowrap">
                          {it.detail_url && <a href={`${it.detail_url}?focus=${it.source_id || it.id}&source=${it.source}`} className="btn btn-link btn-sm p-0 me-2" title="Open"><i className="bi bi-box-arrow-up-right"/></a>}
                          <button className="btn btn-link btn-sm p-0 me-2" title="Copy ID" onClick={()=>navigator.clipboard.writeText(it.id)}><i className="bi bi-clipboard"/></button>
                        </div>
                      </td>
                      <td title={scheduled.toISOString()}>{scheduled.toLocaleString()}</td>
                      <td>
                        <span className={`badge ${countdownCls}`} title={overdue? 'Overdue':'Countdown'}>
                          {overdue? '-' + formatCountdown(absSecs): formatCountdown(absSecs)}
                        </span>
                      </td>
                      <td>{it.days_left}</td>
                      <td><SeverityBadge source={it.source} scheduledAt={scheduled} importance={it.importance} priorityScore={it.priority_score} showLabel={false} /></td>
                    </tr>
                  );
                })}
                {groupByDay && grouped && grouped.map(gr => (
                  <React.Fragment key={gr.date}>
                    <tr className="table-active"><td colSpan={6} className="fw-semibold text-primary">{gr.date} <span className="small text-muted ms-2">{gr.items.length} items</span></td></tr>
                    {gr.items.map(it => {
                      const scheduled = new Date(it.scheduled_at);
                      const baseNow = Date.now() - serverOffset; // 校准
                      const diffSecs = Math.floor((scheduled.getTime() - baseNow)/1000);
                      const absSecs = Math.abs(diffSecs);
                      const countdownCls = badgeClassFor(absSecs, it.source);
                      const overdue = diffSecs < 0;
                      return (
                        <tr key={`${it.source}-${it.id}`} className={`${overdue? 'table-warning':''} position-relative ${compact? 'small':''}`}>
                          <td><span className="badge bg-secondary text-uppercase" title={it.source}>{it.source}</span></td>
                          <td className="position-relative">
                            {it.detail_url ? <a href={`${it.detail_url}?focus=${it.source_id || it.id}&source=${it.source}`} className="text-decoration-none">{it.title || '-'}</a> : (it.title || '-')}
                            <div className="row-actions small text-nowrap">
                              {it.detail_url && <a href={`${it.detail_url}?focus=${it.source_id || it.id}&source=${it.source}`} className="btn btn-link btn-sm p-0 me-2" title="Open"><i className="bi bi-box-arrow-up-right"/></a>}
                              <button className="btn btn-link btn-sm p-0 me-2" title="Copy ID" onClick={()=>navigator.clipboard.writeText(it.id)}><i className="bi bi-clipboard"/></button>
                            </div>
                          </td>
                          <td title={scheduled.toISOString()}>{scheduled.toLocaleString()}</td>
                          <td>
                            <span className={`badge ${countdownCls}`} title={overdue? 'Overdue':'Countdown'}>
                              {overdue? '-' + formatCountdown(absSecs): formatCountdown(absSecs)}
                            </span>
                          </td>
                          <td>{it.days_left}</td>
                          <td><SeverityBadge source={it.source} scheduledAt={scheduled} importance={it.importance} priorityScore={it.priority_score} showLabel={false} /></td>
                        </tr>
                      );
                    })}
                  </React.Fragment>
                ))}
                {(!isLoading && !error && filteredItems.length===0) && (
                  <tr>
                    <td colSpan={6} className="text-center text-muted small">No data returned from API. Try increasing hours or remove filters.</td>
                  </tr>
                )}
              </tbody>
            </table>
            <style>{`
              .row-actions { opacity:0; transition: opacity .15s ease; position:absolute; right:4px; top:50%; transform:translateY(-50%); }
              tr:hover .row-actions { opacity:1; }
              .spin { animation: spin 1s linear infinite; }
              @keyframes spin { from { transform: rotate(0deg);} to { transform: rotate(360deg);} }
              .table-hover tbody tr.table-active:hover { background-color: var(--bs-table-active-bg); }
            `}</style>
          </div>
        )}
      </DataState>
      {showDebug && (
        <div className="mt-4">
          <div className="card card-body small">
            <div className="fw-semibold mb-2">Debug</div>
            <pre className="small mb-0" style={{maxHeight:300, overflow:'auto'}}>{JSON.stringify({queryArgs, filterState, derivedStats, grouped: groupByDay? grouped?.map(g=>({date:g.date, count:g.items.length})): undefined}, null, 2)}</pre>
          </div>
        </div>
      )}
      </div>
    </div>
  );
};

export default UnifiedBoardPage;
