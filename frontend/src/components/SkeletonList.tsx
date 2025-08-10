import React from 'react';

interface SkeletonListProps { count?: number; lines?: number; className?: string }
const SkeletonList: React.FC<SkeletonListProps> = ({ count = 6, lines = 3, className }) => {
  const items = Array.from({ length: count });
  return (
    <div className={className}>
      {items.map((_, idx) => (
        <div key={idx} className="mb-3 p-3 border rounded bg-light position-relative overflow-hidden" style={{minHeight: lines*14+12}}>
          <div className="placeholder-wave" style={{width:'100%'}}>
            {Array.from({length: lines}).map((_, l) => (
              <span key={l} className="placeholder col-" style={{display:'block', width: `${60 + (l*10)%30}%`, height: '10px', marginBottom: '6px'}}></span>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
};
export default SkeletonList;
