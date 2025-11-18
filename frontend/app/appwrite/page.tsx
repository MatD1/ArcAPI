"use client";

import { useState, useEffect } from "react";
import { useRouter } from "next/navigation";
import DashboardLayout from "@/components/layout/DashboardLayout";
import { useAuthStore } from "@/store/authStore";
import { apiClient, getErrorMessage } from "@/lib/api";
import {
  isAppwriteEnabled,
  signOutOfAppwrite,
  getAppwriteSession,
  loginWithGitHub,
  loginWithDiscord,
} from "@/lib/appwrite";

type EntityType =
  | "quests"
  | "items"
  | "skillNodes"
  | "hideoutModules"
  | "enemyTypes"
  | "alerts";

const entityLabels: Record<EntityType, string> = {
  quests: "Quests",
  items: "Items",
  skillNodes: "Skill Nodes",
  hideoutModules: "Hideout Modules",
  enemyTypes: "Enemy Types",
  alerts: "Alerts",
};

export default function AppwritePage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();

  // Appwrite state
  const [enabled, setEnabled] = useState(false);
  const [appwriteUser, setAppwriteUser] = useState<any | null>(null);
  const [appwriteAuthLoading, setAppwriteAuthLoading] = useState(false);

  // Data state
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  // Sync state
  const [syncing, setSyncing] = useState(false);
  const [syncResult, setSyncResult] = useState<{
    synced: number;
    errors: number;
    details: Record<string, { synced: number; errors: number }>;
  } | null>(null);
  const [syncingCategory, setSyncingCategory] = useState<EntityType | null>(
    null
  );
  const [categorySyncResults, setCategorySyncResults] = useState<
    Record<EntityType, { synced: number; errors: number } | null>
  >({
    quests: null,
    items: null,
    skillNodes: null,
    hideoutModules: null,
    enemyTypes: null,
    alerts: null,
  });

  // Entity view state
  const [selectedEntity, setSelectedEntity] = useState<EntityType | null>(null);
  const [entityData, setEntityData] = useState<any[]>([]);
  const [loadingEntity, setLoadingEntity] = useState(false);

  // Check authentication and redirect if needed
  useEffect(() => {
    if (!isAuthenticated) {
      router.push("/login/");
      return;
    }
  }, [isAuthenticated, router]);

  // Check if Appwrite is enabled
  useEffect(() => {
    const checkEnabled = async () => {
      const isEnabled = await isAppwriteEnabled();
      setEnabled(isEnabled);
    };
    checkEnabled();
  }, []);

  // Handle OAuth callbacks
  useEffect(() => {
    if (!isAuthenticated || !enabled) return;

    const params = new URLSearchParams(window.location.search);
    const oauthStatus = params.get("oauth");

    if (oauthStatus === "success") {
      window.history.replaceState({}, "", window.location.pathname);
      setTimeout(async () => {
        try {
          const session = await getAppwriteSession();
          if (session) {
            setAppwriteUser(session);
            await loadCounts();
          }
        } catch (err) {
          console.error("Failed to get Appwrite session after OAuth:", err);
        }
      }, 500);
    } else if (oauthStatus === "failure") {
      setError("OAuth authentication failed. Please try again.");
      window.history.replaceState({}, "", window.location.pathname);
    }
  }, [isAuthenticated, enabled]);

  // Track Appwrite user session
  useEffect(() => {
    if (!enabled) {
      setAppwriteUser(null);
      return;
    }

    let mounted = true;
    const trackAppwriteUser = async () => {
      try {
        const session = await getAppwriteSession();
        if (mounted) {
          setAppwriteUser(session);
        }
      } catch (err) {
        if (mounted) {
          setAppwriteUser(null);
        }
      }
    };

    trackAppwriteUser();
    const interval = setInterval(trackAppwriteUser, 5000);

    return () => {
      mounted = false;
      clearInterval(interval);
    };
  }, [enabled]);

  // Load counts when enabled and authenticated
  useEffect(() => {
    if (!isAuthenticated || !enabled) {
      setLoading(false);
      return;
    }
    loadCounts();
  }, [isAuthenticated, enabled]);

  const loadCounts = async () => {
    if (!enabled) {
      setLoading(false);
      return;
    }
    try {
      setLoading(true);
      setError("");
      const data = await apiClient.getAppwriteCounts();
      setCounts(data);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleAppwriteLogout = async () => {
    setError("");
    setAppwriteAuthLoading(true);
    try {
      await signOutOfAppwrite();
      setAppwriteUser(null);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setAppwriteAuthLoading(false);
    }
  };

  const handleGitHubLogin = async () => {
    setError("");
    setAppwriteAuthLoading(true);
    try {
      await loginWithGitHub();
    } catch (err) {
      setError(getErrorMessage(err));
      setAppwriteAuthLoading(false);
    }
  };

  const handleDiscordLogin = async () => {
    setError("");
    setAppwriteAuthLoading(true);
    try {
      await loginWithDiscord();
    } catch (err) {
      setError(getErrorMessage(err));
      setAppwriteAuthLoading(false);
    }
  };

  const handleForceSync = async () => {
    if (!enabled) {
      setError(
        "Appwrite is not enabled. Please configure environment variables."
      );
      return;
    }

    if (
      !confirm(
        "This will sync all data from the API to Appwrite. This may take a while. Continue?"
      )
    ) {
      return;
    }

    try {
      setSyncing(true);
      setError("");
      setSyncResult(null);
      const result = await apiClient.forceSyncToAppwrite();
      setSyncResult(result);
      await loadCounts();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSyncing(false);
    }
  };

  const handleSyncCategory = async (
    entity: EntityType,
    e: React.MouseEvent
  ) => {
    e.stopPropagation();

    if (!enabled) {
      setError("Appwrite is not enabled");
      return;
    }

    if (!confirm(`Sync all ${entityLabels[entity]} to Appwrite?`)) {
      return;
    }

    try {
      setSyncingCategory(entity);
      setError("");
      let result: { synced: number; errors: number };

      switch (entity) {
        case "quests":
          result = await apiClient.syncQuestsToAppwrite();
          break;
        case "items":
          result = await apiClient.syncItemsToAppwrite();
          break;
        case "skillNodes":
          result = await apiClient.syncSkillNodesToAppwrite();
          break;
        case "hideoutModules":
          result = await apiClient.syncHideoutModulesToAppwrite();
          break;
        case "enemyTypes":
          result = await apiClient.syncEnemyTypesToAppwrite();
          break;
        case "alerts":
          result = await apiClient.syncAlertsToAppwrite();
          break;
        default:
          throw new Error("Unknown entity type");
      }

      setCategorySyncResults((prev) => ({
        ...prev,
        [entity]: result,
      }));
      await loadCounts();
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setSyncingCategory(null);
    }
  };

  const loadEntityData = async (entity: EntityType) => {
    if (!enabled) {
      setError("Appwrite is not enabled");
      return;
    }

    try {
      setLoadingEntity(true);
      setError("");
      let data: any[] = [];

      switch (entity) {
        case "quests":
          data = await apiClient.getAppwriteQuests(100);
          break;
        case "items":
          data = await apiClient.getAppwriteItems(100);
          break;
        case "skillNodes":
          data = await apiClient.getAppwriteSkillNodes(100);
          break;
        case "hideoutModules":
          data = await apiClient.getAppwriteHideoutModules(100);
          break;
        case "enemyTypes":
          data = await apiClient.getAppwriteEnemyTypes(100);
          break;
        case "alerts":
          data = await apiClient.getAppwriteAlerts(100);
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

  if (!isAuthenticated) {
    return null;
  }

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        {/* Header */}
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">
            Appwrite Management
          </h1>
          <p className="text-gray-600 dark:text-gray-400">
            View and manage data synced to Appwrite database
          </p>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {/* Not Enabled Warning */}
        {!enabled && (
          <div className="mb-6 p-4 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg">
            <p className="text-yellow-800 dark:text-yellow-200">
              Appwrite is not enabled. Please configure environment variables.
            </p>
          </div>
        )}

        {/* Appwrite Authentication */}
        {enabled && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-2">
              Appwrite Access Control
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Manage your Appwrite session and access your Appwrite project.
            </p>

            <div className="mb-4">
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                Status:{" "}
              </span>
              <span className="text-sm text-gray-900 dark:text-white">
                {appwriteUser
                  ? `Signed in as ${
                      appwriteUser.email ||
                      appwriteUser.name ||
                      appwriteUser.$id
                    }`
                  : "Not signed in"}
              </span>
            </div>

            {appwriteUser ? (
              <div className="space-y-3">
                <button
                  onClick={handleAppwriteLogout}
                  disabled={appwriteAuthLoading}
                  className="px-4 py-2 bg-gray-200 dark:bg-gray-700 text-gray-900 dark:text-white rounded-md hover:bg-gray-300 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {appwriteAuthLoading
                    ? "Signing out..."
                    : "Sign out of Appwrite"}
                </button>
                {process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT && (
                  <div>
                    <a
                      href={process.env.NEXT_PUBLIC_APPWRITE_ENDPOINT}
                      target="_blank"
                      rel="noreferrer"
                      className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md text-sm font-medium text-gray-700 dark:text-gray-200 hover:bg-gray-50 dark:hover:bg-gray-700 inline-block"
                    >
                      Open Appwrite Console
                    </a>
                  </div>
                )}
              </div>
            ) : (
              <div className="space-y-3">
                <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">
                  Appwrite OAuth
                </h3>
                <div className="flex flex-wrap gap-3">
                  <button
                    onClick={handleGitHubLogin}
                    disabled={appwriteAuthLoading}
                    className="px-4 py-2 bg-gray-800 hover:bg-gray-700 text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                  >
                    <svg
                      className="w-5 h-5"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fillRule="evenodd"
                        d="M10 0C4.477 0 0 4.484 0 10.017c0 4.425 2.865 8.18 6.839 9.504.5.092.682-.217.682-.483 0-.237-.008-.868-.013-1.703-2.782.605-3.369-1.343-3.369-1.343-.454-1.158-1.11-1.466-1.11-1.466-.908-.62.069-.608.069-.608 1.003.07 1.531 1.032 1.531 1.032.892 1.53 2.341 1.088 2.91.832.092-.647.35-1.088.636-1.338-2.22-.253-4.555-1.113-4.555-4.951 0-1.093.39-1.988 1.029-2.688-.103-.253-.446-1.272.098-2.65 0 0 .84-.27 2.75 1.026A9.564 9.564 0 0110 4.844c.85.004 1.705.115 2.504.337 1.909-1.296 2.747-1.027 2.747-1.027.546 1.379.203 2.398.1 2.651.64.7 1.028 1.595 1.028 2.688 0 3.848-2.339 4.695-4.566 4.942.359.31.678.921.678 1.856 0 1.338-.012 2.419-.012 2.747 0 .268.18.58.688.482A10.019 10.019 0 0020 10.017C20 4.484 15.522 0 10 0z"
                        clipRule="evenodd"
                      />
                    </svg>
                    {appwriteAuthLoading
                      ? "Connecting..."
                      : "Sign in with GitHub"}
                  </button>
                  <button
                    onClick={handleDiscordLogin}
                    disabled={appwriteAuthLoading}
                    className="px-4 py-2 bg-[#5865F2] hover:bg-[#4752C4] text-white rounded-md text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-2"
                  >
                    <svg
                      className="w-5 h-5"
                      fill="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path d="M20.317 4.37a19.791 19.791 0 0 0-4.885-1.515.074.074 0 0 0-.079.037c-.21.375-.444.864-.608 1.25a18.27 18.27 0 0 0-5.487 0 12.64 12.64 0 0 0-.617-1.25.077.077 0 0 0-.079-.037A19.736 19.736 0 0 0 3.677 4.37a.07.07 0 0 0-.032.027C.533 9.046-.32 13.58.099 18.057a.082.082 0 0 0 .031.057 19.9 19.9 0 0 0 5.993 3.03.078.078 0 0 0 .084-.028c.462-.63.874-1.295 1.226-1.994a.076.076 0 0 0-.041-.106 13.107 13.107 0 0 1-1.872-.892.077.077 0 0 1-.008-.128 10.2 10.2 0 0 0 .372-.292.074.074 0 0 1 .077-.01c3.928 1.793 8.18 1.793 12.062 0a.074.074 0 0 1 .078.01c.12.098.246.198.373.292a.077.077 0 0 1-.006.127 12.299 12.299 0 0 1-1.873.892.077.077 0 0 0-.041.107c.36.698.772 1.362 1.225 1.993a.076.076 0 0 0 .084.028 19.839 19.839 0 0 0 6.002-3.03.077.077 0 0 0 .032-.054c.5-5.177-.838-9.674-3.549-13.66a.061.061 0 0 0-.031-.03zM8.02 15.33c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.956-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.956 2.418-2.157 2.418zm7.975 0c-1.183 0-2.157-1.085-2.157-2.419 0-1.333.955-2.419 2.157-2.419 1.21 0 2.176 1.096 2.157 2.42 0 1.333-.946 2.418-2.157 2.418z" />
                    </svg>
                    {appwriteAuthLoading
                      ? "Connecting..."
                      : "Sign in with Discord"}
                  </button>
                </div>
              </div>
            )}
          </div>
        )}

        {/* Force Sync Section */}
        {enabled && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">
              Force Sync
            </h2>
            <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
              Sync all data from the API to Appwrite. This will update or insert
              all entities.
            </p>
            <button
              onClick={handleForceSync}
              disabled={syncing}
              className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {syncing ? "Syncing..." : "Force Sync All Data"}
            </button>

            {syncResult && (
              <div className="mt-4 p-4 bg-gray-50 dark:bg-gray-700 rounded-lg">
                <h3 className="font-semibold text-gray-900 dark:text-white mb-2">
                  Sync Results
                </h3>
                <div className="grid grid-cols-2 gap-4 mb-4">
                  <div>
                    <span className="text-sm text-gray-600 dark:text-gray-400">
                      Total Synced:{" "}
                    </span>
                    <span className="font-semibold text-green-600 dark:text-green-400">
                      {syncResult.synced}
                    </span>
                  </div>
                  <div>
                    <span className="text-sm text-gray-600 dark:text-gray-400">
                      Errors:{" "}
                    </span>
                    <span className="font-semibold text-red-600 dark:text-red-400">
                      {syncResult.errors}
                    </span>
                  </div>
                </div>
                <div className="space-y-2">
                  {Object.entries(syncResult.details).map(([entity, stats]) => (
                    <div key={entity} className="flex justify-between text-sm">
                      <span className="text-gray-700 dark:text-gray-300 capitalize">
                        {entity}:
                      </span>
                      <span className="text-gray-900 dark:text-white">
                        {stats.synced} synced, {stats.errors} errors
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}

        {/* Data Counts Section */}
        {enabled && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <div className="flex justify-between items-center mb-4">
              <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                Data in Appwrite
              </h2>
              <button
                onClick={loadCounts}
                disabled={loading}
                className="px-3 py-1 text-sm bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50"
              >
                {loading ? "Loading..." : "Refresh"}
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
                        <div className="text-sm text-gray-600 dark:text-gray-400 mb-1">
                          {entityLabels[entity]}
                        </div>
                        <div className="text-2xl font-bold text-gray-900 dark:text-white">
                          {loading ? "..." : counts[entity] ?? 0}
                        </div>
                      </div>
                      <button
                        onClick={(e) => handleSyncCategory(entity, e)}
                        disabled={isSyncing || syncing}
                        className="ml-2 px-2 py-1 text-xs bg-indigo-600 text-white rounded hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        title={`Sync ${entityLabels[entity]}`}
                      >
                        {isSyncing ? "..." : "🔄"}
                      </button>
                    </div>
                    {syncResult && (
                      <div className="mt-2 text-xs text-gray-600 dark:text-gray-400">
                        <span className="text-green-600 dark:text-green-400">
                          {syncResult.synced} synced
                        </span>
                        {syncResult.errors > 0 && (
                          <span className="ml-2 text-red-600 dark:text-red-400">
                            {syncResult.errors} errors
                          </span>
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
              <div className="text-center py-8 text-gray-500 dark:text-gray-400">
                No data found
              </div>
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
                      <tr
                        key={idx}
                        className="hover:bg-gray-50 dark:hover:bg-gray-700"
                      >
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                          {item.external_id || item.api_id || item.$id || "-"}
                        </td>
                        <td className="px-4 py-3 text-sm text-gray-900 dark:text-white">
                          {item.name || "-"}
                        </td>
                        <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {item.$createdAt
                            ? new Date(item.$createdAt).toLocaleDateString()
                            : item.created_at
                            ? new Date(item.created_at).toLocaleDateString()
                            : "-"}
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
