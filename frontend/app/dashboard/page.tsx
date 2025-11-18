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

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
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
            <DashboardCard
              title="Enemy Types"
              description="Manage enemy types"
              href="/enemy-types/"
            />
            <DashboardCard
              title="Bots"
              description="View bot data"
              href="/bots/"
            />
            <DashboardCard
              title="Maps"
              description="View map data"
              href="/maps/"
            />
            <DashboardCard
              title="Traders"
              description="View trader data"
              href="/traders/"
            />
            <DashboardCard
              title="Projects"
              description="View project data"
              href="/projects/"
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

