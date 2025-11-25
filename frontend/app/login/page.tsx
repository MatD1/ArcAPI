'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import { beginAuthentikLogin, getAuthentikConfig, type AuthentikConfig } from '@/lib/authentik';

export default function LoginPage() {
  const router = useRouter();
  const { login } = useAuthStore();
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthLoading, setOauthLoading] = useState(false);
  const [authConfig, setAuthConfig] = useState<AuthentikConfig | null>(null);
  const [configError, setConfigError] = useState('');

  useEffect(() => {
    let cancelled = false;
    getAuthentikConfig()
      .then((cfg) => {
        if (!cancelled) {
          setAuthConfig(cfg);
        }
      })
      .catch(() => {
        if (!cancelled) {
          setConfigError('Authentik is not configured. Please contact an administrator.');
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      await login(apiKey);
      router.push('/dashboard/');
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleAuthentikLogin = async () => {
    setError('');
    setOauthLoading(true);
    try {
      await beginAuthentikLogin(`${window.location.origin}/auth/callback`);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start Authentik login');
      setOauthLoading(false);
    }
  };


  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
      <div className="max-w-md w-full space-y-8 p-8 bg-white dark:bg-gray-800 rounded-lg shadow-lg">
        <div>
          <h2 className="mt-6 text-center text-3xl font-extrabold text-gray-900 dark:text-white">
            Arc Raiders API
          </h2>
          <p className="mt-2 text-center text-sm text-gray-600 dark:text-gray-400">
            Sign in to access the dashboard
          </p>
        </div>

        {/* Authentik OAuth Button */}
        <div className="space-y-3">
          <button
            type="button"
            onClick={handleAuthentikLogin}
            disabled={oauthLoading || !authConfig?.enabled}
            className="w-full flex items-center justify-center px-4 py-3 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {oauthLoading ? 'Connecting…' : 'Sign in with Authentik'}
          </button>
          {configError && (
            <p className="text-sm text-red-600 dark:text-red-400 text-center">{configError}</p>
          )}
          {!configError && (
            <p className="text-xs text-center text-gray-500 dark:text-gray-400">
              Use Authentik SSO (Discord/GitHub) to access the dashboard.
            </p>
          )}
        </div>

        <div className="relative">
          <div className="absolute inset-0 flex items-center">
            <div className="w-full border-t border-gray-300 dark:border-gray-600"></div>
          </div>
          <div className="relative flex justify-center text-sm">
            <span className="px-2 bg-white dark:bg-gray-800 text-gray-500 dark:text-gray-400">
              Or continue with API key
            </span>
          </div>
        </div>

        <form className="space-y-6" onSubmit={handleSubmit}>
          <div>
            <label htmlFor="api-key" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              API Key
            </label>
            <input
              id="api-key"
              name="api-key"
              type="password"
              required
              className="appearance-none rounded-md relative block w-full px-3 py-2 border border-gray-300 dark:border-gray-600 placeholder-gray-500 dark:placeholder-gray-400 text-gray-900 dark:text-white bg-white dark:bg-gray-700 focus:outline-none focus:ring-indigo-500 focus:border-indigo-500 focus:z-10 sm:text-sm"
              placeholder="Enter your API key"
              value={apiKey}
              onChange={(e) => setApiKey(e.target.value)}
              disabled={loading}
            />
          </div>

          {error && (
            <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-4">
              <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
            </div>
          )}

          <div>
            <button
              type="submit"
              disabled={loading}
              className="group relative w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? 'Signing in...' : 'Sign in with API Key'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

