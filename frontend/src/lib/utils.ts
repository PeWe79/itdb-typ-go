import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export type ItdbTheme = 'dark' | 'light';

const themeStorageKey = 'itdb.theme';

export function getInitialTheme(): ItdbTheme {
  if (typeof window === 'undefined') return 'dark';
  return window.localStorage.getItem(themeStorageKey) === 'light' ? 'light' : 'dark';
}

export function applyTheme(theme: ItdbTheme) {
  if (typeof document === 'undefined') return;
  document.documentElement.dataset.itdbTheme = theme;
  document.documentElement.style.colorScheme = theme;
}

export function persistTheme(theme: ItdbTheme) {
  if (typeof window === 'undefined') return;
  window.localStorage.setItem(themeStorageKey, theme);
}

export function toggleTheme(theme: ItdbTheme): ItdbTheme {
  return theme === 'dark' ? 'light' : 'dark';
}
