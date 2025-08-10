import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';
import api from '../../config/api';
import type { CalendarEvent } from '../events/eventSlice';
import type { Reminder } from '../reminders/reminderSlice';
import type { Task } from '../tasks/taskSlice';

// Dashboard 相关类型定义
export interface PriorityTask extends Task {
  priority_score: number;
  days_left: number;
  is_urgent: boolean;
}

export interface TaskSortConfig {
  user_id: string;
  priority_days: number;
  max_display_count: number;
  weight_urgent: number;
  weight_important: number;
  created_at: string;
  updated_at: string;
}

export interface UpdateTaskSortConfigRequest {
  priority_days?: number;
  max_display_count?: number;
  weight_urgent?: number;
  weight_important?: number;
}

export interface DashboardData {
  upcoming_events: CalendarEvent[];
  priority_tasks: PriorityTask[];
  pending_reminders: Reminder[];
  stats: {
    total_events: number;
    total_tasks: number;
    total_reminders: number;
    completed_tasks: number;
    upcoming_events_count: number;
    pending_reminders_count: number;
  };
}

// Dashboard state
interface DashboardState {
  dashboardData: DashboardData | null;
  priorityTasks: PriorityTask[];
  taskSortConfig: TaskSortConfig | null;
  isLoading: boolean;
  error: string | null;
}

const initialState: DashboardState = {
  dashboardData: null,
  priorityTasks: [],
  taskSortConfig: null,
  isLoading: false,
  error: null,
};

// Async thunks
export const fetchDashboardData = createAsyncThunk<DashboardData, void, { rejectValue: string }>(
  'dashboard/fetchDashboardData',
  async (_, { rejectWithValue }) => {
    try {
      const res = await api.get('/dashboard');
      return res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch dashboard data');
    }
  }
);

export const fetchPriorityTasks = createAsyncThunk<PriorityTask[], number | void, { rejectValue: string }>(
  'dashboard/fetchPriorityTasks',
  async (limit, { rejectWithValue }) => {
    try {
      const limitParam = limit || 20;
      const res = await api.get(`/dashboard/tasks?limit=${limitParam}`);
      return res.data.tasks || res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch priority tasks');
    }
  }
);

export const fetchTaskSortConfig = createAsyncThunk<TaskSortConfig, void, { rejectValue: string }>(
  'dashboard/fetchTaskSortConfig',
  async (_, { rejectWithValue }) => {
    try {
      const res = await api.get('/dashboard/config');
      return res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to fetch task sort config');
    }
  }
);

export const updateTaskSortConfig = createAsyncThunk<TaskSortConfig, UpdateTaskSortConfigRequest, { rejectValue: string }>(
  'dashboard/updateTaskSortConfig',
  async (configData, { rejectWithValue }) => {
    try {
      const res = await api.put('/dashboard/config', configData);
      return res.data;
    } catch (err: any) {
      return rejectWithValue(err.response?.data?.message || 'Failed to update task sort config');
    }
  }
);

// Dashboard slice
const dashboardSlice = createSlice({
  name: 'dashboard',
  initialState,
  reducers: {
    clearError: (state) => {
      state.error = null;
    },
    clearDashboardData: (state) => {
      state.dashboardData = null;
      state.priorityTasks = [];
      state.taskSortConfig = null;
    },
    updateTaskInDashboard: (state, action: PayloadAction<Task>) => {
      const updatedTask = action.payload;
      if (state.dashboardData) {
        // Update task in priority tasks if it exists
        const taskIndex = state.dashboardData.priority_tasks.findIndex(task => task._id === updatedTask._id);
        if (taskIndex !== -1) {
          state.dashboardData.priority_tasks[taskIndex] = {
            ...state.dashboardData.priority_tasks[taskIndex],
            ...updatedTask,
          };
        }
      }
      
      // Update in priority tasks array
      const priorityIndex = state.priorityTasks.findIndex(task => task._id === updatedTask._id);
      if (priorityIndex !== -1) {
        state.priorityTasks[priorityIndex] = {
          ...state.priorityTasks[priorityIndex],
          ...updatedTask,
        };
      }
    },
    removeTaskFromDashboard: (state, action: PayloadAction<string>) => {
      const taskId = action.payload;
      if (state.dashboardData) {
        state.dashboardData.priority_tasks = state.dashboardData.priority_tasks.filter(task => task._id !== taskId);
      }
      state.priorityTasks = state.priorityTasks.filter(task => task._id !== taskId);
    },
  },
  extraReducers: (builder) => {
    // Fetch dashboard data
    builder
      .addCase(fetchDashboardData.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(fetchDashboardData.fulfilled, (state, action) => {
        state.isLoading = false;
        state.dashboardData = action.payload;
      })
      .addCase(fetchDashboardData.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to fetch dashboard data';
      });

    // Fetch priority tasks
    builder
      .addCase(fetchPriorityTasks.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(fetchPriorityTasks.fulfilled, (state, action) => {
        state.isLoading = false;
        state.priorityTasks = action.payload;
      })
      .addCase(fetchPriorityTasks.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to fetch priority tasks';
      });

    // Fetch task sort config
    builder
      .addCase(fetchTaskSortConfig.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(fetchTaskSortConfig.fulfilled, (state, action) => {
        state.isLoading = false;
        state.taskSortConfig = action.payload;
      })
      .addCase(fetchTaskSortConfig.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to fetch task sort config';
      });

    // Update task sort config
    builder
      .addCase(updateTaskSortConfig.pending, (state) => {
        state.isLoading = true;
        state.error = null;
      })
      .addCase(updateTaskSortConfig.fulfilled, (state, action) => {
        state.isLoading = false;
        state.taskSortConfig = action.payload;
      })
      .addCase(updateTaskSortConfig.rejected, (state, action) => {
        state.isLoading = false;
        state.error = action.payload || 'Failed to update task sort config';
      });
  },
});

export const { 
  clearError, 
  clearDashboardData, 
  updateTaskInDashboard, 
  removeTaskFromDashboard 
} = dashboardSlice.actions;

export default dashboardSlice.reducer;
