'use client';

import { useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';

export default function DashboardPage() {
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
    }
  }, [isAuthenticated, router]);

  if (!isAuthenticated) {
    return null;
  }

  const showAccessWarning = user && user.role !== 'admin' && !user.can_access_data;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        {/* Access Warning Banner */}
        {showAccessWarning && (
          <div className="mb-6 bg-yellow-50 dark:bg-yellow-900/20 border-l-4 border-yellow-400 dark:border-yellow-500 p-4 rounded-r-lg">
            <div className="flex">
              <div className="flex-shrink-0">
                <svg className="h-5 w-5 text-yellow-400 dark:text-yellow-500" viewBox="0 0 20 20" fill="currentColor">
                  <path fillRule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
                </svg>
              </div>
              <div className="ml-3 flex-1">
                <h3 className="text-sm font-medium text-yellow-800 dark:text-yellow-200">
                  API Access Restricted
                </h3>
                <div className="mt-2 text-sm text-yellow-700 dark:text-yellow-300">
                  <p>
                    API access is currently closed. To request access, please contact an administrator by emailing{' '}
                    <a
                      href="mailto:info@matjweb.dev"
                      className="font-semibold underline hover:text-yellow-900 dark:hover:text-yellow-100"
                    >
                      info@matjweb.dev
                    </a>
                    .
                  </p>
                </div>
              </div>
            </div>
          </div>
        )}

        <div className="border-4 border-dashed border-gray-200 dark:border-gray-700 rounded-lg p-8">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-4">
            Dashboard
          </h1>
          <p className="text-gray-600 dark:text-gray-400 mb-6">
            Welcome to the Arc Raiders API Management Dashboard
          </p>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
            <DashboardCard
              title="Quests"
              description="Manage game quests"
              href="/quests/"
            />
            <DashboardCard
              title="Items"
              description="Manage game items"
              href="/items/"
            />
            <DashboardCard
              title="Skill Nodes"
              description="Manage skill nodes"
              href="/skill-nodes/"
            />
            <DashboardCard
              title="Hideout Modules"
              description="Manage hideout modules"
              href="/hideout-modules/"
            />
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}

function DashboardCard({
  title,
  description,
  href,
}: {
  title: string;
  description: string;
  href: string;
}) {
  return (
    <a
      href={href}
      className="block p-6 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg hover:shadow-lg transition-shadow"
    >
      <h3 className="text-lg font-semibold text-gray-900 dark:text-white mb-2">{title}</h3>
      <p className="text-sm text-gray-600 dark:text-gray-400">{description}</p>
    </a>
  );
}

