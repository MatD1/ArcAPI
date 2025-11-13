'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type { Item, UserBlueprintProgress } from '@/types';

export default function BlueprintsPage() {
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();
  const [blueprints, setBlueprints] = useState<Item[]>([]);
  const [myProgress, setMyProgress] = useState<UserBlueprintProgress[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [saving, setSaving] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [filterStatus, setFilterStatus] = useState<'all' | 'consumed' | 'available'>('all');

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadData();
  }, [isAuthenticated, router]);

  const loadData = async () => {
    try {
      setLoading(true);
      setError('');
      
      // Load all blueprints
      const blueprintsResponse = await apiClient.getBlueprints();
      setBlueprints(blueprintsResponse.data);

      // Load user's blueprint progress
      try {
        const progressResponse = await apiClient.getMyBlueprintProgress();
        setMyProgress(progressResponse.data);
      } catch (err) {
        // User might not have any progress yet - this is OK
        console.log('No progress found:', err);
        setMyProgress([]);
      }
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleToggleBlueprint = async (blueprint: Item) => {
    try {
      setSaving(true);
      
      // Find current progress for this blueprint
      const currentProgress = myProgress.find(p => p.item_id === blueprint.id);
      const newConsumedStatus = !currentProgress?.consumed;
      
      // Update progress
      await apiClient.updateMyBlueprintProgress(blueprint.external_id, newConsumedStatus);
      
      // Reload progress
      const progressResponse = await apiClient.getMyBlueprintProgress();
      setMyProgress(progressResponse.data);
    } catch (err) {
      alert(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  // Check if a blueprint is consumed
  const isBlueprintConsumed = (blueprintId: number): boolean => {
    const progress = myProgress.find(p => p.item_id === blueprintId);
    return progress?.consumed || false;
  };

  // Filter blueprints based on search and status
  const filteredBlueprints = blueprints.filter(blueprint => {
    // Search filter
    const matchesSearch = searchQuery === '' || 
      blueprint.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
      blueprint.external_id.toLowerCase().includes(searchQuery.toLowerCase()) ||
      blueprint.description?.toLowerCase().includes(searchQuery.toLowerCase());

    // Status filter
    const isConsumed = isBlueprintConsumed(blueprint.id);
    const matchesStatus = 
      filterStatus === 'all' ||
      (filterStatus === 'consumed' && isConsumed) ||
      (filterStatus === 'available' && !isConsumed);

    return matchesSearch && matchesStatus;
  });

  // Calculate statistics
  const totalBlueprints = blueprints.length;
  const consumedCount = blueprints.filter(b => isBlueprintConsumed(b.id)).length;
  const availableCount = totalBlueprints - consumedCount;
  const completionPercentage = totalBlueprints > 0 
    ? Math.round((consumedCount / totalBlueprints) * 100) 
    : 0;

  if (!isAuthenticated) return null;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            Blueprints
          </h1>
          <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
            Track which blueprints you've consumed/learned
          </p>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {loading ? (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
            <p className="mt-2 text-gray-600 dark:text-gray-400">Loading blueprints...</p>
          </div>
        ) : (
          <>
            {/* Statistics Cards */}
            <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
              <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
                <div className="text-sm text-gray-500 dark:text-gray-400">Total Blueprints</div>
                <div className="text-2xl font-bold text-gray-900 dark:text-white">{totalBlueprints}</div>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
                <div className="text-sm text-gray-500 dark:text-gray-400">Consumed</div>
                <div className="text-2xl font-bold text-green-600 dark:text-green-400">{consumedCount}</div>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
                <div className="text-sm text-gray-500 dark:text-gray-400">Available</div>
                <div className="text-2xl font-bold text-blue-600 dark:text-blue-400">{availableCount}</div>
              </div>
              <div className="bg-white dark:bg-gray-800 rounded-lg p-4 border border-gray-200 dark:border-gray-700">
                <div className="text-sm text-gray-500 dark:text-gray-400">Completion</div>
                <div className="text-2xl font-bold text-purple-600 dark:text-purple-400">{completionPercentage}%</div>
              </div>
            </div>

            {/* Filters */}
            <div className="mb-6 flex flex-col sm:flex-row gap-4">
              {/* Search */}
              <div className="flex-1">
                <input
                  type="text"
                  placeholder="Search blueprints..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-lg bg-white dark:bg-gray-800 text-gray-900 dark:text-white focus:ring-2 focus:ring-blue-500"
                />
              </div>

              {/* Status Filter */}
              <div className="flex gap-2">
                <button
                  onClick={() => setFilterStatus('all')}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    filterStatus === 'all'
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
                  }`}
                >
                  All ({totalBlueprints})
                </button>
                <button
                  onClick={() => setFilterStatus('consumed')}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    filterStatus === 'consumed'
                      ? 'bg-green-600 text-white'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
                  }`}
                >
                  Consumed ({consumedCount})
                </button>
                <button
                  onClick={() => setFilterStatus('available')}
                  className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                    filterStatus === 'available'
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-200 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-300 dark:hover:bg-gray-600'
                  }`}
                >
                  Available ({availableCount})
                </button>
              </div>
            </div>

            {/* Blueprints Grid */}
            {filteredBlueprints.length === 0 ? (
              <div className="text-center py-12">
                <p className="text-gray-500 dark:text-gray-400">
                  {searchQuery || filterStatus !== 'all' 
                    ? 'No blueprints match your filters' 
                    : 'No blueprints found'}
                </p>
              </div>
            ) : (
              <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                {filteredBlueprints.map(blueprint => {
                  const isConsumed = isBlueprintConsumed(blueprint.id);
                  return (
                    <div
                      key={blueprint.id}
                      className={`bg-white dark:bg-gray-800 rounded-lg border-2 transition-all ${
                        isConsumed
                          ? 'border-green-500 dark:border-green-600'
                          : 'border-gray-200 dark:border-gray-700'
                      }`}
                    >
                      {/* Image */}
                      {blueprint.image_url && (
                        <div className="aspect-video w-full bg-gray-100 dark:bg-gray-900 rounded-t-lg overflow-hidden">
                          <img
                            src={blueprint.image_url}
                            alt={blueprint.name}
                            className="w-full h-full object-cover"
                            onError={(e) => {
                              e.currentTarget.style.display = 'none';
                            }}
                          />
                        </div>
                      )}

                      {/* Content */}
                      <div className="p-4">
                        {/* Header */}
                        <div className="flex items-start justify-between mb-2">
                          <div className="flex-1">
                            <h3 className="font-semibold text-gray-900 dark:text-white">
                              {blueprint.name}
                            </h3>
                            <p className="text-xs text-gray-500 dark:text-gray-400 font-mono">
                              {blueprint.external_id}
                            </p>
                          </div>
                          
                          {/* Checkbox */}
                          <input
                            type="checkbox"
                            checked={isConsumed}
                            onChange={() => handleToggleBlueprint(blueprint)}
                            disabled={saving}
                            className="h-6 w-6 rounded border-gray-300 text-green-600 focus:ring-green-500 cursor-pointer disabled:opacity-50"
                            title={isConsumed ? 'Mark as available' : 'Mark as consumed'}
                          />
                        </div>

                        {/* Type Badge */}
                        {blueprint.type && (
                          <span className="inline-block px-2 py-1 text-xs rounded-full bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-400 mb-2">
                            {blueprint.type}
                          </span>
                        )}

                        {/* Description */}
                        {blueprint.description && (
                          <p className="text-sm text-gray-600 dark:text-gray-400 line-clamp-3">
                            {blueprint.description}
                          </p>
                        )}

                        {/* Status Badge */}
                        <div className="mt-3 pt-3 border-t border-gray-200 dark:border-gray-700">
                          <span
                            className={`inline-flex items-center px-3 py-1 rounded-full text-xs font-medium ${
                              isConsumed
                                ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                                : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
                            }`}
                          >
                            {isConsumed ? (
                              <>
                                <svg className="w-4 h-4 mr-1" fill="currentColor" viewBox="0 0 20 20">
                                  <path fillRule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clipRule="evenodd" />
                                </svg>
                                Consumed
                              </>
                            ) : (
                              'Available'
                            )}
                          </span>
                        </div>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}

            {/* Saving Indicator */}
            {saving && (
              <div className="fixed bottom-4 right-4 bg-blue-600 text-white px-4 py-2 rounded-lg shadow-lg flex items-center">
                <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                Saving...
              </div>
            )}
          </>
        )}
      </div>
    </DashboardLayout>
  );
}
