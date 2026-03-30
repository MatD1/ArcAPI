'use client';

import { useEffect, useState, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { supabase } from '@/lib/supabase';
import { useAuthStore } from '@/store/authStore';

import LoadingScreen from '@/components/ui/LoadingScreen';

export default function AuthCallbackPage() {
  const router = useRouter();
  const { isAuthenticated, user, isLoading } = useAuthStore();
  const [status, setStatus] = useState('Authenticating with Supabase...');
  const [error, setError] = useState<string | null>(null);
  const redirectAttempted = useRef(false);

  useEffect(() => {
    const verifySession = async () => {
      try {
        const { data: { session }, error: sessionError } = await supabase.auth.getSession();
        if (sessionError) throw sessionError;
        
        if (session) {
          setStatus('Syncing profiles...');
        }
      } catch (err) {
        console.error('Session verification error:', err);
        setError(err instanceof Error ? err.message : 'Unable to verify session.');
      }
    };

    verifySession();
  }, []);

  useEffect(() => {
    if (isAuthenticated && user && !redirectAttempted.current) {
      setStatus('Redirecting...');
      redirectAttempted.current = true;
      router.replace('/dashboard/');
    }
  }, [isAuthenticated, user, router]);

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[#0a0a0c] px-4">
        <div className="max-w-md w-full bg-white/[0.03] backdrop-blur-xl border border-white/10 rounded-3xl p-8 text-center space-y-6">
          <div className="h-16 w-16 rounded-2xl bg-red-500/10 border border-red-500/20 text-red-400 flex items-center justify-center mx-auto text-2xl font-black">
            !
          </div>
          <h1 className="text-xl font-bold text-white tracking-tight">Access Denied</h1>
          <p className="text-sm text-gray-400 font-medium">{error}</p>
          <button
            onClick={() => router.replace('/login/')}
            className="w-full py-3.5 px-6 rounded-xl text-sm font-bold text-white bg-indigo-600 hover:bg-indigo-700 transition-all active:scale-[0.98]"
          >
            Terminal Return
          </button>
        </div>
      </div>
    );
  }

  return <LoadingScreen message={status} />;
}

