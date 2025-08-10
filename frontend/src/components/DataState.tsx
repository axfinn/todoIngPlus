import React from 'react';

interface Props<T> {
  loading: boolean;
  error?: string | null;
  data: T[] | null | undefined;
  emptyHint?: React.ReactNode;
  spinnerSize?: 'sm' | 'md';
  skeleton?: React.ReactNode; // 可选骨架占位（优先于 spinner）
  children: (items: T[]) => React.ReactNode;
}

/**
 * 通用数据状态渲染组件：加载 / 错误 / 空 / 列表
 */
export function DataState<T>({ loading, error, data, emptyHint = <p className="text-muted m-0">No data</p>, spinnerSize = 'md', skeleton, children }: Props<T>) {
  if (loading) {
    if (skeleton) {
      return <>{skeleton}</>;
    }
    return (
      <div className="d-flex justify-content-center py-4">
        <div className={`spinner-border text-primary ${spinnerSize === 'sm' ? 'spinner-border-sm' : ''}`} role="status">
          <span className="visually-hidden">Loading...</span>
        </div>
      </div>
    );
  }
  if (error) {
    return <div className="alert alert-danger py-2 px-3 mb-0">{error}</div>;
  }
  if (!data || data.length === 0) {
    return <div className="py-3 text-center">{emptyHint}</div>;
  }
  return <>{children(data)}</>;
}

export default DataState;
