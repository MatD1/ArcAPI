import { create } from 'zustand';
import type { User } from '@/types';
import { apiClient } from '@/lib/api';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: (apiKey: string) => Promise<void>;
  loginWithOIDC: (idToken: string, refreshToken?: string, expiresIn?: number) => Promise<void>;
  logout: () => void;
  setUser: (user: User | null) => void;
  refreshUser: () => Promise<void>;
}

const AUTH_STORAGE_KEY = 'auth-storage';

const loadFromStorage = (): { user: User | null; isAuthenticated: boolean } => {
  if (typeof window === 'undefined') return { user: null, isAuthenticated: false };
  try {
    const stored = localStorage.getItem(AUTH_STORAGE_KEY);
    if (stored) {
      const parsed = JSON.parse(stored);
      return { user: parsed.user ?? null, isAuthenticated: !!parsed.isAuthenticated };
    }
  } catch {
    // ignore
  }
  return { user: null, isAuthenticated: false };
};

const saveToStorage = (user: User | null, isAuthenticated: boolean) => {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify({ user, isAuthenticated }));
  } catch {
    // ignore
  }
};

export const useAuthStore = create<AuthState>((set, get) => {
  const stored = loadFromStorage();
  
  // Initialize user from stored token on app load
  if (typeof window !== 'undefined' && stored.isAuthenticated) {
    const token = localStorage.getItem('jwt_token');
    if (token) {
      // Refresh user data in the background
      apiClient.getCurrentUser()
        .then((user) => {
          set({ user, isAuthenticated: true, isLoading: false });
          saveToStorage(user, true);
        })
        .catch(() => {
          // If fetching user fails, clear auth
          set({ user: null, isAuthenticated: false, isLoading: false });
          saveToStorage(null, false);
          apiClient.clearAuth();
        });
    }
  }
  
  return {
    ...stored,
    isLoading: stored.isAuthenticated, // Loading if we have stored auth
    login: async (apiKey: string) => {
      set({ isLoading: true });
      try {
        const response = await apiClient.login(apiKey);
        const user = await apiClient.getCurrentUser();
        set({ user, isAuthenticated: true, isLoading: false });
        saveToStorage(user, true);
      } catch (error) {
        set({ isLoading: false });
        throw error;
      }
    },
    loginWithOIDC: async (idToken: string, refreshToken?: string, expiresIn?: number) => {
      set({ isLoading: true });
      try {
        apiClient.setOIDCTokens(idToken, refreshToken, expiresIn);
        const user = await apiClient.getCurrentUser();
        set({ user, isAuthenticated: true, isLoading: false });
        saveToStorage(user, true);
      } catch (error) {
        set({ isLoading: false });
        throw error;
      }
    },
    logout: () => {
      apiClient.clearAuth();
      set({ user: null, isAuthenticated: false, isLoading: false });
      saveToStorage(null, false);
    },
    setUser: (user: User | null) => {
      set({ user, isAuthenticated: !!user, isLoading: false });
      saveToStorage(user, !!user);
    },
    refreshUser: async () => {
      try {
        const user = await apiClient.getCurrentUser();
        set({ user, isAuthenticated: true });
        saveToStorage(user, true);
      } catch (error) {
        console.error('Failed to refresh user:', error);
      }
    },
  };
});

