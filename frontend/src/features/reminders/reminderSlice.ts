import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';
import api from '../../config/api';

// Reminder 类型定义
export interface Reminder {
  _id: string;
  event_id: string;
  message: string;
  remind_at: string;
  type: 'email' | 'app';
  is_sent: boolean;
  sent_at?: string;
  user_id: string;
  created_at: string;
  updated_at: string;
}

// Reminder creation request
export interface CreateReminderRequest {
  event_id: string;
  message: string;
  remind_at: string;
  type: 'email' | 'app';
}

// Reminder update request
export interface UpdateReminderRequest extends Partial<CreateReminderRequest> {}

// Reminder state
interface ReminderState {
  reminders: Reminder[];
  upcomingReminders: Reminder[];
  isLoading: boolean;
  error: string | null;
  selectedReminder: Reminder | null;
}

const initialState: ReminderState = {
  reminders: [],
  upcomingReminders: [],
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
      return res.data.reminders || res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch reminders');
    }
  }
);

export const fetchUpcomingReminders = createAsyncThunk<Reminder[], number | void, { rejectValue: string }>(
  'reminders/fetchUpcomingReminders',
  async (hours, { rejectWithValue }) => {
    try {
      const hoursParam = hours || 24;
      const res = await api.get(`/reminders/upcoming?hours=${hoursParam}`);
      return res.data.reminders || res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch upcoming reminders');
    }
  }
);

export const createReminder = createAsyncThunk<Reminder, CreateReminderRequest, { rejectValue: string }>(
  'reminders/createReminder',
  async (reminderData, { rejectWithValue }) => {
    try {
      const res = await api.post('/reminders', reminderData);
      return res.data.reminder || res.data;
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
      return res.data.reminder || res.data;
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

export const snoozeReminder = createAsyncThunk<Reminder, { id: string; minutes: number }, { rejectValue: string }>(
  'reminders/snoozeReminder',
  async ({ id, minutes }, { rejectWithValue }) => {
    try {
      const res = await api.post(`/reminders/${id}/snooze`, { minutes });
      return res.data.reminder || res.data;
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
      state.upcomingReminders = [];
      state.selectedReminder = null;
    },
    markReminderAsSent: (state, action: PayloadAction<string>) => {
      const reminder = state.reminders.find(r => r._id === action.payload);
      if (reminder) {
        reminder.is_sent = true;
        reminder.sent_at = new Date().toISOString();
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

    // Fetch upcoming reminders
    builder
      .addCase(fetchUpcomingReminders.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(fetchUpcomingReminders.fulfilled, (state, action) => {
        state.isLoading = false;
        state.upcomingReminders = action.payload;
      })
      .addCase(fetchUpcomingReminders.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to fetch upcoming reminders';
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
        const updatedReminder = action.payload;
        const index = state.reminders.findIndex(reminder => reminder._id === updatedReminder._id);
        if (index !== -1) {
          state.reminders[index] = updatedReminder;
        }
        if (state.selectedReminder?._id === updatedReminder._id) {
          state.selectedReminder = updatedReminder;
        }
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
        state.reminders = state.reminders.filter(reminder => reminder._id !== action.payload);
        if (state.selectedReminder?._id === action.payload) {
          state.selectedReminder = null;
        }
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
      .addCase(snoozeReminder.fulfilled, (state, action) => {
        state.isLoading = false;
        const updatedReminder = action.payload;
        const index = state.reminders.findIndex(reminder => reminder._id === updatedReminder._id);
        if (index !== -1) {
          state.reminders[index] = updatedReminder;
        }
        // Update upcoming reminders list
        const upcomingIndex = state.upcomingReminders.findIndex(reminder => reminder._id === updatedReminder._id);
        if (upcomingIndex !== -1) {
          state.upcomingReminders[upcomingIndex] = updatedReminder;
        }
      })
      .addCase(snoozeReminder.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to snooze reminder';
      });
  },
});

export const { setSelectedReminder, clearError, clearReminders, markReminderAsSent } = reminderSlice.actions;
export default reminderSlice.reducer;
