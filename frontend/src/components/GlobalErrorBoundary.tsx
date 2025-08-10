import React from 'react';
import { useToast } from './ToastProvider';

interface State { hasError: boolean; error?: any }

class GlobalErrorBoundaryInner extends React.Component<{ children: React.ReactNode, onCatch: (e: any) => void }, State> {
  state: State = { hasError: false };
  static getDerivedStateFromError(error: any) { return { hasError: true, error }; }
  componentDidCatch(error: any, info: any) { this.props.onCatch({ error, info }); }
  render() {
    if(this.state.hasError) {
      return (
        <div className="container py-5">
          <div className="alert alert-danger">
            <h5 className="alert-heading">Unexpected Error</h5>
            <p className="mb-0 small text-break">{String(this.state.error)}</p>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}

export const GlobalErrorBoundary: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const toast = useToast();
  return (
    <GlobalErrorBoundaryInner onCatch={(e) => toast.push({ type: 'error', message: e.error?.message || 'Unexpected error' })}>
      {children}
    </GlobalErrorBoundaryInner>
  );
};
