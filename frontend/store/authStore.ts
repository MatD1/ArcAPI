import { create } from 'zustand';
import type { User } from '@/types';
import { apiClient } from '@/lib/api';
import { supabase } from '@/lib/supabase';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  authError: string | null;
  signInWithGithub: () => Promise<void>;
  login: (apiKey: string) => Promise<void>;
  logout: () => void;
  setUser: (user: User | null) => void;
  refreshUser: () => Promise<void>;
  initialize: () => Promise<void>;
  clearError: () => void;
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
  } catch (error) {
    console.warn('[AuthStore] Failed to load auth from storage:', error);
  }
  return { user: null, isAuthenticated: false };
};

const saveToStorage = (user: User | null, isAuthenticated: boolean) => {
  if (typeof window === 'undefined') return;
  try {
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify({ user, isAuthenticated }));
  } catch (error) {
    console.warn('[AuthStore] Failed to save auth to storage:', error);
  }
};

export const useAuthStore = create<AuthState>((set, get) => {
  const stored = loadFromStorage();

  // Handle Supabase Auth State Changes
  if (typeof window !== 'undefined') {
    supabase.auth.onAuthStateChange(async (event, session) => {
      console.debug('[AuthStore] Supabase auth state changed:', event);
      
      if (session) {
        // Sync token with API Client
        apiClient.setSupabaseToken(session.access_token);
        
        // Fetch User and Role from backend/profiles
        try {
          console.debug('[AuthStore] Fetching user data for session:', session.user.email);
          const user = await apiClient.getCurrentUser();
          
          // Double check role in Supabase profiles table as requested
          const { data: profile, error: profileError } = await supabase
            .from('profiles')
            .select('role')
            .eq('id', session.user.id)
            .single();
            
          if (profileError) {
            console.warn('[AuthStore] Failed to fetch user profile:', profileError);
          } else if (profile) {
            user.role = profile.role;
          }
          
          console.debug('[AuthStore] User authenticated successfully:', user.email);
          set({ user, isAuthenticated: true, isLoading: false, authError: null });
          saveToStorage(user, true);
        } catch (error) {
          const errorMsg = error instanceof Error ? error.message : String(error);
          console.error('[AuthStore] Failed to sync user after auth change:', errorMsg, error);
          set({ authError: `Failed to load user data: ${errorMsg}`, isLoading: false });
        }
      } else if (event === 'SIGNED_OUT') {
        console.debug('[AuthStore] User signed out');
        set({ user: null, isAuthenticated: false, isLoading: false, authError: null });
        saveToStorage(null, false);
        apiClient.clearAuth();
      }
    });
  }

  return {
    ...stored,
    isLoading: false,
    authError: null,
    initialize: async () => {
      console.debug('[AuthStore] Initializing auth...');
      set({ isLoading: true });
      try {
        const { data: { session }, error: sessionError } = await supabase.auth.getSession();
        
        if (sessionError) {
          console.warn('[AuthStore] Failed to get session:', sessionError);
          set({ isLoading: false, authError: sessionError.message });
          return;
        }
        
        if (session) {
          apiClient.setSupabaseToken(session.access_token);
          try {
            console.debug('[AuthStore] Fetching current user...');
            const user = await apiClient.getCurrentUser();
            const { data: profile, error: profileError } = await supabase
              .from('profiles')
              .select('role')
              .eq('id', session.user.id)
              .single();
              
            if (profileError) {
              console.warn('[AuthStore] Failed to fetch user profile:', profileError);
            } else if (profile) {
              user.role = profile.role;
            }
            console.debug('[AuthStore] Initialization complete, user authenticated');
            set({ user, isAuthenticated: true, isLoading: false, authError: null });
            saveToStorage(user, true);
          } catch (error) {
            const errorMsg = error instanceof Error ? error.message : String(error);
            console.error('[AuthStore] Failed to fetch current user:', errorMsg, error);
            set({ isLoading: false, authError: `Failed to load user: ${errorMsg}` });
          }
        } else {
          console.debug('[AuthStore] No active session');
          set({ isLoading: false });
        }
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        console.error('[AuthStore] Initialization failed:', errorMsg, error);
        set({ isLoading: false, authError: `Initialization failed: ${errorMsg}` });
      }
    },
    signInWithGithub: async () => {
      console.debug('[AuthStore] Initiating GitHub login...');
      set({ isLoading: true, authError: null });
      const { error } = await supabase.auth.signInWithOAuth({
        provider: 'github',
        options: {
          redirectTo: `${window.location.origin}/auth/callback`,
        },
      });
      if (error) {
        const errorMsg = error.message || 'Unknown GitHub OAuth error';
        console.error('[AuthStore] GitHub OAuth failed:', errorMsg, error);
        set({ isLoading: false, authError: errorMsg });
        throw error;
      }
    },
    login: async (apiKey: string) => {
      console.debug('[AuthStore] Logging in with API key...');
      set({ isLoading: true, authError: null });
      try {
        const response = await apiClient.login(apiKey);
        console.debug('[AuthStore] API key login successful');
        const user = await apiClient.getCurrentUser();
        set({ user, isAuthenticated: true, isLoading: false, authError: null });
        saveToStorage(user, true);
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        console.error('[AuthStore] API key login failed:', errorMsg, error);
        set({ isLoading: false, authError: `Login failed: ${errorMsg}` });
        throw error;
      }
    },
    logout: async () => {
      console.debug('[AuthStore] Logging out...');
      await supabase.auth.signOut();
      apiClient.clearAuth();
      set({ user: null, isAuthenticated: false, isLoading: false, authError: null });
      saveToStorage(null, false);
      console.debug('[AuthStore] Logout complete');
    },
    setUser: (user: User | null) => {
      console.debug('[AuthStore] Setting user:', user?.email);
      set({ user, isAuthenticated: !!user, isLoading: false, authError: null });
      saveToStorage(user, !!user);
    },
    refreshUser: async () => {
      console.debug('[AuthStore] Refreshing user data...');
      try {
        const user = await apiClient.getCurrentUser();
        const { data: { session }, error: sessionError } = await supabase.auth.getSession();
        
        if (sessionError) {
          console.warn('[AuthStore] Failed to get session for refresh:', sessionError);
        } else if (session) {
          const { data: profile, error: profileError } = await supabase
            .from('profiles')
            .select('role')
            .eq('id', session.user.id)
            .single();
            
          if (profileError) {
            console.warn('[AuthStore] Failed to fetch profile during refresh:', profileError);
          } else if (profile) {
            user.role = profile.role;
          }
        }
        console.debug('[AuthStore] User refresh complete');
        set({ user, isAuthenticated: true, authError: null });
        saveToStorage(user, true);
      } catch (error) {
        const errorMsg = error instanceof Error ? error.message : String(error);
        console.error('[AuthStore] Failed to refresh user:', errorMsg, error);
        set({ authError: `Failed to refresh user: ${errorMsg}` });
      }
    },
    clearError: () => {
      set({ authError: null });
    },
  };
});

