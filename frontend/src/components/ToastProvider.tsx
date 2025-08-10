import React, { createContext, useCallback, useContext, useState } from 'react';

export interface ToastMessage { id: string; type?: 'success'|'error'|'info'|'warning'; title?: string; message: string; duration?: number }
interface ToastContextValue { push: (msg: Omit<ToastMessage,'id'>) => void }
const ToastContext = createContext<ToastContextValue | null>(null);

export const useToast = () => {
  const ctx = useContext(ToastContext);
  if(!ctx) throw new Error('useToast must be used within ToastProvider');
  return ctx;
};

const ToastProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [toasts, setToasts] = useState<ToastMessage[]>([]);

  const push = useCallback((msg: Omit<ToastMessage,'id'>) => {
    const id = Math.random().toString(36).slice(2);
    const t: ToastMessage = { id, type: 'info', duration: 4000, ...msg };
    setToasts(prev => [...prev, t]);
    if(t.duration) {
      setTimeout(() => setToasts(prev => prev.filter(x => x.id !== id)), t.duration);
    }
  }, []);

  const remove = (id: string) => setToasts(prev => prev.filter(t => t.id !== id));

  return (
    <ToastContext.Provider value={{ push }}>
      {children}
      <div style={{position:'fixed', top: 12, right: 12, zIndex: 1080, display:'flex', flexDirection:'column', gap: '8px'}}>
        {toasts.map(t => (
          <div key={t.id} className={`toast show border-0 shadow-sm text-bg-${t.type==='error'?'danger':t.type==='success'?'success':t.type==='warning'?'warning':'secondary'}`} style={{minWidth:260}}>
            <div className="toast-header">
              <strong className="me-auto small">{t.title || (t.type||'info').toUpperCase()}</strong>
              <button className="btn-close" onClick={() => remove(t.id)} />
            </div>
            <div className="toast-body small">{t.message}</div>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  );
};

export default ToastProvider;
