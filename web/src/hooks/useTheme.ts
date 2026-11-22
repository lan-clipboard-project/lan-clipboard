/**
 * Hook for managing theme mode (light/dark/system).
 * Persists preference to localStorage and applies the 'dark' class to <html>.
 */

import { useEffect, useState, useCallback } from 'react';
import type { ThemeMode } from '../types';
import { LS_THEME_KEY } from '../utils/constants';

/** Determine the effective theme based on current mode */
function getEffectiveTheme(mode: ThemeMode): 'light' | 'dark' {
  if (mode === 'system') {
    return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
  }
  return mode;
}

/** Apply or remove the 'dark' class on <html> */
function applyTheme(theme: 'light' | 'dark'): void {
  if (theme === 'dark') {
    document.documentElement.classList.add('dark');
  } else {
    document.documentElement.classList.remove('dark');
  }
}

export function useTheme() {
  const [mode, setModeState] = useState<ThemeMode>(() => {
    const saved = localStorage.getItem(LS_THEME_KEY);
    return (saved as ThemeMode) || 'system';
  });

  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode);
    localStorage.setItem(LS_THEME_KEY, newMode);
    applyTheme(getEffectiveTheme(newMode));
  }, []);

  /** Cycle through: light → dark → system → light */
  const cycleMode = useCallback(() => {
    setMode(mode === 'light' ? 'dark' : mode === 'dark' ? 'system' : 'light');
  }, [mode, setMode]);

  // Apply theme on mount and when mode changes
  useEffect(() => {
    applyTheme(getEffectiveTheme(mode));
  }, [mode]);

  // Listen for system preference changes when mode is 'system'
  useEffect(() => {
    if (mode !== 'system') return;

    const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
    const handler = (e: MediaQueryListEvent) => {
      applyTheme(e.matches ? 'dark' : 'light');
    };

    mediaQuery.addEventListener('change', handler);
    return () => mediaQuery.removeEventListener('change', handler);
  }, [mode]);

  return { mode, setMode, cycleMode };
}
