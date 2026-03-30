'use client';

import { Loader2 } from 'lucide-react';

interface LoadingScreenProps {
  message?: string;
}

export default function LoadingScreen({ message = 'Loading...' }: LoadingScreenProps) {
  return (
    <div className="min-h-screen relative flex flex-col items-center justify-center bg-[#0a0a0c] overflow-hidden font-sans">
      {/* Background Elements matching Login Page */}
      <div className="absolute inset-0 z-0">
        <div className="absolute top-[-10%] left-[-10%] w-[40%] h-[40%] bg-indigo-500/5 blur-[120px] rounded-full text-indigo-500" />
        <div className="absolute bottom-[-10%] right-[-10%] w-[40%] h-[40%] bg-purple-500/5 blur-[120px] rounded-full text-purple-500" />
        <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 brightness-50 contrast-150 mix-blend-overlay" />
      </div>

      <div className="relative z-10 flex flex-col items-center space-y-8 animate-in fade-in zoom-in duration-700">
        <div className="relative">
          {/* Animated rings */}
          <div className="absolute inset-0 rounded-full border-2 border-indigo-500/20 animate-ping" />
          <div className="absolute inset-[-8px] rounded-full border border-purple-500/10 animate-pulse delay-75" />
          
          <div className="relative flex items-center justify-center w-20 h-20 rounded-3xl bg-white/[0.03] backdrop-blur-xl border border-white/10 shadow-2xl">
            <Loader2 className="w-10 h-10 text-indigo-500 animate-spin" />
          </div>
        </div>

        <div className="text-center space-y-3">
          <h2 className="text-xl font-bold text-white tracking-tight">
            Authentication Subsystem
          </h2>
          <div className="flex flex-col items-center">
             <div className="px-4 py-1.5 rounded-full bg-white/[0.03] border border-white/5 backdrop-blur-md">
                <span className="text-sm font-medium text-indigo-400 animate-pulse uppercase tracking-widest text-[10px]">
                  {message}
                </span>
             </div>
          </div>
        </div>

        <div className="w-48 h-[1px] bg-gradient-to-r from-transparent via-white/10 to-transparent" />
        
        <p className="text-[10px] text-gray-500 font-bold uppercase tracking-[0.2em]">
          Strategic Data Access Node
        </p>
      </div>
    </div>
  );
}
