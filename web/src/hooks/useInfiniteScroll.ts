/**
 * Hook for infinite scroll — loads more content when the user
 * scrolls near the bottom of the container.
 */

import { useEffect, useRef, useCallback } from 'react';
import { SCROLL_THRESHOLD } from '../utils/constants';

interface UseInfiniteScrollOptions {
  onLoadMore: () => void;
  hasMore: boolean;
  loading: boolean;
}

export function useInfiniteScroll({ onLoadMore, hasMore, loading }: UseInfiniteScrollOptions) {
  const scrollRef = useRef<HTMLDivElement>(null);

  const handleScroll = useCallback(() => {
    const el = scrollRef.current;
    if (!el || loading || !hasMore) return;

    const { scrollTop, scrollHeight, clientHeight } = el;
    if (scrollHeight - scrollTop - clientHeight < SCROLL_THRESHOLD) {
      onLoadMore();
    }
  }, [onLoadMore, hasMore, loading]);

  useEffect(() => {
    const el = scrollRef.current;
    if (!el) return;

    el.addEventListener('scroll', handleScroll, { passive: true });
    return () => el.removeEventListener('scroll', handleScroll);
  }, [handleScroll]);

  return { scrollRef };
}
