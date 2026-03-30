'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { useAuthStore } from '@/store/authStore';
import { getErrorMessage } from '@/lib/api';
import { Github, Key, Loader2, Lock, ShieldCheck } from 'lucide-react';
import { clsx } from 'clsx';

export default function LoginPage() {
  const router = useRouter();
  const { login, signInWithGithub } = useAuthStore();
  const [apiKey, setApiKey] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [oauthLoading, setOauthLoading] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!apiKey.trim()) return;
    
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

  const handleGithubLogin = async () => {
    setError('');
    setOauthLoading(true);
    try {
      await signInWithGithub();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to start GitHub login');
      setOauthLoading(false);
    }
  };

  return (
    <div className="min-h-screen relative flex items-center justify-center bg-[#0a0a0c] overflow-hidden font-sans">
      {/* Dynamic Background Elements */}
      <div className="absolute inset-0 z-0">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-indigo-500/10 blur-[120px] rounded-full" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-purple-500/10 blur-[120px] rounded-full" />
        <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 brightness-50 contrast-150 mix-blend-overlay" />
      </div>

      <div className="relative z-10 max-w-md w-full px-6 py-12">
        <div className="text-center mb-10">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-gradient-to-tr from-indigo-600 to-purple-600 mb-6 shadow-xl shadow-indigo-500/20 ring-1 ring-white/20">
            <ShieldCheck className="w-8 h-8 text-white" />
          </div>
          <h1 className="text-4xl font-black text-white tracking-tight mb-2 selection:bg-indigo-500">
            ARC <span className="text-transparent bg-clip-text bg-gradient-to-r from-indigo-400 to-purple-400">API</span>
          </h1>
          <p className="text-gray-400 text-sm font-medium">
            Next-generation data management interface
          </p>
        </div>

        <div className="bg-white/[0.03] backdrop-blur-xl border border-white/10 rounded-3xl p-8 shadow-2xl relative overflow-hidden group">
          {/* Subtle hover glow */}
          <div className="absolute inset-0 bg-gradient-to-br from-indigo-500/5 to-purple-500/5 opacity-0 group-hover:opacity-100 transition-opacity duration-500" />
          
          <div className="relative space-y-8">
            {/* GitHub Section */}
            <div>
              <button
                onClick={handleGithubLogin}
                disabled={oauthLoading || loading}
                className={clsx(
                  "w-full flex items-center justify-center gap-3 px-6 py-3.5 rounded-xl text-sm font-bold text-white transition-all duration-300",
                  "bg-[#24292f] hover:bg-[#2b3137] active:scale-[0.98] shadow-lg shadow-black/20",
                  "border border-white/5 hover:border-white/10",
                  "disabled:opacity-50 disabled:cursor-not-allowed disabled:scale-100"
                )}
              >
                {oauthLoading ? (
                  <Loader2 className="w-5 h-5 animate-spin" />
                ) : (
                  <Github className="w-5 h-5" />
                )}
                <span>Continue with GitHub</span>
              </button>
              <p className="mt-4 text-center text-[11px] text-gray-500 uppercase tracking-widest font-semibold">
                Secure enterprise authentication
              </p>
            </div>

            <div className="relative">
              <div className="absolute inset-0 flex items-center">
                <div className="w-full border-t border-white/5"></div>
              </div>
              <div className="relative flex justify-center text-[10px] uppercase font-bold tracking-widest">
                <span className="px-3 bg-[#0d0d0f] text-gray-500">Alternative Access</span>
              </div>
            </div>

            {/* API Key Section */}
            <form onSubmit={handleSubmit} className="space-y-4">
              <div className="group/input">
                <label className="block text-xs font-bold text-gray-400 mb-1.5 ml-1 transition-colors group-focus-within/input:text-indigo-400">
                  DEVELOPER API KEY
                </label>
                <div className="relative">
                  <div className="absolute inset-y-0 left-0 pl-4 flex items-center pointer-events-none">
                    <Key className="h-4 w-4 text-gray-500 group-focus-within/input:text-indigo-400 transition-colors" />
                  </div>
                  <input
                    type="password"
                    autoComplete="off"
                    className={clsx(
                      "block w-full pl-11 pr-4 py-3 bg-black/40 border border-white/5 rounded-xl text-white placeholder-gray-600",
                      "focus:outline-none focus:ring-2 focus:ring-indigo-500/50 focus:border-indigo-500 transition-all text-sm"
                    )}
                    placeholder="sk_arc_••••••••••••"
                    value={apiKey}
                    onChange={(e) => setApiKey(e.target.value)}
                    disabled={loading || oauthLoading}
                  />
                </div>
              </div>

              {error && (
                <div className="flex items-center gap-2 p-3.5 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-xs font-medium animate-in fade-in slide-in-from-top-1">
                  <Lock className="w-4 h-4 shrink-0" />
                  <p>{error}</p>
                </div>
              )}

              <button
                type="submit"
                disabled={loading || oauthLoading || !apiKey.trim()}
                className={clsx(
                  "w-full py-3.5 px-6 rounded-xl text-sm font-bold text-white transition-all duration-300",
                  "bg-gradient-to-r from-indigo-600 to-indigo-700 hover:from-indigo-500 hover:to-indigo-600 active:scale-[0.98] shadow-lg shadow-indigo-500/20",
                  "disabled:opacity-30 disabled:cursor-not-allowed disabled:scale-100 disabled:shadow-none"
                )}
              >
                {loading ? (
                  <div className="flex items-center justify-center gap-2">
                    <Loader2 className="w-4 h-4 animate-spin" />
                    <span>Verifying Access…</span>
                  </div>
                ) : (
                  "Identify Subsystem"
                )}
              </button>
            </form>
          </div>
        </div>

        <p className="mt-8 text-center text-xs text-gray-600 font-medium">
          Access is strictly monitored. Unauthorized attempts are logged.
        </p>
      </div>
    </div>
  );
}

