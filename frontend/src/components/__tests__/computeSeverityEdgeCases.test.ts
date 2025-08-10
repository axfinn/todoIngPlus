import { describe, it, expect } from 'vitest';
import { computeSeverity } from '../SeverityBadge';

describe('computeSeverity edge cases', () => {
  const now = new Date();

  it('returns low severity when no data provided', () => {
    const res = computeSeverity({ source: 'task' }, now);
    expect(res.level).toBe('low');
    expect(res.score).toBe(0);
  });

  it('applies importance weighting', () => {
    const soon = new Date(now.getTime() + 5 * 3600 * 1000); // 5h
    const a = computeSeverity({ source: 'event', scheduledAt: soon, importance: 1 }, now).score;
    const b = computeSeverity({ source: 'event', scheduledAt: soon, importance: 5 }, now).score;
    expect(b).toBeGreaterThan(a);
  });

  it('applies priorityScore bucket increases', () => {
    const sched = new Date(now.getTime() + 100 * 3600 * 1000); // >72h
    const low = computeSeverity({ source: 'task', scheduledAt: sched, priorityScore: 10 }, now).score;
    const mid = computeSeverity({ source: 'task', scheduledAt: sched, priorityScore: 45 }, now).score;
    const high = computeSeverity({ source: 'task', scheduledAt: sched, priorityScore: 85 }, now).score;
    expect(mid).toBeGreaterThan(low);
    expect(high).toBeGreaterThan(mid);
  });

  it('adds strong urgency for within 24h', () => {
    const soon = new Date(now.getTime() + 2 * 3600 * 1000);
    const far = new Date(now.getTime() + 10 * 24 * 3600 * 1000);
    const nearScore = computeSeverity({ source: 'event', scheduledAt: soon }, now).score;
    const farScore = computeSeverity({ source: 'event', scheduledAt: far }, now).score;
    expect(nearScore).toBeGreaterThan(farScore);
  });

  it('bumps past-due items', () => {
    const past = new Date(now.getTime() - 2 * 3600 * 1000);
    const pastRes = computeSeverity({ source: 'task', scheduledAt: past, importance: 5 }, now);
    expect(pastRes.level).toBe('critical'); // importance (10) + within 24h (10) + past due (4) = 24
  });
});
