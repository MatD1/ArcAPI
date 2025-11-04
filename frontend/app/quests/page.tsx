'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type { Quest } from '@/types';
import DataTable from '@/components/crud/DataTable';
import EntityForm from '@/components/crud/EntityForm';

export default function QuestsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [quests, setQuests] = useState<Quest[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState<Quest | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [syncing, setSyncing] = useState(false);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadQuests();
  }, [isAuthenticated, router, page]);

  const loadQuests = async () => {
    try {
      setLoading(true);
      const response = await apiClient.getQuests(page, 20);
      setQuests(response.data);
      setTotal(response.pagination.total);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleForceSync = async () => {
    if (!confirm('This will force a sync from GitHub. Continue?')) return;
    try {
      setSyncing(true);
      await apiClient.forceSync();
      alert('Sync triggered successfully. It may take a few moments to complete.');
      // Wait a bit then reload
      setTimeout(() => {
        loadQuests();
      }, 3000);
    } catch (err) {
      alert(getErrorMessage(err));
    } finally {
      setSyncing(false);
    }
  };

  const handleCreate = () => {
    setEditing(null);
    setShowForm(true);
  };

  const handleEdit = (quest: Quest) => {
    setEditing(quest);
    setShowForm(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this quest?')) return;
    try {
      await apiClient.deleteQuest(id);
      loadQuests();
    } catch (err) {
      alert(getErrorMessage(err));
    }
  };

  const handleSubmit = async (data: Partial<Quest>) => {
    try {
      if (editing) {
        await apiClient.updateQuest(editing.id, data);
      } else {
        await apiClient.createQuest(data);
      }
      setShowForm(false);
      setEditing(null);
      loadQuests();
    } catch (err) {
      throw err;
    }
  };

  if (!isAuthenticated) return null;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Quests</h1>
          <div className="flex gap-2">
            <button
              onClick={handleForceSync}
              disabled={syncing}
              className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700 disabled:opacity-50"
            >
              {syncing ? 'Syncing...' : 'Force Sync'}
            </button>
            <button
              onClick={handleCreate}
              className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700"
            >
              Create Quest
            </button>
          </div>
        </div>

        {showForm && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
              {editing ? 'Edit Quest' : 'Create Quest'}
            </h2>
            <EntityForm
              entity={editing}
              type="quest"
              onSubmit={handleSubmit}
              onCancel={() => {
                setShowForm(false);
                setEditing(null);
              }}
            />
          </div>
        )}

        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {loading ? (
          <div className="text-center py-8">Loading...</div>
        ) : (
          <>
            <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
              <DataTable
                data={quests}
                onEdit={handleEdit}
                onDelete={handleDelete}
                type="quest"
              />
            </div>
            <div className="mt-4 flex justify-between">
              <button
                onClick={() => setPage(Math.max(1, page - 1))}
                disabled={page === 1}
                className="px-4 py-2 border rounded-md disabled:opacity-50"
              >
                Previous
              </button>
              <span className="py-2 text-gray-700 dark:text-gray-300">
                Page {page} of {Math.ceil(total / 20)}
              </span>
              <button
                onClick={() => setPage(page + 1)}
                disabled={page >= Math.ceil(total / 20)}
                className="px-4 py-2 border rounded-md disabled:opacity-50"
              >
                Next
              </button>
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}
