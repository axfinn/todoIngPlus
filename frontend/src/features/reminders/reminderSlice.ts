import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';
import api from '../../config/api';

// 该 slice 原始结构与后端旧实现不一致。已重构匹配 backend-go/internal/models/reminder.go 中返回的字段。

// Reminder 类型 (后端返回 ReminderWithEvent 时会带嵌套 event，可在页面层处理，这里聚焦主字段)
export interface Reminder {
  id: string;                 // 后端 json:"id"
  event_id: string;           // 事件 ObjectID
  user_id: string;
  advance_days: number;
  reminder_times: string[];   // 多个 HH:MM
  reminder_type: 'app' | 'email' | 'both';
  custom_message?: string;
  is_active: boolean;
  last_sent?: string;
  next_send?: string;
  created_at: string;
  updated_at: string;
}

// 创建请求
export interface CreateReminderRequest {
  event_id: string;
  advance_days: number;
  reminder_times: string[];
  reminder_type: 'app' | 'email' | 'both';
  custom_message?: string;
}

// 更新请求 (全部可选)
export interface UpdateReminderRequest extends Partial<CreateReminderRequest> {}

// Reminder state
interface ReminderState {
  reminders: Reminder[];
  isLoading: boolean;
  error: string | null;
  selectedReminder: Reminder | null;
}

const initialState: ReminderState = {
  reminders: [],
  isLoading: false,
  error: null,
  selectedReminder: null,
};

// Async thunks
export const fetchReminders = createAsyncThunk<Reminder[], { page?: number; limit?: number } | void, { rejectValue: string }>(
  'reminders/fetchReminders',
  async (params, { rejectWithValue }) => {
    try {
      const { page = 1, limit = 50 } = params || {};
      const res = await api.get(`/reminders?page=${page}&limit=${limit}`);
      const list = res.data.reminders || res.data || [];
      return list.map((r: any) => ({
        id: r.id || r._id,
        event_id: r.event_id || r.eventId,
        user_id: r.user_id,
        advance_days: r.advance_days ?? 0,
        reminder_times: Array.isArray(r.reminder_times) ? r.reminder_times : (Array.isArray(r.reminderTimes) ? r.reminderTimes : []),
        reminder_type: r.reminder_type || r.reminderType,
        custom_message: r.custom_message || r.customMessage,
        is_active: r.is_active,
        last_sent: r.last_sent,
        next_send: r.next_send,
        created_at: r.created_at,
        updated_at: r.updated_at,
      }));
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch reminders');
    }
  }
);


export const createReminder = createAsyncThunk<Reminder, CreateReminderRequest, { rejectValue: string }>(
  'reminders/createReminder',
  async (data, { rejectWithValue }) => {
    try {
      const res = await api.post('/reminders', data);
      const r = res.data.reminder || res.data;
      return {
        id: r.id || r._id,
        event_id: r.event_id || r.eventId,
        user_id: r.user_id,
        advance_days: r.advance_days ?? 0,
        reminder_times: Array.isArray(r.reminder_times) ? r.reminder_times : (Array.isArray(r.reminderTimes) ? r.reminderTimes : []),
        reminder_type: r.reminder_type || r.reminderType,
        custom_message: r.custom_message || r.customMessage,
        is_active: r.is_active,
        last_sent: r.last_sent,
        next_send: r.next_send,
        created_at: r.created_at,
        updated_at: r.updated_at,
      } as Reminder;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to create reminder');
    }
  }
);

