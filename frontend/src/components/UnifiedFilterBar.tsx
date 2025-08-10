import React, { useState } from 'react';

export interface UnifiedFilterValues {
  sources: string[];
  hours: number;
  limit?: number;
  minSeverity?: number; // severity score threshold
}

interface Props {
  initial?: Partial<UnifiedFilterValues>;
  onApply: (values: UnifiedFilterValues) => void;
  onReset?: (values: UnifiedFilterValues) => void;
  compact?: boolean;
}

const ALL_SOURCES = ['task','event','reminder'];

export const UnifiedFilterBar: React.FC<Props> = ({ initial, onApply, onReset, compact }) => {
  const [sources, setSources] = useState<string[]>(initial?.sources || ALL_SOURCES);
  const [hours, setHours] = useState<number>(initial?.hours || 24*7);
  const [limit, setLimit] = useState<number | ''>(initial?.limit || '');
  const [minSeverity, setMinSeverity] = useState<number>(initial?.minSeverity || 0);

  const toggleSource = (s: string) => { setSources(p => p.includes(s) ? p.filter(x=>x!==s) : [...p, s]); };
  const apply = () => { onApply({ sources, hours, limit: typeof limit==='number'? limit: undefined, minSeverity }); };
  const reset = () => {
    const v: UnifiedFilterValues = { sources: ALL_SOURCES, hours: 24*7, limit: undefined, minSeverity: 0 };
    setSources(v.sources); setHours(v.hours); setLimit(''); setMinSeverity(0);
    onReset?.(v); onApply(v);
  };

  return (
    <div className={`card mb-3 ${compact? 'py-2':''}`}>
      <div className="card-body py-3">
        <div className="row g-3 align-items-end">
          <div className="col-md-3">
            <label className="form-label mb-1 small fw-semibold">Sources</label>
            <div className="d-flex flex-wrap gap-2">
              {ALL_SOURCES.map(s => (
                <button key={s} type="button" onClick={()=>toggleSource(s)} className={`btn btn-sm ${sources.includes(s)?'btn-primary':'btn-outline-secondary'}`}>{s}</button>
              ))}
            </div>
          </div>
          <div className="col-md-2">
            <label className="form-label mb-1 small fw-semibold">Hours</label>
            <div className="d-flex gap-2">
              <input type="number" className="form-control form-control-sm" value={hours} min={1} max={24*90} onChange={e=>setHours(Number(e.target.value)||24)} />
              <div className="btn-group btn-group-sm" role="group">
                <button type="button" className={`btn btn-outline-secondary ${hours===24*7?'active':''}`} onClick={()=>setHours(24*7)}>7d</button>
                <button type="button" className={`btn btn-outline-secondary ${hours===24*30?'active':''}`} onClick={()=>setHours(24*30)}>30d</button>
              </div>
            </div>
            <div className="form-text small">范围: 1h - 2160h(90d)</div>
          </div>
          <div className="col-md-2">
            <label className="form-label mb-1 small fw-semibold">Limit</label>
            <input type="number" className="form-control form-control-sm" value={limit} min={1} max={500} onChange={e=>setLimit(e.target.value?Number(e.target.value):'')} />
          </div>
          <div className="col-md-3">
            <label className="form-label mb-1 small fw-semibold">Min Severity</label>
            <input type="range" className="form-range" min={0} max={30} value={minSeverity} onChange={e=>setMinSeverity(Number(e.target.value))} />
            <div className="small text-muted">score ≥ {minSeverity}</div>
          </div>
          <div className="col-md-2 text-end">
            <button className="btn btn-sm btn-primary me-2" onClick={apply}><i className="bi bi-funnel me-1"/>Apply</button>
            <button className="btn btn-sm btn-outline-secondary" onClick={reset}>Reset</button>
          </div>
        </div>
      </div>
    </div>
  );
};

export default UnifiedFilterBar;
