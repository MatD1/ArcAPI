'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type { Alert } from '@/types';
import DataTable from '@/components/crud/DataTable';
import EntityForm from '@/components/crud/EntityForm';

export default function AlertsPage() {
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [editing, setEditing] = useState<Alert | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadAlerts();
  }, [isAuthenticated, router, page]);

  const loadAlerts = async () => {
    try {
      setLoading(true);
      const response = await apiClient.getAlerts(page, 20);
      setAlerts(response.data);
      setTotal(response.pagination.total);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = () => {
    setEditing(null);
    setShowForm(true);
  };

  const handleEdit = (alert: Alert) => {
    setEditing(alert);
    setShowForm(true);
  };

  const handleDelete = async (id: number) => {
    if (!confirm('Are you sure you want to delete this alert?')) return;
    try {
      await apiClient.deleteAlert(id);
      loadAlerts();
    } catch (err) {
      alert(getErrorMessage(err));
    }
  };

  const handleSubmit = async (data: Partial<Alert>) => {
    try {
      if (editing) {
        await apiClient.updateAlert(editing.id, data);
      } else {
        await apiClient.createAlert(data);
      }
      setShowForm(false);
      setEditing(null);
      loadAlerts();
    } catch (err) {
      throw err;
    }
  };

  if (!isAuthenticated) return null;

  // Only admins can manage alerts
  const isAdmin = user?.role === 'admin';

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="flex justify-between items-center mb-6">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white">Alerts</h1>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              Manage system alerts that will be displayed to mobile app users
            </p>
          </div>
          {isAdmin && (
            <button
              onClick={handleCreate}
              className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700"
            >
              Create Alert
            </button>
          )}
        </div>

        {showForm && isAdmin && (
          <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
            <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">
              {editing ? 'Edit Alert' : 'Create Alert'}
            </h2>
            <EntityForm
              entity={editing}
              type="alert"
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
                data={alerts}
                onEdit={isAdmin ? handleEdit : undefined}
                onDelete={isAdmin ? handleDelete : undefined}
                type="alert"
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

