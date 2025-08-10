import { createSlice, createAsyncThunk, type PayloadAction } from '@reduxjs/toolkit';
import api from '../../config/api';

// 说明: 该 slice 原先使用 start_time/end_time 等旧字段, 与后端 Go 新版事件模型不兼容。
// 已重构为匹配 backend-go/internal/models/event.go 中的 Event / CreateEventRequest。

// Event type (与后端字段保持一致)
export interface CalendarEvent {
	id: string;               // 后端 json:"id"
	user_id: string;
	title: string;
	description: string;
	event_type: string;       // birthday / anniversary / holiday / custom / meeting / deadline
	event_date: string;       // ISO 时间
	recurrence_type: string;  // none / yearly / monthly / weekly / daily
	recurrence_config?: Record<string, any>;
	importance_level: number;
	tags?: string[];
	location?: string;
	is_all_day: boolean;
	created_at: string;
	updated_at: string;
	is_active?: boolean;
}

// CreateEventRequest 与后端期望字段匹配
export interface CreateEventRequest {
	title: string;
	description?: string;
	event_type: string;
	event_date: string;          // 前端发送原始字符串(支持 datetime-local) 由后端多格式解析
	recurrence_type: string;     // none / yearly / monthly / weekly / daily
	recurrence_config?: Record<string, any>;
	importance_level: number;
	tags?: string[];
	location?: string;
	is_all_day: boolean;
	raw_event_date?: string;     // 额外传递，后端备用
}

export interface UpdateEventRequest extends Partial<CreateEventRequest> {}

interface EventState {
	events: CalendarEvent[];
	isLoading: boolean;
	error: string | null;
	selectedEvent: CalendarEvent | null;
}

const initialState: EventState = {
	events: [],
	isLoading: false,
	error: null,
	selectedEvent: null,
};

export const fetchEvents = createAsyncThunk<CalendarEvent[], void, { rejectValue: string }>(
	'events/fetchAll',
	async (_, { rejectWithValue }) => {
		try {
			const res = await api.get('/events');
			const list = res.data.events || res.data || [];
			// 规范化字段: 旧 _id -> id
			return list.map((e: any) => ({
				id: e.id || e._id,
				user_id: e.user_id,
				title: e.title,
				description: e.description,
				event_type: e.event_type,
				event_date: e.event_date,
				recurrence_type: e.recurrence_type,
				recurrence_config: e.recurrence_config,
				importance_level: e.importance_level,
				tags: e.tags,
				location: e.location,
				is_all_day: e.is_all_day,
				created_at: e.created_at,
				updated_at: e.updated_at,
				is_active: e.is_active,
			}));
		} catch (err: any) {
			return rejectWithValue(err.response?.data?.message || 'Failed to fetch events');
		}
	}
);


export const createEvent = createAsyncThunk<CalendarEvent, CreateEventRequest, { rejectValue: string }>(
	'events/create',
	async (data, { rejectWithValue }) => {
		try {
			const payload = { ...data, raw_event_date: data.raw_event_date || data.event_date };
			const res = await api.post('/events', payload);
			const e = res.data;
			return {
				id: e.id || e._id,
				user_id: e.user_id,
				title: e.title,
				description: e.description,
				event_type: e.event_type,
				event_date: e.event_date,
				recurrence_type: e.recurrence_type,
				recurrence_config: e.recurrence_config,
				importance_level: e.importance_level,
				tags: e.tags,
				location: e.location,
				is_all_day: e.is_all_day,
				created_at: e.created_at,
				updated_at: e.updated_at,
				is_active: e.is_active,
			} as CalendarEvent;
		} catch (err: any) {
			return rejectWithValue(err.response?.data?.message || 'Failed to create event');
		}
	}
);

export const updateEvent = createAsyncThunk<CalendarEvent, { id: string; data: UpdateEventRequest }, { rejectValue: string }>(
	'events/update',
	async ({ id, data }, { rejectWithValue }) => {
		try {
			const res = await api.put(`/events/${id}`, data);
			const e = res.data;
			return {
				id: e.id || e._id,
				user_id: e.user_id,
				title: e.title,
				description: e.description,
				event_type: e.event_type,
				event_date: e.event_date,
				recurrence_type: e.recurrence_type,
				recurrence_config: e.recurrence_config,
				importance_level: e.importance_level,
				tags: e.tags,
				location: e.location,
				is_all_day: e.is_all_day,
				created_at: e.created_at,
				updated_at: e.updated_at,
				is_active: e.is_active,
			} as CalendarEvent;
		} catch (err: any) {
			return rejectWithValue(err.response?.data?.message || 'Failed to update event');
		}
	}
);

export const deleteEvent = createAsyncThunk<string, string, { rejectValue: string }>(
	'events/delete',
	async (id, { rejectWithValue }) => {
		try {
			await api.delete(`/events/${id}`);
			return id;
		} catch (err: any) {
			return rejectWithValue(err.response?.data?.message || 'Failed to delete event');
		}
	}
);

const eventsSlice = createSlice({
	name: 'events',
	initialState,
	reducers: {
		setSelectedEvent: (state, action: PayloadAction<CalendarEvent | null>) => {
			state.selectedEvent = action.payload;
		},
		clearEvents: (state) => {
			state.events = [];
			state.selectedEvent = null;
		},
		clearError: (state) => { state.error = null; },
	},
	extraReducers: (builder) => {
		builder
			.addCase(fetchEvents.pending, (state) => { state.isLoading = true; state.error = null; })
			.addCase(fetchEvents.fulfilled, (state, action) => { state.isLoading = false; state.events = action.payload; })
			.addCase(fetchEvents.rejected, (state, action) => { state.isLoading = false; state.error = action.payload || 'Failed to fetch events'; })
			.addCase(createEvent.fulfilled, (state, action) => { state.events.unshift(action.payload); })
			.addCase(updateEvent.fulfilled, (state, action) => {
				const idx = state.events.findIndex(e => e.id === action.payload.id);
				if (idx !== -1) state.events[idx] = action.payload;
				if (state.selectedEvent?.id === action.payload.id) state.selectedEvent = action.payload;
			})
			.addCase(deleteEvent.fulfilled, (state, action) => { state.events = state.events.filter(e => e.id !== action.payload); });
	}
});

export const { setSelectedEvent, clearEvents, clearError } = eventsSlice.actions;
export default eventsSlice.reducer;
