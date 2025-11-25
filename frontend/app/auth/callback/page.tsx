'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/authStore';
import { clearPKCEContext, exchangeCodeForTokens, getPKCEContext } from '@/lib/authentik';

export default function AuthCallbackPage() {
  const router = useRouter();
  const loginWithOIDC = useAuthStore((state) => state.loginWithOIDC);
  const [status, setStatus] = useState('Validating authorization code...');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const run = async () => {
      if (typeof window === 'undefined') {
        return;
      }
      const params = new URLSearchParams(window.location.search);
      const code = params.get('code');
      const stateParam = params.get('state');
      const authError = params.get('error');
      const errorDescription = params.get('error_description');

      if (authError) {
        setError(`${authError}${errorDescription ? `: ${errorDescription}` : ''}`);
        clearPKCEContext();
        return;
      }

      if (!code) {
        setError('Missing authorization code. Please try signing in again.');
        clearPKCEContext();
        return;
      }

      const pkce = getPKCEContext();
      if (!pkce) {
        setError('Login session expired or missing. Please start over.');
        return;
      }

      if (!stateParam || stateParam !== pkce.state) {
        setError('State validation failed. Please try signing in again.');
        clearPKCEContext();
        return;
      }

      try {
        setStatus('Exchanging code for tokens...');
        const tokens = await exchangeCodeForTokens(code, pkce.redirectUri, pkce.verifier);
        if (!tokens?.id_token) {
          throw new Error('Authentik did not return an ID token.');
        }
        setStatus('Fetching user information...');
        await loginWithOIDC(tokens.id_token, tokens.refresh_token, tokens.expires_in);
        clearPKCEContext();
        setStatus('Login successful. Redirecting...');
        router.replace('/dashboard/');
      } catch (err) {
        const message = err instanceof Error ? err.message : 'Unable to complete login.';
        setError(message);
        clearPKCEContext();
      }
    };

    run();
  }, [loginWithOIDC, router]);

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <div className="max-w-md w-full bg-white dark:bg-gray-800 rounded-lg shadow p-8 text-center space-y-4">
        {!error ? (
          <>
            <div className="animate-spin h-10 w-10 border-4 border-indigo-500 border-t-transparent rounded-full mx-auto" />
            <h1 className="text-lg font-semibold text-gray-900 dark:text-white">Signing you in…</h1>
            <p className="text-sm text-gray-600 dark:text-gray-400">{status}</p>
          </>
        ) : (
          <>
            <div className="h-10 w-10 rounded-full bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-300 flex items-center justify-center mx-auto">
              !
            </div>
            <h1 className="text-lg font-semibold text-gray-900 dark:text-white">Authentication Failed</h1>
            <p className="text-sm text-gray-600 dark:text-gray-400">{error}</p>
            <button
              onClick={() => router.replace('/login/')}
              className="mt-4 px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500"
            >
              Back to Login
            </button>
          </>
        )}
      </div>
    </div>
  );
}

