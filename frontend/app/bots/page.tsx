'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import ViewModal from '@/components/crud/ViewModal';

interface Bot {
  id: number;
  external_id: string;
  name: string;
  data?: any;
  synced_at?: string;
  created_at?: string;
  updated_at?: string;
}

export default function BotsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [bots, setBots] = useState<Bot[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedBot, setSelectedBot] = useState<Bot | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loadAll, setLoadAll] = useState(false);
  const pageSize = 20;

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadBots();
  }, [isAuthenticated, router, page, loadAll]);

  const loadBots = async () => {
    try {
      setLoading(true);
      setError('');
      if (loadAll) {
        // Load all data at once
        const result = await apiClient.getBots(0, 10000);
        setBots(result.data || []);
        setTotal(result.pagination?.total || 0);
      } else {
        // Load paginated data
        const offset = (page - 1) * pageSize;
        const result = await apiClient.getBots(offset, pageSize);
        setBots(result.data || []);
        setTotal(result.pagination?.total || 0);
      }
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleToggleLoadAll = () => {
    setLoadAll(!loadAll);
    setPage(1); // Reset to first page when toggling
  };

  const getMultilingualName = (data: any): string => {
    if (!data) return '';
    if (typeof data.name === 'string') return data.name;
    if (data.name && typeof data.name === 'object') {
      return data.name.en || data.name[Object.keys(data.name)[0]] || '';
    }
    return '';
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6 flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Bots</h1>
            <p className="text-gray-600 dark:text-gray-400">View bot data from the repository</p>
          </div>
          <button
            onClick={handleToggleLoadAll}
            className="px-4 py-2 text-sm font-medium rounded-md border border-gray-300 dark:border-gray-600 text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700"
          >
            {loadAll ? 'Show Paginated' : 'Load All'}
          </button>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8">Loading...</div>
        ) : (
          <>
            <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead className="bg-gray-50 dark:bg-gray-800">
                    <tr>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        ID
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        External ID
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Name
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Type
                      </th>
                      <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Threat
                      </th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                    {bots.length === 0 ? (
                      <tr>
                        <td colSpan={6} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                          No bots found
                        </td>
                      </tr>
                    ) : (
                      bots.map((bot) => {
                        const displayName = bot.name || getMultilingualName(bot.data) || bot.external_id;
                        const botType = bot.data?.type || '-';
                        const threat = bot.data?.threat || '-';
                        return (
                          <tr key={bot.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                              {bot.id}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {bot.external_id}
                            </td>
                            <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                              {displayName}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {botType}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm">
                              <span
                                className={`px-2 py-1 text-xs font-semibold rounded-full ${
                                  threat === 'Low'
                                    ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300'
                                    : threat === 'Medium'
                                    ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-300'
                                    : threat === 'High'
                                    ? 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300'
                                    : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                }`}
                              >
                                {threat}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                              <button
                                onClick={() => setSelectedBot(bot)}
                                className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 dark:hover:text-indigo-300"
                              >
                                View
                              </button>
                            </td>
                          </tr>
                        );
                      })
                    )}
                  </tbody>
                </table>
              </div>
            </div>
            {!loadAll && total > pageSize && (
              <div className="mt-4 flex justify-between items-center">
                <button
                  onClick={() => setPage(Math.max(1, page - 1))}
                  disabled={page === 1}
                  className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Previous
                </button>
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  Page {page} of {Math.ceil(total / pageSize)} ({total} total)
                </span>
                <button
                  onClick={() => setPage(page + 1)}
                  disabled={page >= Math.ceil(total / pageSize)}
                  className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-800 hover:bg-gray-50 dark:hover:bg-gray-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Next
                </button>
              </div>
            )}
            {loadAll && (
              <div className="mt-4 text-center">
                <span className="text-sm text-gray-700 dark:text-gray-300">
                  Showing all {bots.length} bots
                </span>
              </div>
            )}
          </>
        )}

        {selectedBot && (
          <ViewModal
            entity={selectedBot as any}
            type="quest"
            onClose={() => setSelectedBot(null)}
          />
        )}
      </div>
    </DashboardLayout>
  );
}
