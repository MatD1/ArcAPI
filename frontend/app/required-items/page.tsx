'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type { RequiredItemResponse } from '@/types';

export default function RequiredItemsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [requiredItems, setRequiredItems] = useState<RequiredItemResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadRequiredItems();
  }, [isAuthenticated, router]);

  const loadRequiredItems = async () => {
    try {
      setLoading(true);
      const response = await apiClient.getRequiredItems();
      // Sort by total quantity descending
      const sorted = response.data.sort((a, b) => b.total_quantity - a.total_quantity);
      setRequiredItems(sorted);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  if (!isAuthenticated) return null;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
            Required Items
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            Overview of all items needed for quests and hideout module upgrades
          </p>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8">Loading...</div>
        ) : (
          <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                <thead className="bg-gray-50 dark:bg-gray-900">
                  <tr>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Item
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Total Needed
                    </th>
                    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                      Used In
                    </th>
                  </tr>
                </thead>
                <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                  {requiredItems.length === 0 ? (
                    <tr>
                      <td colSpan={3} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                        No required items found
                      </td>
                    </tr>
                  ) : (
                    requiredItems.map((reqItem) => (
                      <tr key={reqItem.item.id} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                        <td className="px-6 py-4 whitespace-nowrap">
                          <div className="flex items-center">
                            {reqItem.item.image_url && (
                              <img
                                src={reqItem.item.image_url}
                                alt={reqItem.item.name}
                                className="h-10 w-10 object-contain rounded mr-3"
                                onError={(e) => {
                                  const target = e.target as HTMLImageElement;
                                  target.style.display = 'none';
                                }}
                              />
                            )}
                            <div>
                              <div className="text-sm font-medium text-gray-900 dark:text-white">
                                {reqItem.item.name}
                              </div>
                              {reqItem.item.type && (
                                <div className="text-xs text-gray-500 dark:text-gray-400">
                                  {reqItem.item.type}
                                </div>
                              )}
                            </div>
                          </div>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span className="text-sm font-semibold text-gray-900 dark:text-white">
                            {reqItem.total_quantity}
                          </span>
                        </td>
                        <td className="px-6 py-4">
                          <div className="space-y-1">
                            {reqItem.usages.map((usage, idx) => (
                              <div key={idx} className="text-sm">
                                <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium mr-2">
                                  {usage.source_type === 'quest' ? (
                                    <span className="bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-200">
                                      Quest
                                    </span>
                                  ) : (
                                    <span className="bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-200">
                                      Hideout
                                    </span>
                                  )}
                                </span>
                                <span className="text-gray-900 dark:text-white">
                                  {usage.source_name}
                                </span>
                                {usage.level && (
                                  <span className="text-gray-500 dark:text-gray-400 ml-1">
                                    (Level {usage.level})
                                  </span>
                                )}
                                <span className="text-gray-600 dark:text-gray-300 ml-2">
                                  ×{usage.quantity}
                                </span>
                              </div>
                            ))}
                          </div>
                        </td>
                      </tr>
                    ))
                  )}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

