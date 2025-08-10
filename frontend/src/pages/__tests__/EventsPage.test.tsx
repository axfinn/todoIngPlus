/// <reference types="vitest" />
import { describe, test, expect, vi } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import EventsPage from '../EventsPage';
import { NowContext } from '../../App';
import { MemoryRouter } from 'react-router-dom';

// Mock i18n
vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (k: string, def?: string) => def || k })
}));
vi.mock('../../features/unified/unifiedApi', () => ({
  useGetUpcomingQuery: () => ({ data: undefined })
}));
vi.mock('../../config/api', () => ({
  default: {
    get: () => Promise.resolve({ data: { events: [] } }),
    post: () => Promise.resolve({ data: {} }),
    delete: () => Promise.resolve({ data: {} })
  }
}));

describe('EventsPage create modal', () => {
  test('open create modal on button click', async () => {
    render(
      <MemoryRouter>
        <NowContext.Provider value={Date.now()}>
          <EventsPage />
        </NowContext.Provider>
      </MemoryRouter>
    );

    const createBtn = await screen.findByRole('button', { name: /create/i });
    fireEvent.click(createBtn);

    await waitFor(() => {
      const el = document.getElementById('event_date');
      expect(el).not.toBeNull();
    });
  });
});
