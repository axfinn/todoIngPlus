import React from 'react';

export type SeverityLevel = 'critical' | 'high' | 'medium' | 'low';

export interface SeverityInput {
  source: string;
  scheduledAt?: Date | string;
  importance?: number; // 1-5
  priorityScore?: number; // numeric (0-100?)
}

function normalizeDate(d?: Date | string): Date | undefined {
  if (!d) return undefined;
  if (d instanceof Date) return d;
  const dt = new Date(d);
  if (isNaN(dt.getTime())) return undefined;
  return dt;
}

export interface SeverityResult { level: SeverityLevel; score: number; variant: string; label: string; }

export function computeSeverity(input: SeverityInput, now: Date = new Date()): SeverityResult {
  const { scheduledAt, importance, priorityScore } = input;
  const sched = normalizeDate(scheduledAt);
  let hoursLeft = Infinity;
  if (sched) hoursLeft = (sched.getTime() - now.getTime()) / 3600000;

  let score = 0;
  if (importance && importance > 0) score += importance * 2; // 1..5 -> 2..10
  if (typeof priorityScore === 'number') {
    if (priorityScore >= 80) score += 8; else if (priorityScore >= 60) score += 6; else if (priorityScore >= 40) score += 4; else if (priorityScore >= 20) score += 2; else score += 1;
  }
  if (isFinite(hoursLeft)) {
    if (hoursLeft <= 24) score += 10; else if (hoursLeft <= 72) score += 6; else if (hoursLeft <= 168) score += 3; else if (hoursLeft <= 336) score += 1; // two weeks
  }
  // Past due slight bump to highlight stale urgent things
  if (hoursLeft < 0) score += 4;

  let level: SeverityLevel = 'low';
  if (score >= 20) level = 'critical';
  else if (score >= 14) level = 'high';
  else if (score >= 8) level = 'medium';

  const variantMap: Record<SeverityLevel,string> = {
    critical: 'danger',
    high: 'warning',
    medium: 'primary',
    low: 'secondary'
  };

  return { level, score, variant: variantMap[level], label: level };
}

interface Props extends SeverityInput { className?: string; showLabel?: boolean; }

const SeverityBadge: React.FC<Props> = ({ className = '', showLabel = true, ...rest }) => {
  const sev = computeSeverity(rest);
  return <span className={`badge bg-${sev.variant} ${className}`} title={`severity score: ${sev.score}`}>{showLabel ? sev.label : ''}</span>;
};

export default SeverityBadge;
