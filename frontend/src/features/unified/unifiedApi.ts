import { createApi, fetchBaseQuery } from '@reduxjs/toolkit/query/react';

export interface UnifiedUpcomingItem {
  id: string;
  source: 'task' | 'event' | 'reminder';
  title: string;
  scheduled_at: string;
  countdown_seconds: number;
  days_left: number;
  importance?: number;
  priority_score?: number;
  detail_url?: string;
  source_id?: string;
}

export interface UnifiedUpcomingResponse {
  items: UnifiedUpcomingItem[];
  hours: number;
  total: number;
  server_timestamp?: number;
  stats?: { tasks: number; events: number; reminders: number };
}

export interface UpcomingQueryArgs { hours?: number; sources?: string[]; limit?: number }

export const unifiedApi = createApi({
  reducerPath: 'unifiedApi',
  baseQuery: fetchBaseQuery({
    baseUrl: '/api',
    prepareHeaders: (headers) => {
      const token = localStorage.getItem('token');
      if (token) headers.set('Authorization', `Bearer ${token}`);
      return headers;
    }
  }),
  tagTypes: ['UnifiedUpcoming'],
  refetchOnFocus: true,
  refetchOnReconnect: true,
  keepUnusedDataFor: 30, // 秒：与计划 stale 30s 策略一致
  endpoints: (builder) => ({
    getUpcoming: builder.query<UnifiedUpcomingResponse, UpcomingQueryArgs | void>({
      query: (args) => {
        const h = args?.hours ?? 24*7;
        const params: string[] = [`hours=${h}`];
        if (args?.sources?.length) params.push(`sources=${encodeURIComponent(args.sources.join(','))}`);
        if (args?.limit) params.push(`limit=${args.limit}`);
  // 追加 debug=1 可选，用于后端未来返回更多诊断(目前忽略)
  params.push('debug=1');
  return `unified/upcoming?${params.join('&')}`;
      },
      providesTags: (result) => result ? [{ type: 'UnifiedUpcoming', id: 'LIST' }] : [],
    }),
  }),
});

export const { useGetUpcomingQuery } = unifiedApi;
