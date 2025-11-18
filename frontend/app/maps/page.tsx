'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import ViewModal from '@/components/crud/ViewModal';

interface Map {
  id: number;
  external_id: string;
  name: string;
  data?: any;
  synced_at?: string;
  created_at?: string;
  updated_at?: string;
}

export default function MapsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [maps, setMaps] = useState<Map[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedMap, setSelectedMap] = useState<Map | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loadAll, setLoadAll] = useState(false);
  const pageSize = 20;

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadMaps();
  }, [isAuthenticated, router, page, loadAll]);

  const loadMaps = async () => {
    try {
      setLoading(true);
      setError('');
      if (loadAll) {
        // Load all data at once
        const result = await apiClient.getMaps(0, 10000);
        setMaps(result.data || []);
        setTotal(result.pagination?.total || 0);
      } else {
        // Load paginated data
        const offset = (page - 1) * pageSize;
        const result = await apiClient.getMaps(offset, pageSize);
        setMaps(result.data || []);
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
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Maps</h1>
            <p className="text-gray-600 dark:text-gray-400">View map data from the repository</p>
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
                        Image
                      </th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                    {maps.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                          No maps found
                        </td>
                      </tr>
                    ) : (
                      maps.map((map) => {
                        const displayName = map.name || getMultilingualName(map.data) || map.external_id;
                        const imageUrl = map.data?.image;
                        return (
                          <tr key={map.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                              {map.id}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {map.external_id}
                            </td>
                            <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                              {displayName}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm">
                              {imageUrl ? (
                                <img
                                  src={imageUrl}
                                  alt={displayName}
                                  className="h-12 w-12 object-cover rounded"
                                  onError={(e) => {
                                    (e.target as HTMLImageElement).style.display = 'none';
                                  }}
                                />
                              ) : (
                                <span className="text-gray-400">-</span>
                              )}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                              <button
                                onClick={() => setSelectedMap(map)}
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
                  Showing all {maps.length} maps
                </span>
              </div>
            )}
          </>
        )}

        {selectedMap && (
          <ViewModal
            entity={selectedMap as any}
            type="quest"
            onClose={() => setSelectedMap(null)}
          />
        )}
      </div>
    </DashboardLayout>
  );
}
