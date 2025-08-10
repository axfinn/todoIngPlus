import { configureStore } from '@reduxjs/toolkit';
import { unifiedApi } from '../features/unified/unifiedApi';
import authReducer from '../features/auth/authSlice';
import taskReducer from '../features/tasks/taskSlice';
import reportReducer from '../features/reports/reportSlice';
import reminderReducer from '../features/reminders/reminderSlice';
import eventsReducer from '../features/events/eventSlice';
import dashboardReducer from '../features/dashboard/dashboardSlice';
import unifiedReducer from '../features/unified/unifiedSlice';
import notificationReducer from '../features/notifications/notificationSlice';

export const store = configureStore({
  reducer: {
    auth: authReducer,
    tasks: taskReducer,
    reports: reportReducer,
  reminders: reminderReducer,
  events: eventsReducer,
  dashboard: dashboardReducer,
  unified: unifiedReducer,
  notifications: notificationReducer,
  [unifiedApi.reducerPath]: unifiedApi.reducer,
  },
  middleware: (getDefault) => getDefault().concat(unifiedApi.middleware),
});

export type RootState = ReturnType<typeof store.getState>;
export type AppDispatch = typeof store.dispatch;