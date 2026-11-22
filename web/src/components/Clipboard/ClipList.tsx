/**
 * ClipList — main scrollable content area.
 * Renders pinned section, then date-grouped clips with infinite scroll.
 */

import { useCallback, useEffect, useLayoutEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { deleteClip, fetchClipDates } from '../../api/clips';
import { useClipStore } from '../../store/clipStore';
import { useInfiniteScroll } from '../../hooks/useInfiniteScroll';
import { getClipRows, useClipExits } from '../../hooks/useClipExits';
import type { ClipRow } from '../../hooks/useClipExits';
import { formatTime, getDateKey, getDateLabel } from '../../utils/format';
import type { ClipType } from '../../types';
import ConfirmDialog from '../Common/ConfirmDialog';
import PinnedSection from './PinnedSection';
import DateGroup from './DateGroup';
import ClipCard from './ClipCard';
import ClipPresence from './ClipPresence';
import TimelineDateNav from './TimelineDateNav';

const timeLabelGap = 5 * 60 * 1000;

function getVisibleTimelineTop(list: HTMLDivElement) {
  const inset = Number.parseFloat(getComputedStyle(list).scrollPaddingTop) || 0;
  return list.getBoundingClientRect().top + inset;
}

function groupNearbyClips(rows: ClipRow[]) {
  const groups: ClipRow[][] = [];
  rows.forEach((row, index) => {
    const previous = rows[index - 1];
    if (!previous || Date.parse(previous.clip.createdAt) - Date.parse(row.clip.createdAt) >= timeLabelGap) {
      groups.push([]);
    }
    groups[groups.length - 1].push(row);
  });
  return groups;
}

interface ClipListProps {
  onPreviewImage: (url: string) => void;
  onShowToast: (message: string, type: 'success' | 'error') => void;
}

export default function ClipList({ onPreviewImage, onShowToast }: ClipListProps) {
  const { t, i18n } = useTranslation();

  const clips = useClipStore((s) => s.clips);
  const pinnedClips = useClipStore((s) => s.pinnedClips);
  const loading = useClipStore((s) => s.loading);
  const hasMore = useClipStore((s) => s.hasMore);
  const newerCursor = useClipStore((s) => s.newerCursor);
  const timelineRevision = useClipStore((s) => s.timelineRevision);
  const filterType = useClipStore((s) => s.filterType);
  const searchQuery = useClipStore((s) => s.searchQuery);
  const searchResults = useClipStore((s) => s.searchResults);
  const isSearching = useClipStore((s) => s.isSearching);
  const fetchMore = useClipStore((s) => s.fetchMore);
  const fetchClips = useClipStore((s) => s.fetchClips);
  const removeClip = useClipStore((s) => s.removeClip);
  const [dateIndex, setDateIndex] = useState<{ type: ClipType | null; keys: string[] } | null>(null);
  const [activeDate, setActiveDate] = useState<string | null>(null);
  const [pendingDate, setPendingDate] = useState<string | null>(null);
  const [clipToDelete, setClipToDelete] = useState<string | null>(null);

  const isSearch = searchQuery.trim().length > 0;
  const displayClips = isSearch ? searchResults : clips;
  const exits = useClipExits(searchQuery, filterType);
  const pinnedRows = useMemo(() => getClipRows(pinnedClips, exits, 'pinnedClips'), [pinnedClips, exits]);
  const handleLoadMore = useCallback(() => {
    if (!isSearch) {
      fetchMore();
    }
  }, [fetchMore, isSearch]);

  const { scrollRef } = useInfiniteScroll({
    onLoadMore: handleLoadMore,
    hasMore: !isSearch && hasMore,
    loading,
  });

  // Search terms used to highlight matches inside the rendered cards.
  const highlightTerms = useMemo(
    () => (isSearch ? searchQuery.trim().split(/\s+/).filter(Boolean) : undefined),
    [isSearch, searchQuery],
  );

  const groups = useMemo(() => {
    const byDate = new Map<string, ClipRow[]>();
    const ordered = getClipRows(displayClips, exits, isSearch ? 'searchResults' : 'clips')
      .sort((a, b) => Date.parse(b.clip.createdAt) - Date.parse(a.clip.createdAt));
    for (const row of ordered) {
      const key = getDateKey(row.clip.createdAt);
      const group = byDate.get(key);
      if (group) group.push(row);
      else byDate.set(key, [row]);
    }
    return [...byDate].map(([key, rows]) => ({ key, rows }));
  }, [displayClips, exits, isSearch]);

  useEffect(() => {
    if (isSearch) return;
    const controller = new AbortController();
    fetchClipDates(filterType, controller.signal).then((keys) => {
      if (!controller.signal.aborted) setDateIndex({ type: filterType, keys });
    }).catch(() => {
      if (!controller.signal.aborted) setDateIndex(null);
    });
    return () => controller.abort();
  }, [filterType, isSearch, timelineRevision]);

  const indexedDates = !isSearch && dateIndex?.type === filterType ? dateIndex.keys : [];
  const dates = [...new Set([...indexedDates, ...groups.map(({ key }) => key)])]
    .sort().reverse().map((key) => ({ key, label: getDateLabel(key, t, i18n.resolvedLanguage) }));

  const scrollToDate = useCallback((date: string, behavior: ScrollBehavior) => {
    const list = scrollRef.current;
    const heading = document.getElementById(`timeline-${date}`);
    if (!list || !heading || !list.contains(heading)) return;

    const top = heading.getBoundingClientRect().top - getVisibleTimelineTop(list) + list.scrollTop - 16;
    heading.focus({ preventScroll: true });
    list.scrollTo({ top, behavior });
    setActiveDate(date);
  }, [scrollRef]);

  useLayoutEffect(() => {
    if (!pendingDate || isSearch || !groups.some(({ key }) => key === pendingDate)) return;
    scrollToDate(pendingDate, 'instant');
    setPendingDate(null);
  }, [groups, isSearch, pendingDate, scrollToDate]);

  const handleDateSelect = async (date: string) => {
    if (groups.some(({ key }) => key === date)) {
      const reduceMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;
      scrollToDate(date, reduceMotion ? 'instant' : 'smooth');
      return;
    }

    setPendingDate(date);
    const found = await fetchClips(date);
    if (!found || useClipStore.getState().searchQuery.trim()) {
      setPendingDate(null);
      if (!found) onShowToast(t('date.jumpFailed'), 'error');
    }
  };

  const updateActiveDate = () => {
    const list = scrollRef.current;
    if (!list || !groups.length) return;
    const top = getVisibleTimelineTop(list) + 20;
    let visibleDate = groups[0].key;
    for (const heading of list.querySelectorAll<HTMLElement>('section[data-timeline-date]')) {
      if (heading.getBoundingClientRect().top > top) break;
      visibleDate = heading.dataset.timelineDate ?? visibleDate;
    }
    setActiveDate(visibleDate);
  };

  const isEmpty = groups.length === 0 && pinnedRows.length === 0 && !loading;
  const currentDate = groups.some(({ key }) => key === activeDate) ? activeDate : groups[0]?.key ?? null;

  const handleDelete = async () => {
    if (clipToDelete === null) return;
    try {
      await deleteClip(clipToDelete);
      removeClip(clipToDelete);
    } catch {
      onShowToast(t('error.deleteFailed'), 'error');
    }
  };

  return (
    <div className="relative min-h-0 flex-1">
      {dates.length > 0 && (
        <TimelineDateNav dates={dates} activeDate={currentDate} pendingDate={pendingDate} onSelect={handleDateSelect} />
      )}
      <div ref={scrollRef} onScroll={updateActiveDate} className="timeline-scroll absolute inset-x-0 overflow-y-auto">
        <div className={`timeline-content mx-auto max-w-3xl space-y-6 px-4 ${dates.length ? 'max-[1119px]:pl-12' : ''}`}>
          {/* Search results header */}
          {isSearch && (
            <div className="flex items-center gap-2 text-sm text-gray-500 dark:text-gray-400">
              <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              <span>
                {isSearching
                  ? t('search.searching')
                  : searchResults.length > 0
                    ? `${t('search.results')} (${searchResults.length})`
                    : t('search.noResults')}
              </span>
            </div>
          )}

          {(groups.length > 0 || (!isSearch && (pinnedRows.length > 0 || newerCursor))) && (
            <div className="flow-root">
              {/* Keep spacing inside each animated row so it contracts with the content. */}
              <div className="-mb-6">
                {!isSearch && pinnedRows.length > 0 && (
                  <PinnedSection rows={pinnedRows} onPreviewImage={onPreviewImage} onRequestDelete={setClipToDelete}
                    onShowToast={onShowToast} />
                )}

                {!isSearch && newerCursor && (
                  <div className="pb-6 text-center">
                    <button type="button" onClick={() => fetchMore('newer')} disabled={loading}
                      className="rounded-lg px-3 py-1.5 text-sm font-medium text-blue-700 hover:bg-blue-50 disabled:opacity-50 dark:text-blue-300 dark:hover:bg-gray-800">
                      {t('date.loadNewer')}
                    </button>
                  </div>
                )}

                {groups.map((group) => {
                  const groupExiting = group.rows.every((row) => row.exiting);
                  return (
                    <DateGroup key={group.key} date={group.key} label={getDateLabel(group.key, t, i18n.resolvedLanguage)}
                      exiting={groupExiting}>
                      {groupNearbyClips(group.rows).flatMap((nearby) => {
                        const first = nearby.find((row) => !row.exiting);
                        const createdAt = (first ?? nearby[0]).clip.createdAt;
                        return [
                          <ClipPresence key={`time-${nearby[0].clip.id}`} exiting={!first}>
                            <div className="pb-3">
                              <time dateTime={createdAt} className="block py-1 text-center text-xs font-medium text-gray-500 dark:text-gray-400">
                                {formatTime(createdAt, i18n.resolvedLanguage)}
                              </time>
                            </div>
                          </ClipPresence>,
                          ...nearby.map(({ clip, exiting }) => (
                            <ClipPresence key={clip.id} exiting={exiting}>
                              <div className="pb-3">
                                <ClipCard clip={clip} onPreviewImage={onPreviewImage} onRequestDelete={setClipToDelete}
                                  onShowToast={onShowToast} highlightTerms={highlightTerms} />
                              </div>
                            </ClipPresence>
                          )),
                        ];
                      })}
                    </DateGroup>
                  );
                })}
              </div>
            </div>
          )}

          {/* Loading spinner */}
          {loading && (
            <div className="flex justify-center py-4">
              <div className="w-6 h-6 border-2 border-gray-300 dark:border-gray-600 border-t-blue-500 rounded-full animate-spin" />
            </div>
          )}

          {/* Empty state */}
          {isEmpty && !isSearch && (
            <div className="flex flex-col items-center justify-center py-20 text-gray-400 dark:text-gray-500">
              <svg className="w-16 h-16 mb-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M9 12h3.75M9 15h3.75M9 18h3.75m3 .75H18a2.25 2.25 0 002.25-2.25V6.108c0-1.135-.845-2.098-1.976-2.192a48.424 48.424 0 00-1.123-.08m-5.801 0c-.065.21-.1.433-.1.664 0 .414.336.75.75.75h4.5a.75.75 0 00.75-.75 2.25 2.25 0 00-.1-.664m-5.8 0A2.251 2.251 0 0113.5 2.25H15c1.012 0 1.867.668 2.15 1.586m-5.8 0c-.376.023-.75.05-1.124.08C9.095 4.01 8.25 4.973 8.25 6.108V8.25m0 0H4.875c-.621 0-1.125.504-1.125 1.125v11.25c0 .621.504 1.125 1.125 1.125h9.75c.621 0 1.125-.504 1.125-1.125V9.375c0-.621-.504-1.125-1.125-1.125H8.25zM6.75 12h.008v.008H6.75V12zm0 3h.008v.008H6.75V15zm0 3h.008v.008H6.75V18z" />
              </svg>
              <p className="text-lg font-medium">{t('empty.title')}</p>
              <p className="text-sm mt-1">{t('empty.description')}</p>
            </div>
          )}
        </div>
      </div>
      {clipToDelete !== null && (
        <ConfirmDialog
          message={t('dialog.confirmDelete')}
          open={displayClips.some(({ id }) => id === clipToDelete) || (!isSearch && pinnedClips.some(({ id }) => id === clipToDelete))}
          onConfirm={handleDelete}
          onClose={() => setClipToDelete(null)}
        />
      )}
    </div>
  );
}
