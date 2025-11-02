'use client';

import { Suspense, useEffect, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import { apiClient, getErrorMessage } from '@/lib/api';
import { useAuthStore } from '@/store/authStore';

function CallbackContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { login, setUser } = useAuthStore();
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    const handleCallback = async () => {
      try {
        // Get temp token from URL (backend redirected here with token in query)
        const tempToken = searchParams.get('token');
        
        if (!tempToken) {
          setError('No authentication token found. Please try logging in again.');
          setLoading(false);
          return;
        }

        // Exchange temp token for actual auth data
        const response = await fetch(`${window.location.origin}/api/v1/auth/exchange-token?token=${tempToken}`);
        
        if (!response.ok) {
          const errorData = await response.json().catch(() => ({ error: 'Authentication failed' }));
          throw new Error(errorData.error || 'Authentication failed');
        }

        const data = await response.json();
        const { token, user, api_key, api_key_warning } = data;

        // Store the JWT token
        if (typeof window !== 'undefined') {
          localStorage.setItem('jwt_token', token);
        }

        // Set user in store
        setUser(user);

        // If API key was auto-created, store it and use it for login
        if (api_key) {
          // Show warning if present
          if (api_key_warning) {
            alert(`API Key Created!\n\n${api_key_warning}\n\nYour API Key: ${api_key}\n\nPlease save this key now.`);
          }
          // Use the API key to complete login
          apiClient.setAuth(api_key, token);
          await login(api_key);
          router.push('/dashboard/');
        } else {
          // User already has an API key, but we don't have it
          // We'll need to prompt them to enter it or create a new one
          // For now, just set the user and redirect - they can use API key login
          setUser(user);
          router.push('/dashboard/');
        }
      } catch (err) {
        setError(getErrorMessage(err));
        setLoading(false);
      }
    };

    handleCallback();
  }, [searchParams, router, login, setUser]);

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Completing authentication...</p>
        </div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="max-w-md w-full space-y-8 p-8 bg-white dark:bg-gray-800 rounded-lg shadow-lg">
          <div>
            <h2 className="text-center text-3xl font-extrabold text-gray-900 dark:text-white">
              Authentication Error
            </h2>
          </div>
          <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-4">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
          <button
            onClick={() => router.push('/login/')}
            className="w-full flex justify-center py-2 px-4 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700"
          >
            Return to Login
          </button>
        </div>
      </div>
    );
  }

  return null;
}

export default function GitHubCallbackPage() {
  return (
    <Suspense fallback={
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="text-center">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-600 mx-auto mb-4"></div>
          <p className="text-gray-600 dark:text-gray-400">Loading...</p>
        </div>
      </div>
    }>
      <CallbackContent />
    </Suspense>
  );
}
