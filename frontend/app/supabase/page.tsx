'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import type { User as SupabaseUser } from '@supabase/supabase-js';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import {
  isSupabaseEnabledSync,
  isSupabaseEnabled,
  startSupabaseGithubLogin,
  signOutOfSupabase,
  getSupabaseClient,
} from '@/lib/supabase';

type EntityType = 'quests' | 'items' | 'skillNodes' | 'hideoutModules' | 'enemyTypes' | 'alerts';

export default function SupabasePage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [enabled, setEnabled] = useState(false);
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<{
    synced: number;
    errors: number;
    details: Record<string, { synced: number; errors: number }>;
  } | null>(null);
  const [error, setError] = useState('');
  const [selectedEntity, setSelectedEntity] = useState<EntityType | null>(null);
  const [entityData, setEntityData] = useState<any[]>([]);
  const [loadingEntity, setLoadingEntity] = useState(false);
  const [syncingCategory, setSyncingCategory] = useState<EntityType | null>(null);
  const [categorySyncResults, setCategorySyncResults] = useState<Record<EntityType, { synced: number; errors: number } | null>>({
    quests: null,
    items: null,
    skillNodes: null,
    hideoutModules: null,
    enemyTypes: null,
    alerts: null,
  });
  const [supabaseAuthLoading, setSupabaseAuthLoading] = useState(false);
  const [supabaseUser, setSupabaseUser] = useState<SupabaseUser | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    checkEnabled();
    loadCounts();
  }, [isAuthenticated, router]);

  useEffect(() => {
    let mounted = true;
    let unsubscribe: (() => void) | undefined;

    const trackSupabaseUser = async () => {
      if (!enabled) {
        setSupabaseUser(null);
        return;
      }

      const client = await getSupabaseClient();
      if (!client) {
        setSupabaseUser(null);
        return;
      }

      const { data } = await client.auth.getUser();
      if (mounted) {
        setSupabaseUser(data.user ?? null);
      }

      const { data: listener } = client.auth.onAuthStateChange((_event, session) => {
        if (mounted) {
          setSupabaseUser(session?.user ?? null);
        }
      });

      unsubscribe = () => {
        listener.subscription.unsubscribe();
      };
    };

    trackSupabaseUser();

    return () => {
      mounted = false;
      unsubscribe?.();
    };
  }, [enabled]);

  const checkEnabled = async () => {
    const enabled = await isSupabaseEnabled();
    setEnabled(enabled);
  };

  const handleSupabaseLogin = async () => {
    setError('');
    setSupabaseAuthLoading(true);
    try {
      await startSupabaseGithubLogin('/supabase');
    } catch (err) {
      setError(getErrorMessage(err));
      setSupabaseAuthLoading(false);
    }
  };

  const handleSupabaseLogout = async () => {
    setError('');
    setSupabaseAuthLoading(true);
    try {
      await signOutOfSupabase();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSupabaseAuthLoading(false);
    }
  };

  const loadCounts = async () => {
    const enabled = await isSupabaseEnabled();
    if (!enabled) {
      setLoading(false);
      return;
    }
    try {
      setLoading(true);
      const data = await apiClient.getSupabaseCounts();
      setCounts(data);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleForceSync = async () => {
    const enabled = await isSupabaseEnabled();
    if (!enabled) {
      setError('Supabase is not enabled. Please configure environment variables.');
      return;
    }

    if (!confirm('This will sync all data from the API to Supabase. This may take a while. Continue?')) {
      return;
    }

    try {
      setSyncing(true);
      setError('');
      setSyncResult(null);
      const result = await apiClient.forceSyncToSupabase();
      setSyncResult(result);
      await loadCounts(); // Refresh counts after sync
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSyncing(false);
    }
  };

  const handleSyncCategory = async (entity: EntityType, e: React.MouseEvent) => {
    e.stopPropagation(); // Prevent triggering the view data action

    const enabled = await isSupabaseEnabled();
    if (!enabled) {
      setError('Supabase is not enabled');
      return;
    }

    if (!confirm(`Sync all ${entityLabels[entity]} to Supabase?`)) {
      return;
    }

    try {
      setSyncingCategory(entity);
      setError('');
      let result: { synced: number; errors: number };

      switch (entity) {
        case 'quests':
          result = await apiClient.syncQuestsToSupabase();
          break;
        case 'items':
          result = await apiClient.syncItemsToSupabase();
          break;
        case 'skillNodes':
          result = await apiClient.syncSkillNodesToSupabase();
          break;
        case 'hideoutModules':
          result = await apiClient.syncHideoutModulesToSupabase();
          break;
        case 'enemyTypes':
          result = await apiClient.syncEnemyTypesToSupabase();
          break;
        case 'alerts':
          result = await apiClient.syncAlertsToSupabase();
          break;
        default:
          throw new Error('Unknown entity type');
      }

      setCategorySyncResults((prev) => ({
        ...prev,
        [entity]: result,
      }));
      await loadCounts(); // Refresh counts after sync
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSyncingCategory(null);
    }
  };

  const loadEntityData = async (entity: EntityType) => {
    const enabled = await isSupabaseEnabled();
    if (!enabled) {
      setError('Supabase is not enabled');
      return;
    }

    try {
      setLoadingEntity(true);
      setError('');
      let data: any[] = [];

      switch (entity) {
        case 'quests':
          data = await apiClient.getSupabaseQuests(100);
          break;
        case 'items':
          data = await apiClient.getSupabaseItems(100);
          break;
        case 'skillNodes':
          data = await apiClient.getSupabaseSkillNodes(100);
          break;
        case 'hideoutModules':
          data = await apiClient.getSupabaseHideoutModules(100);
          break;
        case 'enemyTypes':
          data = await apiClient.getSupabaseEnemyTypes(100);
          break;
        case 'alerts':
          data = await apiClient.getSupabaseAlerts(100);
          break;
      }

      setEntityData(data);
      setSelectedEntity(entity);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoadingEntity(false);
    }
  };

  if (!isAuthenticated) return null;

  const entityLabels: Record<EntityType, string> = {
    quests: 'Quests',
    items: 'Items',
    skillNodes: 'Skill Nodes',
    hideoutModules: 'Hideout Modules',
    enemyTypes: 'Enemy Types',
    alerts: 'Alerts',
  };

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Supabase Management</h1>
          <p className="text-gray-600 dark:text-gray-400">
            View and manage data synced to Supabase database
          </p>
        </div>

        {enabled && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">Supabase Access Control</h2>
            <p className="text-sm text-gray-600 dark:text-gray-400">
              Sign in with Supabase (GitHub) to access your Supabase project or Studio using the same browser session.
            </p>
            <div className="mt-4">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">Status:</span>
              <span className="ml-2 text-gray-900 dark:text-white">
                {supabaseUser ? `Signed in as ${supabaseUser.email || supabaseUser.user_metadata?.user_name || supabaseUser.id}` : 'Not signed in'}
              </span>
            </div>
            <div className="mt-4 flex flex-wrap gap-3">
              {supabaseUser ? (
                <button
                  onClick={handleSupabaseLogout}
                  disabled={supabaseAuthLoading}
                  className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-md hover:bg-gray-300 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {supabaseAuthLoading ? 'Signing out...' : 'Sign out of Supabase'}
                </button>
              ) : (
                <button
                  onClick={handleSupabaseLogin}
                  disabled={supabaseAuthLoading}
                  className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {supabaseAuthLoading ? 'Redirecting...' : 'Sign in with Supabase (GitHub)'}
                </button>
              )}
              {process.env.NEXT_PUBLIC_SUPABASE_URL && (
                <a
                  href={process.env.NEXT_PUBLIC_SUPABASE_URL}
                  target="_blank"
                  rel="noreferrer"
                  className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700"
                >
                  Open Supabase URL
                </a>
              )}
            </div>
          </div>
        )}

        {!enabled && (
          <div className="mb-6 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
            <p className="text-yellow-800 dark:text-yellow-200 mb-2">
              Supabase is not enabled. Set <code className="bg-yellow-100 dark:bg-yellow-900 px-1 rounded">NEXT_PUBLIC_SUPABASE_ENABLED=true</code> in your environment variables.
            </p>
            <details className="mt-2">
              <summary className="cursor-pointer text-sm text-yellow-700 dark:text-yellow-300">Debug: Check environment variables</summary>
              <div className="mt-2 p-2 bg-yellow-100 dark:bg-yellow-900 rounded text-xs font-mono">
                <div>NEXT_PUBLIC_SUPABASE_ENABLED: {process.env.NEXT_PUBLIC_SUPABASE_ENABLED || '(not set)'}</div>
                <div>NEXT_PUBLIC_SUPABASE_URL: {process.env.NEXT_PUBLIC_SUPABASE_URL ? '✓ Set' : '✗ Not set'}</div>
                <div>NEXT_PUBLIC_SUPABASE_ANON_KEY: {process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY ? '✓ Set' : '✗ Not set'}</div>
                <div className="mt-2 text-yellow-600 dark:text-yellow-400">
                  Note: For static exports, env vars must be set at build time and included in next.config.js
                </div>
              </div>
            </details>
          </div>
        )}

        {error && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {/* Force Sync Section */}
        <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Force Sync</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Sync all data from the API to Supabase. This will update or insert all entities.
          </p>
          <button
            onClick={handleForceSync}
            disabled={!enabled || syncing}
            className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {syncing ? 'Syncing...' : 'Force Sync All Data'}
          </button>

          {syncResult && (
            <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
              <h3 className="font-semibold text-gray-900 dark:text-white mb-2">Sync Results</h3>
              <div className="grid grid-cols-2 gap-4 mb-4">
                <div>
                  <span className="text-sm text-gray-600 dark:text-gray-400">Total Synced:</span>
                  <span className="ml-2 font-semibold text-green-600 dark:text-green-400">{syncResult.synced}</span>
                </div>
                <div>
                  <span className="text-sm text-gray-600 dark:text-gray-400">Errors:</span>
                  <span className="ml-2 font-semibold text-red-600 dark:text-red-400">{syncResult.errors}</span>
                </div>
              </div>
              <div className="space-y-2">
                {Object.entries(syncResult.details).map(([entity, stats]) => (
                  <div key={entity} className="flex justify-between text-sm">
                    <span className="text-gray-700 dark:text-gray-300 capitalize">{entity}:</span>
                    <span className="text-gray-900 dark:text-white">
                      {stats.synced} synced, {stats.errors} errors
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>

        {/* Data Counts Section */}
        {enabled && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">Data in Supabase</h2>
              <button
                onClick={loadCounts}
                disabled={loading}
                className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50"
              >
                {loading ? 'Loading...' : 'Refresh'}
              </button>
            </div>
            <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
              {(Object.keys(entityLabels) as EntityType[]).map((entity) => {
                const isSyncing = syncingCategory === entity;
                const syncResult = categorySyncResults[entity];
                return (
                  <div
                    key={entity}
                    className="p-4 border border-gray-200 dark:border-gray-700 rounded-lg hover:shadow-md transition-shadow"
                  >
                    <div className="flex justify-between items-start mb-2">
                      <div
                        className="flex-1 cursor-pointer"
                        onClick={() => loadEntityData(entity)}
                      >
                        <div className="text-sm text-gray-600 dark:text-gray-400 mb-1">{entityLabels[entity]}</div>
                        <div className="text-2xl font-bold text-gray-900 dark:text-white">
                          {loading ? '...' : counts[entity] ?? 0}
                        </div>
                      </div>
                      <button
                        onClick={(e) => handleSyncCategory(entity, e)}
                        disabled={isSyncing || syncing}
                        className="ml-2 px-2 py-1 text-xs bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        title={`Sync ${entityLabels[entity]}`}
                      >
                        {isSyncing ? '...' : '🔄'}
                      </button>
                    </div>
                    {syncResult && (
                      <div className="mt-2 text-xs text-gray-600 dark:text-gray-400">
                        <span className="text-green-600 dark:text-green-400">{syncResult.synced} synced</span>
                        {syncResult.errors > 0 && (
                          <span className="ml-2 text-red-600 dark:text-red-400">{syncResult.errors} errors</span>
                        )}
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        )}

        {/* Entity Data View */}
        {selectedEntity && (
          <div className="p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                {entityLabels[selectedEntity]} ({entityData.length})
              </h2>
              <button
                onClick={() => setSelectedEntity(null)}
                className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
              >
                Close
              </button>
            </div>

            {loadingEntity ? (
              <div className="text-center py-8">Loading...</div>
            ) : entityData.length === 0 ? (
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">No data found</div>
            ) : (
              <div className="overflow-x-auto">
                <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
                  <thead className="bg-gray-50 dark:bg-gray-700">
                    <tr>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        ID
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Name
                      </th>
                      <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-300 uppercase tracking-wider">
                        Created
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-800 divide-y divide-gray-200 dark:divide-gray-700">
                    {entityData.map((item, idx) => (
                      <tr key={idx} className="hover:bg-gray-50 dark:hover:bg-gray-700">
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                          {item.external_id || item.api_id || item.id || '-'}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">{item.name || '-'}</td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {item.created_at
                            ? new Date(item.created_at).toLocaleDateString()
                            : '-'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