export const updateReminder = createAsyncThunk<Reminder, { id: string; reminderData: UpdateReminderRequest }, { rejectValue: string }>(
  'reminders/updateReminder',
  async ({ id, reminderData }, { rejectWithValue }) => {
    try {
      const res = await api.put(`/reminders/${id}`, reminderData);
      const r = res.data.reminder || res.data;
      return {
        id: r.id || r._id,
        event_id: r.event_id || r.eventId,
        user_id: r.user_id,
        advance_days: r.advance_days ?? 0,
        reminder_times: Array.isArray(r.reminder_times) ? r.reminder_times : (Array.isArray(r.reminderTimes) ? r.reminderTimes : []),
        reminder_type: r.reminder_type || r.reminderType,
        custom_message: r.custom_message || r.customMessage,
        is_active: r.is_active,
        last_sent: r.last_sent,
        next_send: r.next_send,
        created_at: r.created_at,
        updated_at: r.updated_at,
      } as Reminder;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to update reminder');
    }
  }
);

export const deleteReminder = createAsyncThunk<string, string, { rejectValue: string }>(
  'reminders/deleteReminder',
  async (id, { rejectWithValue }) => {
    try {
      await api.delete(`/reminders/${id}`);
      return id;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to delete reminder');
    }
  }
);

export const snoozeReminder = createAsyncThunk<Reminder | { message: string; snooze_minutes: number }, { id: string; minutes: number }, { rejectValue: string }>(
  'reminders/snoozeReminder',
  async ({ id, minutes }, { rejectWithValue }) => {
    try {
      // 后端期望字段 snooze_minutes
      const res = await api.post(`/reminders/${id}/snooze`, { snooze_minutes: minutes });
      // 当前后端返回 {message, snooze_minutes} 而不是完整 Reminder，直接透传
      return res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to snooze reminder');
    }
  }
);

// Reminder slice
const reminderSlice = createSlice({
  name: 'reminders',
  initialState,
  reducers: {
    setSelectedReminder: (state, action: PayloadAction<Reminder | null>) => {
      state.selectedReminder = action.payload;
    },
    clearError: (state) => {
      state.error = null;
    },
    clearReminders: (state) => {
      state.reminders = [];
      state.selectedReminder = null;
    },
    markReminderAsSent: (state, action: PayloadAction<string>) => {
      // 新模型暂未包含 is_sent/sent_at 字段，保留占位逻辑（未来如果需要可扩展）
      const reminder = state.reminders.find(r => r.id === action.payload);
      if (reminder) {
        // no-op for now
      }
    },
  },
  extraReducers: (builder) => {
    // Fetch reminders
    builder
      .addCase(fetchReminders.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(fetchReminders.fulfilled, (state, action) => {
        state.isLoading = false;
        state.reminders = action.payload;
      })
      .addCase(fetchReminders.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to fetch reminders';
      });


    // Create reminder
    builder
      .addCase(createReminder.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(createReminder.fulfilled, (state, action) => {
        state.isLoading = false;
        state.reminders.unshift(action.payload);
      })
      .addCase(createReminder.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to create reminder';
      });

    // Update reminder
    builder
      .addCase(updateReminder.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(updateReminder.fulfilled, (state, action) => {
        state.isLoading = false;
        const updated = action.payload;
        const idx = state.reminders.findIndex(r => r.id === updated.id);
        if (idx !== -1) state.reminders[idx] = updated;
        if (state.selectedReminder?.id === updated.id) state.selectedReminder = updated;
      })
      .addCase(updateReminder.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to update reminder';
      });

    // Delete reminder
    builder
      .addCase(deleteReminder.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(deleteReminder.fulfilled, (state, action) => {
        state.isLoading = false;
        state.reminders = state.reminders.filter(r => r.id !== action.payload);
        if (state.selectedReminder?.id === action.payload) state.selectedReminder = null;
      })
      .addCase(deleteReminder.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to delete reminder';
      });

    // Snooze reminder
    builder
      .addCase(snoozeReminder.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(snoozeReminder.fulfilled, (state) => {
        state.isLoading = false; // 仅状态复位
      })
      .addCase(snoozeReminder.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to snooze reminder';
      });
  },
});

export const { setSelectedReminder, clearError, clearReminders, markReminderAsSent } = reminderSlice.actions;
export default reminderSlice.reducer;
