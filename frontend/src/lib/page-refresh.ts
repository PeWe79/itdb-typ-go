import { useEffect, useRef } from 'react';

export const PAGE_REFRESH_EVENT = 'itdb:refresh';

// dispatchPageRefresh 派发全局页面数据刷新事件，供不使用 react-query 的页面监听
export function dispatchPageRefresh() {
  window.dispatchEvent(new Event(PAGE_REFRESH_EVENT));
}

// usePageRefresh 监听全局页面数据刷新事件并触发重新加载回调
export function usePageRefresh(onRefresh: () => void) {
  const callbackRef = useRef(onRefresh);
  useEffect(() => {
    callbackRef.current = onRefresh;
  });
  useEffect(() => {
    const handler = () => callbackRef.current();
    window.addEventListener(PAGE_REFRESH_EVENT, handler);
    return () => window.removeEventListener(PAGE_REFRESH_EVENT, handler);
  }, []);
}
