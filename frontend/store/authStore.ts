import { create } from 'zustand';
import type { User } from '@/types';
import { apiClient } from '@/lib/api';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  login: (apiKey: string) => Promise<void>;
  logout: () => void;
  setUser: (user: User | null) => void;
}

// Load from localStorage on init
const loadFromStorage = (): { user: User | null; isAuthenticated: boolean } => {
  if (typeof window === 'undefined') return { user: null, isAuthenticated: false };
  try {
    const stored = localStorage.getItem('auth-storage');
    if (stored) {
      const parsed = JSON.parse(stored);
      return { user: parsed.user, isAuthenticated: parsed.isAuthenticated };
    }
  } catch {}
  return { user: null, isAuthenticated: false };
};

const saveToStorage = (user: User | null, isAuthenticated: boolean) => {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem('auth-storage', JSON.stringify({ user, isAuthenticated }));
  } catch {}
};

export const useAuthStore = create<AuthState>((set) => {
  const stored = loadFromStorage();
  return {
    ...stored,
    login: async (apiKey: string) => {
      try {
        const response = await apiClient.login(apiKey);
        set({ user: response.user, isAuthenticated: true });
        saveToStorage(response.user, true);
      } catch (error) {
        throw error;
      }
    },
    logout: () => {
      apiClient.clearAuth();
      set({ user: null, isAuthenticated: false });
      saveToStorage(null, false);
    },
    setUser: (user: User | null) => {
      set({ user, isAuthenticated: !!user });
      saveToStorage(user, !!user);
    },
  };
});

