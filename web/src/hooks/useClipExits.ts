import { useEffect, useMemo, useState } from 'react';
import { useClipStore } from '../store/clipStore';
import type { Clip, ClipType } from '../types';

type ClipListName = 'clips' | 'pinnedClips' | 'searchResults';
type RemovedClip = { clip: Clip; index: number };
type RemovalBatch = {
  view: string;
  lists: Partial<Record<ClipListName, RemovedClip[]>>;
};

export interface ClipRow {
  clip: Clip;
  exiting: boolean;
}

export const clipExitDuration = 240;

function viewKey(query: string, filter: ClipType | null) {
  return JSON.stringify([query.trim(), filter]);
}

export function useClipExits(query: string, filter: ClipType | null) {
  const [batches, setBatches] = useState<RemovalBatch[]>([]);
  const view = viewKey(query, filter);

  useEffect(() => {
    const timers = new Set<number>();
    const motion = window.matchMedia('(prefers-reduced-motion: reduce)');
    const unsubscribe = useClipStore.subscribe((current, previous) => {
      const removal = current.lastRemoval;
      if (!removal || removal === previous.lastRemoval || motion.matches) return;

      const names: ClipListName[] = previous.searchQuery.trim()
        ? ['searchResults']
        : ['pinnedClips', 'clips'];
      const batch: RemovalBatch = { view: viewKey(previous.searchQuery, previous.filterType), lists: {} };
      for (const name of names) {
        const removed = previous[name].flatMap((clip, index) =>
          removal.id === null || clip.id === removal.id ? [{ clip, index }] : []);
        if (removed.length) batch.lists[name] = removed;
      }
      if (!Object.keys(batch.lists).length) return;

      setBatches((active) => [...active, batch]);
      const timer = window.setTimeout(() => {
        setBatches((active) => active.filter((entry) => entry !== batch));
        timers.delete(timer);
      }, clipExitDuration + 32);
      timers.add(timer);
    });
    return () => {
      unsubscribe();
      for (const timer of timers) window.clearTimeout(timer);
    };
  }, []);

  return useMemo(() => batches.filter((batch) => batch.view === view), [batches, view]);
}

export function getClipRows(clips: Clip[], batches: RemovalBatch[], name: ClipListName): ClipRow[] {
  const rows: ClipRow[] = clips.map((clip) => ({ clip, exiting: false }));
  const present = new Set(clips.map(({ id }) => id));
  // Restore the most recent removal first so earlier indexes still line up.
  for (const { lists } of [...batches].reverse()) {
    for (const { clip, index } of lists[name] ?? []) {
      if (present.has(clip.id)) continue;
      rows.splice(index, 0, { clip, exiting: true });
      present.add(clip.id);
    }
  }
  return rows;
}
