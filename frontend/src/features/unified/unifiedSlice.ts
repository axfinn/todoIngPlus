import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';
import api from '../../config/api';

export interface UnifiedItem {
  id: string;
  source: 'task' | 'event' | 'reminder';
  sub_type?: string;
  title: string;
  scheduled_at: string;
  countdown_seconds: number;
  days_left: number;
  importance?: number;
  priority_score?: number;
  related_event_id?: string;
}

interface UnifiedState {
  items: UnifiedItem[];
  hours: number;
  isLoading: boolean;
  error: string | null;
  serverTimestamp?: number;
}

const initialState: UnifiedState = {
  items: [],
  hours: 24 * 7,
  isLoading: false,
  error: null,
};

interface FetchUnifiedArgs { hours?: number; sources?: string[]; limit?: number }
export const fetchUnifiedUpcoming = createAsyncThunk<{ items: UnifiedItem[]; hours: number; serverTimestamp?: number }, FetchUnifiedArgs | void, { rejectValue: string }>(
  'unified/fetchUpcoming',
  async (args, { rejectWithValue }) => {
    try {
      const h = args?.hours || 24 * 7;
      const params: string[] = [`hours=${h}`];
      if (args?.sources && args.sources.length) { params.push(`sources=${encodeURIComponent(args.sources.join(','))}`); }
      if (args?.limit) { params.push(`limit=${args.limit}`); }
  const res = await api.get(`/unified/upcoming?${params.join('&')}`);
  return { items: res.data.items || [], hours: h, serverTimestamp: res.data.server_timestamp };
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch upcoming unified items');
    }
  }
);

const unifiedSlice = createSlice({
  name: 'unified',
  initialState,
  reducers: {
    clearUnified: (state) => { state.items = []; state.error = null; },
  },
  extraReducers: (builder) => {
    builder
      .addCase(fetchUnifiedUpcoming.pending, (state) => { state.isLoading = true; state.error = null; })
      .addCase(fetchUnifiedUpcoming.fulfilled, (state, action: PayloadAction<{ items: UnifiedItem[]; hours: number; serverTimestamp?: number }>) => {
        state.isLoading = false;
        state.items = action.payload.items;
        state.hours = action.payload.hours;
        state.serverTimestamp = action.payload.serverTimestamp;
      })
      .addCase(fetchUnifiedUpcoming.rejected, (state, action) => { state.isLoading = false; state.error = action.payload || 'Failed'; });
  }
});

export const { clearUnified } = unifiedSlice.actions;
export default unifiedSlice.reducer;
