/**
 * Zustand store for managing clipboard state.
 * Handles clips, pinned items, search, and pagination.
 */

import { create } from 'zustand';
import type { Clip, ClipType } from '../types';
import * as api from '../api/clips';
import { PAGE_SIZE } from '../utils/constants';
import { getDateKey, getNextDayCursor } from '../utils/format';

let timelineRequest = 0;

interface ClipStore {
  /* State */
  clips: Clip[];
  pinnedClips: Clip[];
  loading: boolean;
  hasMore: boolean;
  cursor: string | null;
  newerCursor: string | null;
  timelineRevision: number;
  searchQuery: string;
  searchResults: Clip[];
  isSearching: boolean;
  filterType: ClipType | null;
  lastRemoval: { id: string | null } | null;

  /* Actions */
  fetchClips: (date?: string) => Promise<boolean>;
  fetchMore: (direction?: 'older' | 'newer') => Promise<void>;
  fetchPinned: () => Promise<void>;
  addClip: (clip: Clip) => void;
  removeClip: (id: string) => void;
  updateClip: (clip: Clip) => void;
  clearClips: () => void;
  setSearch: (query: string) => void;
  performSearch: () => Promise<void>;
  setFilter: (type: ClipType | null) => void;
}

export const useClipStore = create<ClipStore>((set, get) => ({
  clips: [],
  pinnedClips: [],
  loading: false,
  hasMore: true,
  cursor: null,
  newerCursor: null,
  timelineRevision: 0,
  searchQuery: '',
  searchResults: [],
  isSearching: false,
  filterType: null,
  lastRemoval: null,

  fetchClips: async (date) => {
    const request = ++timelineRequest;
    set({ loading: true });
    try {
      const { filterType } = get();
      const before = date ? getNextDayCursor(date) : undefined;
      const result = await api.fetchClips(before, PAGE_SIZE, filterType ?? undefined);
      if (request !== timelineRequest) return false;
      if (date && !result.clips.some((clip) => getDateKey(clip.createdAt) === date)) {
        set({ loading: false });
        return false;
      }
      set((state) => ({
        clips: result.clips,
        hasMore: result.hasMore,
        cursor: result.nextCursor || null,
        newerCursor: date ? result.clips[0]?.createdAt ?? null : null,
        timelineRevision: date ? state.timelineRevision : state.timelineRevision + 1,
        loading: false,
      }));
      return true;
    } catch {
      if (request === timelineRequest) set({ loading: false });
      return false;
    }
  },

  fetchMore: async (direction = 'older') => {
    const { loading, cursor, newerCursor, filterType } = get();
    const newer = direction === 'newer';
    const pageCursor = newer ? newerCursor : cursor;
    if (loading || !pageCursor) return;

    const request = timelineRequest;
    set({ loading: true });
    try {
      const result = await api.fetchClips(pageCursor, PAGE_SIZE, filterType ?? undefined, direction);
      if (request !== timelineRequest) return;
      set((state) => {
        const currentIDs = new Set(state.clips.map((clip) => clip.id));
        const incoming = result.clips.filter((clip) => !currentIDs.has(clip.id));
        return {
          clips: newer ? [...incoming, ...state.clips] : [...state.clips, ...incoming],
          hasMore: newer ? state.hasMore : result.hasMore,
          cursor: newer ? state.cursor : result.nextCursor || null,
          newerCursor: newer ? result.nextCursor || null : state.newerCursor,
          loading: false,
        };
      });
    } catch {
      if (request === timelineRequest) set({ loading: false });
    }
  },

  fetchPinned: async () => {
    try {
      const pinned = await api.fetchPinnedClips();
      set({ pinnedClips: pinned });
    } catch {
      // Silently fail — pinned section will just be empty
    }
  },

  addClip: (clip: Clip) => {
    if (clip.pinned) {
      set((state) => ({
        pinnedClips: [clip, ...state.pinnedClips.filter((c) => c.id !== clip.id)],
      }));
    } else {
      set((state) => ({
        clips: state.newerCursor || (state.filterType && clip.type !== state.filterType)
          ? state.clips
          : [clip, ...state.clips.filter((c) => c.id !== clip.id)],
        timelineRevision: state.timelineRevision + 1,
      }));
    }
  },

  removeClip: (id: string) => {
    set((state) => ({
      clips: state.clips.filter((c) => c.id !== id),
      pinnedClips: state.pinnedClips.filter((c) => c.id !== id),
      searchResults: state.searchResults.filter((c) => c.id !== id),
      lastRemoval: { id },
      timelineRevision: state.timelineRevision + 1,
    }));
  },

  updateClip: (clip: Clip) => {
    set((state) => {
      // Remove from both lists first
      const filteredClips = state.clips.filter((c) => c.id !== clip.id);
      const filteredPinned = state.pinnedClips.filter((c) => c.id !== clip.id);
      const updatedSearchResults = state.searchResults.map((c) => (c.id === clip.id ? clip : c));

      if (clip.pinned) {
        return {
          clips: filteredClips,
          pinnedClips: [clip, ...filteredPinned],
          searchResults: updatedSearchResults,
          timelineRevision: state.timelineRevision + 1,
        };
      } else {
        const newerThanWindow = state.newerCursor && Date.parse(clip.createdAt) > Date.parse(state.newerCursor);
        return {
          clips: newerThanWindow || (state.filterType && clip.type !== state.filterType)
            ? filteredClips
            : [clip, ...filteredClips],
          pinnedClips: filteredPinned,
          searchResults: updatedSearchResults,
          timelineRevision: state.timelineRevision + 1,
        };
      }
    });
  },

  clearClips: () => {
    timelineRequest++;
    set((state) => ({
      clips: [],
      pinnedClips: [],
      searchResults: [],
      lastRemoval: { id: null },
      hasMore: false,
      cursor: null,
      newerCursor: null,
      loading: false,
      timelineRevision: state.timelineRevision + 1,
    }));
  },

  setSearch: (query: string) => {
    set({ searchQuery: query });
  },

  performSearch: async () => {
    const { searchQuery, filterType } = get();
    if (!searchQuery.trim()) {
      set({ searchResults: [], isSearching: false });
      return;
    }

    set({ isSearching: true });
    try {
      const results = await api.searchClips(searchQuery, filterType ?? undefined);
      set({ searchResults: results, isSearching: false });
    } catch {
      set({ isSearching: false });
    }
  },

  setFilter: (type: ClipType | null) => {
    set({ filterType: type });
  },
}));
