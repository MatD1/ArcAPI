'use client';

import { useState, useEffect, Suspense } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type {
  AllUserProgress,
  UserQuestProgress,
  UserHideoutModuleProgress,
  UserSkillNodeProgress,
  UserBlueprintProgress,
} from '@/types';

function ProgressContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const userIdParam = searchParams.get('userId');
  const userId = userIdParam ? Number(userIdParam) : null;
  
  const { isAuthenticated, user } = useAuthStore();
  const [progress, setProgress] = useState<AllUserProgress | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [activeTab, setActiveTab] = useState<'quests' | 'hideout' | 'skills' | 'blueprints'>('quests');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    if (user?.role !== 'admin') {
      router.push('/dashboard/');
      return;
    }
    if (!userId) {
      setError('No user ID provided');
      setLoading(false);
      return;
    }
    loadProgress();
  }, [isAuthenticated, router, user, userId]);

  const loadProgress = async () => {
    if (!userId) return;
    try {
      setLoading(true);
      const data = await apiClient.getAllUserProgress(userId);
      setProgress(data);
      setError('');
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleQuestToggle = async (questProgress: UserQuestProgress) => {
    if (!progress || !questProgress.quest || !userId) return;
    try {
      setSaving(true);
      await apiClient.updateUserQuestProgress(
        userId,
        questProgress.quest.external_id,
        !questProgress.completed
      );
      await loadProgress();
    } catch (err) {
      alert(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const handleHideoutUpdate = async (hideoutProgress: UserHideoutModuleProgress) => {
    if (!progress || !hideoutProgress.hideout_module || !userId) return;
    const unlocked = prompt(
      `Unlocked (true/false):`,
      hideoutProgress.unlocked ? 'true' : 'false'
    );
    if (unlocked === null) return;

    const level = prompt(`Level (0-${hideoutProgress.hideout_module.max_level || 10}):`, hideoutProgress.level.toString());
    if (level === null) return;

    try {
      setSaving(true);
      await apiClient.updateUserHideoutProgress(
        userId,
        hideoutProgress.hideout_module.external_id,
        unlocked === 'true',
        parseInt(level)
      );
      await loadProgress();
    } catch (err) {
      alert(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const handleSkillNodeUpdate = async (skillProgress: UserSkillNodeProgress) => {
    if (!progress || !skillProgress.skill_node || !userId) return;
    const unlocked = prompt(
      `Unlocked (true/false):`,
      skillProgress.unlocked ? 'true' : 'false'
    );
    if (unlocked === null) return;

    const level = prompt(`Level (0-${skillProgress.skill_node.max_points || 5}):`, skillProgress.level.toString());
    if (level === null) return;

    try {
      setSaving(true);
      await apiClient.updateUserSkillNodeProgress(
        userId,
        skillProgress.skill_node.external_id,
        unlocked === 'true',
        parseInt(level)
      );
      await loadProgress();
    } catch (err) {
      alert(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  const handleBlueprintToggle = async (blueprintProgress: UserBlueprintProgress) => {
    if (!progress || !blueprintProgress.item || !userId) return;
    try {
      setSaving(true);
      await apiClient.updateUserBlueprintProgress(
        userId,
        blueprintProgress.item.external_id,
        !blueprintProgress.consumed
      );
      await loadProgress();
    } catch (err) {
      alert(getErrorMessage(err));
    } finally {
      setSaving(false);
    }
  };

  if (!isAuthenticated || user?.role !== 'admin') return null;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6">
          <button
            onClick={() => router.push('/users/')}
            className="text-blue-600 dark:text-blue-400 hover:underline mb-4"
          >
            ← Back to Users
          </button>
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">
            Player Progress
          </h1>
          {progress && (
            <p className="mt-2 text-sm text-gray-600 dark:text-gray-400">
              User: {progress.user.username} ({progress.user.email})
            </p>
          )}
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {loading ? (
          <div className="text-center py-12">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
            <p className="mt-2 text-gray-600 dark:text-gray-400">Loading progress...</p>
          </div>
        ) : progress ? (
          <>
            {/* Tabs */}
            <div className="border-b border-gray-200 dark:border-gray-700 mb-6">
              <nav className="-mb-px flex space-x-8">
                {[
                  { key: 'quests' as const, label: 'Quests', count: progress.progress.quests.length },
                  { key: 'hideout' as const, label: 'Hideout', count: progress.progress.hideout_modules.length },
                  { key: 'skills' as const, label: 'Skills', count: progress.progress.skill_nodes.length },
                  { key: 'blueprints' as const, label: 'Blueprints', count: progress.progress.blueprints.length },
                ].map(tab => (
                  <button
                    key={tab.key}
                    onClick={() => setActiveTab(tab.key)}
                    className={`${
                      activeTab === tab.key
                        ? 'border-blue-500 text-blue-600 dark:text-blue-400'
                        : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300'
                    } whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm`}
                  >
                    {tab.label} ({tab.count})
                  </button>
                ))}
              </nav>
            </div>

            {/* Quests Tab */}
            {activeTab === 'quests' && (
              <div className="space-y-2">
                <div className="flex justify-between items-center mb-4">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                    Quest Progress
                  </h2>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    {progress.progress.quests.filter(q => q.completed).length} /{' '}
                    {progress.progress.quests.length} completed
                  </div>
                </div>
                {progress.progress.quests.length === 0 ? (
                  <p className="text-gray-500 dark:text-gray-400">No quest progress yet</p>
                ) : (
                  progress.progress.quests.map(quest => (
                    <div
                      key={quest.id}
                      className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
                    >
                      <div className="flex items-center space-x-3 flex-1">
                        <input
                          type="checkbox"
                          checked={quest.completed}
                          onChange={() => handleQuestToggle(quest)}
                          disabled={saving}
                          className="h-5 w-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                        />
                        <div>
                          <p className="font-medium text-gray-900 dark:text-white">
                            {quest.quest?.name || 'Unknown Quest'}
                          </p>
                          <p className="text-sm text-gray-500 dark:text-gray-400">
                            {quest.quest?.external_id}
                          </p>
                        </div>
                      </div>
                      <span
                        className={`px-2 py-1 text-xs rounded-full ${
                          quest.completed
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                            : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
                        }`}
                      >
                        {quest.completed ? 'Completed' : 'Incomplete'}
                      </span>
                    </div>
                  ))
                )}
              </div>
            )}

            {/* Hideout Tab */}
            {activeTab === 'hideout' && (
              <div className="space-y-2">
                <div className="flex justify-between items-center mb-4">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                    Hideout Progress
                  </h2>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    {progress.progress.hideout_modules.filter(h => h.unlocked).length} /{' '}
                    {progress.progress.hideout_modules.length} unlocked
                  </div>
                </div>
                {progress.progress.hideout_modules.length === 0 ? (
                  <p className="text-gray-500 dark:text-gray-400">No hideout progress yet</p>
                ) : (
                  progress.progress.hideout_modules.map(hideout => (
                    <div
                      key={hideout.id}
                      className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
                    >
                      <div className="flex-1">
                        <p className="font-medium text-gray-900 dark:text-white">
                          {hideout.hideout_module?.name || 'Unknown Module'}
                        </p>
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                          {hideout.hideout_module?.external_id}
                        </p>
                      </div>
                      <div className="flex items-center space-x-3">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Level {hideout.level}
                        </span>
                        <span
                          className={`px-2 py-1 text-xs rounded-full ${
                            hideout.unlocked
                              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                              : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
                          }`}
                        >
                          {hideout.unlocked ? 'Unlocked' : 'Locked'}
                        </span>
                        <button
                          onClick={() => handleHideoutUpdate(hideout)}
                          disabled={saving}
                          className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                        >
                          Edit
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}

            {/* Skills Tab */}
            {activeTab === 'skills' && (
              <div className="space-y-2">
                <div className="flex justify-between items-center mb-4">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                    Skill Node Progress
                  </h2>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    {progress.progress.skill_nodes.filter(s => s.unlocked).length} /{' '}
                    {progress.progress.skill_nodes.length} unlocked
                  </div>
                </div>
                {progress.progress.skill_nodes.length === 0 ? (
                  <p className="text-gray-500 dark:text-gray-400">No skill progress yet</p>
                ) : (
                  progress.progress.skill_nodes.map(skill => (
                    <div
                      key={skill.id}
                      className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
                    >
                      <div className="flex-1">
                        <p className="font-medium text-gray-900 dark:text-white">
                          {skill.skill_node?.name || 'Unknown Skill'}
                        </p>
                        <p className="text-sm text-gray-500 dark:text-gray-400">
                          {skill.skill_node?.external_id}
                        </p>
                      </div>
                      <div className="flex items-center space-x-3">
                        <span className="text-sm text-gray-600 dark:text-gray-400">
                          Level {skill.level}
                        </span>
                        <span
                          className={`px-2 py-1 text-xs rounded-full ${
                            skill.unlocked
                              ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                              : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
                          }`}
                        >
                          {skill.unlocked ? 'Unlocked' : 'Locked'}
                        </span>
                        <button
                          onClick={() => handleSkillNodeUpdate(skill)}
                          disabled={saving}
                          className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 disabled:opacity-50"
                        >
                          Edit
                        </button>
                      </div>
                    </div>
                  ))
                )}
              </div>
            )}

            {/* Blueprints Tab */}
            {activeTab === 'blueprints' && (
              <div className="space-y-2">
                <div className="flex justify-between items-center mb-4">
                  <h2 className="text-xl font-semibold text-gray-900 dark:text-white">
                    Blueprint Progress
                  </h2>
                  <div className="text-sm text-gray-600 dark:text-gray-400">
                    {progress.progress.blueprints.filter(b => b.consumed).length} /{' '}
                    {progress.progress.blueprints.length} consumed
                  </div>
                </div>
                {progress.progress.blueprints.length === 0 ? (
                  <p className="text-gray-500 dark:text-gray-400">No blueprint progress yet</p>
                ) : (
                  progress.progress.blueprints.map(blueprint => (
                    <div
                      key={blueprint.id}
                      className="flex items-center justify-between p-3 bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700"
                    >
                      <div className="flex items-center space-x-3 flex-1">
                        <input
                          type="checkbox"
                          checked={blueprint.consumed}
                          onChange={() => handleBlueprintToggle(blueprint)}
                          disabled={saving}
                          className="h-5 w-5 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                        />
                        <div>
                          <p className="font-medium text-gray-900 dark:text-white">
                            {blueprint.item?.name || 'Unknown Blueprint'}
                          </p>
                          <p className="text-sm text-gray-500 dark:text-gray-400">
                            {blueprint.item?.external_id}
                          </p>
                        </div>
                      </div>
                      <span
                        className={`px-2 py-1 text-xs rounded-full ${
                          blueprint.consumed
                            ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-400'
                            : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-400'
                        }`}
                      >
                        {blueprint.consumed ? 'Consumed' : 'Available'}
                      </span>
                    </div>
                  ))
                )}
              </div>
            )}

            {saving && (
              <div className="fixed bottom-4 right-4 bg-blue-600 text-white px-4 py-2 rounded-lg shadow-lg">
                Saving...
              </div>
            )}
          </>
        ) : (
          <div className="text-center py-12 text-gray-500 dark:text-gray-400">
            No progress data found
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

export default function UserProgressPage() {
  return (
    <Suspense fallback={
      <DashboardLayout>
        <div className="flex items-center justify-center min-h-screen">
          <div className="text-center">
            <div className="inline-block animate-spin rounded-full h-8 w-8 border-b-2 border-gray-900 dark:border-white"></div>
            <p className="mt-2 text-gray-600 dark:text-gray-400">Loading...</p>
          </div>
        </div>
      </DashboardLayout>
    }>
      <ProgressContent />
    </Suspense>
  );
}
