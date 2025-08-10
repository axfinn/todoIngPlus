import { createSlice, createAsyncThunk } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';
import api from '../../config/api';

export interface NotificationItem {
  id: string;
  type: string;
  message: string;
  event_id?: string;
  created_at: string;
  read_at?: string;
  metadata?: Record<string, any>;
}

interface NotificationState {
  items: NotificationItem[];
  unread: number;
  connected: boolean;
  error: string | null;
}

const initialState: NotificationState = { items: [], unread: 0, connected: false, error: null };

export const fetchNotifications = createAsyncThunk<NotificationItem[]>('notifications/fetch', async () => {
  const res = await api.get('/notifications?limit=50');
  const list: NotificationItem[] = res.data.notifications || [];
  return list;
});

const slice = createSlice({
  name: 'notifications',
  initialState,
  reducers: {
    notificationReceived(state, action: PayloadAction<NotificationItem>) {
      const exists = state.items.find(i => i.id === action.payload.id);
      if (!exists) {
        state.items.unshift(action.payload);
      }
      state.unread = state.items.filter(i => !i.read_at).length;
    },
    markReadLocal(state, action: PayloadAction<string>) {
      const n = state.items.find(i => i.id === action.payload); if (n && !n.read_at) { n.read_at = new Date().toISOString(); }
      state.unread = state.items.filter(i => !i.read_at).length;
    },
    connectionChanged(state, action: PayloadAction<boolean>) { state.connected = action.payload; },
    setError(state, action: PayloadAction<string|null>) { state.error = action.payload; }
  },
  extraReducers: builder => {
    builder.addCase(fetchNotifications.fulfilled, (state, action)=> {
      state.items = action.payload.sort((a,b)=> b.created_at.localeCompare(a.created_at));
      state.unread = state.items.filter(i=> !i.read_at).length;
    });
  }
});

export const { notificationReceived, markReadLocal, connectionChanged, setError } = slice.actions;
export default slice.reducer;
