import { useEffect, useState } from 'react';
import { api } from '@/lib/auth';

export type BrandSettings = {
  siteName: string;
  loginName: string;
  appName: string;
  appSubtitle: string;
  iconData: string;
};

export const defaultBrandSettings: BrandSettings = {
  siteName: 'ITDB',
  loginName: 'ITDB',
  appName: 'ITDB',
  appSubtitle: 'IT Asset Management',
  iconData: '/favicon.svg',
};

let snapshot = defaultBrandSettings;
let loadedAt = 0;
let pending: Promise<BrandSettings> | null = null;
const listeners = new Set<(value: BrandSettings) => void>();
const BRAND_CACHE_TTL_MS = 30_000;

export function setBrandSettings(value: Partial<BrandSettings>) {
  snapshot = normalizeBrandSettings(value);
  loadedAt = Date.now();
  if (typeof document !== 'undefined') {
    document.title = snapshot.siteName;
    const icon = document.querySelector<HTMLLinkElement>("link[rel='icon']");
    if (icon) icon.href = snapshot.iconData;
  }
  listeners.forEach(listener => listener(snapshot));
}

function loadBrandSettings() {
  if (loadedAt > 0 && Date.now() - loadedAt < BRAND_CACHE_TTL_MS) {
    return Promise.resolve(snapshot);
  }
  if (pending) return pending;
  pending = api<Partial<BrandSettings>>('/api/public/base', { auth: false })
    .then(next => {
      setBrandSettings(next);
      return snapshot;
    })
    .catch(() => snapshot)
    .finally(() => {
      pending = null;
    });
  return pending;
}

export function useBrandSettings(initial?: Partial<BrandSettings> | null) {
  const [value, setValue] = useState(() => (initial ? normalizeBrandSettings(initial) : snapshot));

  useEffect(() => {
    listeners.add(setValue);
    return () => {
      listeners.delete(setValue);
    };
  }, []);

  useEffect(() => {
    let cancelled = false;
    void loadBrandSettings().then(next => {
      if (!cancelled) setValue(next);
    });
    return () => {
      cancelled = true;
    };
  }, []);

  return value;
}

function normalizeBrandSettings(value: Partial<BrandSettings>): BrandSettings {
  return {
    siteName: text(value.siteName, defaultBrandSettings.siteName),
    loginName: text(value.loginName, defaultBrandSettings.loginName),
    appName: text(value.appName, defaultBrandSettings.appName),
    appSubtitle: text(value.appSubtitle, defaultBrandSettings.appSubtitle),
    iconData: text(value.iconData, defaultBrandSettings.iconData),
  };
}

function text(value: string | undefined, fallback: string) {
  return value?.trim() || fallback;
}
