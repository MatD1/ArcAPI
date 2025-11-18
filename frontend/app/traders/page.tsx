'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import ViewModal from '@/components/crud/ViewModal';

interface Trader {
  id: number;
  external_id: string;
  name: string;
  data?: any;
  synced_at?: string;
  created_at?: string;
  updated_at?: string;
}

export default function TradersPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [traders, setTraders] = useState<Trader[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedTrader, setSelectedTrader] = useState<Trader | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loadAll, setLoadAll] = useState(false);
  const pageSize = 20;

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadTraders();
  }, [isAuthenticated, router, page, loadAll]);

  const loadTraders = async () => {
    try {
      setLoading(true);
      setError('');
      if (loadAll) {
        // Load all data at once
        const result = await apiClient.getTraders(0, 10000);
        setTraders(result.data || []);
        setTotal(result.total || 0);
      } else {
        // Load paginated data
        const offset = (page - 1) * pageSize;
        const result = await apiClient.getTraders(offset, pageSize);
        setTraders(result.data || []);
        setTotal(result.total || 0);
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

  const getTradeCount = (trader: Trader): number => {
    if (!trader.data || !trader.data.trades) return 0;
    if (Array.isArray(trader.data.trades)) return trader.data.trades.length;
    return 0;
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6 flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Traders</h1>
            <p className="text-gray-600 dark:text-gray-400">View trader data from the repository</p>
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
                        Trades
                      </th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                    {traders.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                          No traders found
                        </td>
                      </tr>
                    ) : (
                      traders.map((trader) => {
                        const tradeCount = getTradeCount(trader);
                        return (
                          <tr key={trader.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                              {trader.id}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {trader.external_id}
                            </td>
                            <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                              {trader.name}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              <span className="px-2 py-1 text-xs font-semibold rounded-full bg-indigo-100 text-indigo-800 dark:bg-indigo-900/20 dark:text-indigo-300">
                                {tradeCount} {tradeCount === 1 ? 'trade' : 'trades'}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                              <button
                                onClick={() => setSelectedTrader(trader)}
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
                  Showing all {traders.length} traders
                </span>
              </div>
            )}
          </>
        )}

        {selectedTrader && (
          <ViewModal
            entity={selectedTrader as any}
            type="quest"
            onClose={() => setSelectedTrader(null)}
          />
        )}
      </div>
    </DashboardLayout>
  );
}
