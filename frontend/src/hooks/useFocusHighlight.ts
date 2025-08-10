import { useEffect } from 'react';

interface Options {
  paramName?: string;
  attrName?: string; // data attribute name e.g. data-event-id
  highlightClass?: string;
  durationMs?: number;
}

/**
 * useFocusHighlight: 根据 URL ?focus=xxx 给匹配元素添加高亮并滚动定位。
 */
export function useFocusHighlight({ paramName = 'focus', attrName = 'data-id', highlightClass = 'focus-highlight', durationMs = 4000 }: Options = {}) {
  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const id = params.get(paramName);
    if (!id) return;
    // 支持多种可能的属性前缀 (data-id / data-event-id / data-reminder-id / data-task-id)
    const candidates = [
      `[${attrName}="${id}"]`,
      `[data-event-id="${id}"]`,
      `[data-reminder-id="${id}"]`,
      `[data-task-id="${id}"]`
    ];
    const selector = candidates.join(',');
    const el = document.querySelector<HTMLElement>(selector);
    if (el) {
      el.classList.add(highlightClass);
      try { el.scrollIntoView({ behavior: 'smooth', block: 'center' }); } catch {}
      const timer = setTimeout(() => el.classList.remove(highlightClass), durationMs);
      return () => clearTimeout(timer);
    }
  }, [paramName, attrName, highlightClass, durationMs]);
}

export default useFocusHighlight;
