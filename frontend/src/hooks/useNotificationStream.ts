import { useEffect, useRef } from 'react';
import { useDispatch } from 'react-redux';
import { notificationReceived, connectionChanged, setError } from '../features/notifications/notificationSlice';

export default function useNotificationStream(enabled: boolean) {
  const dispatch = useDispatch();
  const ref = useRef<EventSource | null>(null);

  useEffect(()=> {
    if (!enabled) return;
    const token = localStorage.getItem('token');
    if (!token) return;
    const es = new EventSource(`/api/notifications/stream?token=${encodeURIComponent(token)}`);
    ref.current = es;
    es.onopen = () => dispatch(connectionChanged(true));
    es.onerror = () => { dispatch(connectionChanged(false)); dispatch(setError('stream_error')); };
    es.addEventListener('notification', (ev: MessageEvent) => {
      try { const data = JSON.parse(ev.data); dispatch(notificationReceived(data)); } catch {}
    });
    return () => { es.close(); dispatch(connectionChanged(false)); };
  }, [enabled, dispatch]);
}
