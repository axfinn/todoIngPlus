/// <reference types="vitest" />
import { describe, test, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import RemindersPage from '../RemindersPage';
import { MemoryRouter } from 'react-router-dom';

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, def?: string) => def || k })
}));
vi.mock('../../features/unified/unifiedApi', () => ({
  useGetUpcomingQuery: () => ({ data: undefined })
}));
vi.mock('../../config/api', () => ({
  default: {
    get: (url: string) => {
      if (url.startsWith('/reminders/simple')) return Promise.resolve({ data: { reminders: [] } });
      if (url.startsWith('/events/options')) return Promise.resolve({ data: { data: [] } });
      return Promise.resolve({ data: {} });
    },
    post: () => Promise.resolve({ data: {} })
  }
}));

describe('RemindersPage create modal', () => {
  test('open create reminder modal on button click', async () => {
    render(
      <MemoryRouter>
        <RemindersPage />
      </MemoryRouter>
    );

    const btn = await screen.findByRole('button', { name: /create/i });
    fireEvent.click(btn);

    await waitFor(() => {
      const el = document.getElementById('event_id');
      expect(el).not.toBeNull();
    });
  });
});
