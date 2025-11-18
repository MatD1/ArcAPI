'use client';

import { useState, useEffect } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { useAuthStore } from '@/store/authStore';
import Link from 'next/link';

const navigation = [
  { name: 'Dashboard', href: '/dashboard/', icon: '📊' },
  { name: 'Quests', href: '/quests/', icon: '🎯' },
  { name: 'Items', href: '/items/', icon: '📦' },
  { name: 'Blueprints', href: '/blueprints/', icon: '📘' },
  { name: 'Required Items', href: '/required-items/', icon: '📋' },
  { name: 'Skill Nodes', href: '/skill-nodes/', icon: '🔧' },
  { name: 'Hideout Modules', href: '/hideout-modules/', icon: '🏠' },
  { name: 'Enemy Types', href: '/enemy-types/', icon: '👾' },
  { name: 'Bots', href: '/bots/', icon: '🤖' },
  { name: 'Maps', href: '/maps/', icon: '🗺️' },
  { name: 'Traders', href: '/traders/', icon: '💼' },
  { name: 'Projects', href: '/projects/', icon: '🚀' },
];

const adminNavigation = [
  { name: 'Users', href: '/users/', icon: '👥' },
  { name: 'API Keys', href: '/api-keys/', icon: '🔑' },
  { name: 'JWT Tokens', href: '/jwt-tokens/', icon: '🎫' },
  { name: 'Alerts', href: '/alerts/', icon: '🚨' },
  { name: 'Audit Logs', href: '/logs/', icon: '📋' },
  { name: 'API Test', href: '/dashboard/api-test/', icon: '🧪' },
  { name: 'Appwrite', href: '/appwrite/', icon: '🗄️' },
  { name: 'Export Data', href: '/export/', icon: '📥' },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const { user, logout } = useAuthStore();
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleLogout = () => {
    logout();
    router.push('/login/');
  };

  // Close mobile menu when route changes
  useEffect(() => {
    setMobileMenuOpen(false);
  }, [pathname]);

  // Close mobile menu on escape key
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setMobileMenuOpen(false);
      }
    };
    document.addEventListener('keydown', handleEscape);
    return () => document.removeEventListener('keydown', handleEscape);
  }, []);

  if (!user) {
    return null;
  }

  const isAdmin = user.role === 'admin';

  const NavLink = ({ item, onClick }: { item: typeof navigation[0]; onClick?: () => void }) => {
    const isActive = pathname === item.href || pathname?.startsWith(item.href + '/');
    return (
      <Link
        href={item.href}
        onClick={onClick}
        className={`flex items-center px-4 py-3 text-sm font-medium rounded-lg transition-colors ${
          isActive
            ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/20 dark:text-indigo-300'
            : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
        }`}
      >
        <span className="mr-3 text-lg">{item.icon}</span>
        {item.name}
      </Link>
    );
  };

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      {/* Mobile menu overlay */}
      {mobileMenuOpen && (
        <div
          className="fixed inset-0 z-40 bg-gray-600 bg-opacity-75 lg:hidden"
          onClick={() => setMobileMenuOpen(false)}
        />
      )}

      {/* Sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-64 bg-white dark:bg-gray-800 border-r border-gray-200 dark:border-gray-700 transform transition-transform duration-300 ease-in-out lg:translate-x-0 ${
          mobileMenuOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex flex-col h-full">
          {/* Logo/Brand */}
          <div className="flex items-center justify-between h-16 px-6 border-b border-gray-200 dark:border-gray-700">
            <h1 className="text-xl font-bold text-gray-900 dark:text-white">Arc Raiders API</h1>
            <button
              onClick={() => setMobileMenuOpen(false)}
              className="lg:hidden text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Navigation */}
          <nav className="flex-1 px-4 py-6 space-y-1 overflow-y-auto">
            <div className="mb-6">
              <h2 className="px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                Data
              </h2>
              <div className="space-y-1">
                {navigation.map((item) => (
                  <NavLink key={item.name} item={item} onClick={() => setMobileMenuOpen(false)} />
                ))}
              </div>
            </div>

            {isAdmin && (
              <div>
                <h2 className="px-4 text-xs font-semibold text-gray-500 dark:text-gray-400 uppercase tracking-wider mb-2">
                  Administration
                </h2>
                <div className="space-y-1">
                  {adminNavigation.map((item) => (
                    <NavLink key={item.name} item={item} onClick={() => setMobileMenuOpen(false)} />
                  ))}
                </div>
              </div>
            )}
          </nav>

          {/* User Info & Logout */}
          <div className="border-t border-gray-200 dark:border-gray-700 p-4">
            <div className="flex items-center mb-3">
              <div className="flex-shrink-0">
                <div className="h-10 w-10 rounded-full bg-indigo-100 dark:bg-indigo-900/20 flex items-center justify-center">
                  <span className="text-indigo-600 dark:text-indigo-400 font-semibold text-sm">
                    {user.username.charAt(0).toUpperCase()}
                  </span>
                </div>
              </div>
              <div className="ml-3 flex-1 min-w-0">
                <p className="text-sm font-medium text-gray-900 dark:text-white truncate">{user.username}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">{user.role}</p>
              </div>
            </div>
            <button
              onClick={handleLogout}
              className="w-full flex items-center justify-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-colors"
            >
              <svg className="w-4 h-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 16l4-4m0 0l-4-4m4 4H7m6 4v1a3 3 0 01-3 3H6a3 3 0 01-3-3V7a3 3 0 013-3h4a3 3 0 013 3v1" />
              </svg>
              Logout
            </button>
          </div>
        </div>
      </aside>

      {/* Main Content */}
      <div className="lg:pl-64">
        {/* Top Bar (Mobile) */}
        <div className="sticky top-0 z-30 lg:hidden bg-white dark:bg-gray-800 border-b border-gray-200 dark:border-gray-700">
          <div className="flex items-center justify-between h-16 px-4">
            <button
              onClick={() => setMobileMenuOpen(true)}
              className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
            >
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            <h1 className="text-lg font-bold text-gray-900 dark:text-white">Arc Raiders API</h1>
            <div className="flex items-center space-x-2">
              <span className="text-sm text-gray-700 dark:text-gray-300">{user.username}</span>
              <button
                onClick={handleLogout}
                className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"
              >
                Logout
              </button>
            </div>
          </div>
        </div>

        {/* Warning Banner - Always Visible */}
        <div className="w-full bg-yellow-50 dark:bg-yellow-900/20 border-b border-yellow-200 dark:border-yellow-800">
          <div className="mx-auto px-4 sm:px-6 lg:px-8 py-3">
            <div className="flex items-start">
              <div className="flex-shrink-0">
                <svg
                  className="h-5 w-5 text-yellow-400 dark:text-yellow-500 mt-0.5"
                  viewBox="0 0 20 20"
                  fill="currentColor"
                  aria-hidden="true"
                >
                  <path
                    fillRule="evenodd"
                    d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <p className="text-sm text-yellow-800 dark:text-yellow-200">
                  <span className="font-semibold">API Access Restricted:</span>{' '}
                  API access is disabled for all users. To request access, contact the developer via Discord or email{' '}
                  <a
                    href="mailto:admin@matjweb.dev"
                    className="font-semibold underline hover:text-yellow-900 dark:hover:text-yellow-100 transition-colors"
                  >
                    admin@matjweb.dev
                  </a>
                  .
                </p>
              </div>
            </div>
          </div>
        </div>

        {/* Page Content */}
        <main className="py-6 px-4 sm:px-6 lg:px-8">
          {children}
        </main>
      </div>
    </div>
  );
}
