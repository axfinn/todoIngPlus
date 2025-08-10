import { render } from '@testing-library/react';
import { describe, it, expect } from 'vitest';
import SeverityBadge, { computeSeverity } from '../../components/SeverityBadge';

describe('computeSeverity', () => {
  it('assigns higher severity for imminent high-importance items', () => {
    const now = new Date();
    const soon = new Date(now.getTime() + 3 * 3600 * 1000); // 3h
    const res = computeSeverity({ source: 'event', scheduledAt: soon, importance: 5 }, now);
    expect(res.level === 'critical' || res.level === 'high').toBeTruthy();
  });

  it('assigns low severity for distant low importance', () => {
    const now = new Date();
    const distant = new Date(now.getTime() + 30 * 24 * 3600 * 1000);
    const res = computeSeverity({ source: 'task', scheduledAt: distant, importance: 1, priorityScore: 5 }, now);
    expect(res.level).toBe('low');
  });
});

describe('SeverityBadge component', () => {
  it('renders without crashing', () => {
    const { getByTitle } = render(<SeverityBadge source="task" scheduledAt={new Date()} priorityScore={50} />);
    expect(getByTitle(/severity score/i)).toBeInTheDocument();
  });
});
