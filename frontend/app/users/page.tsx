'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type { User, APIKey, JWTToken } from '@/types';
import { formatDate } from '@/lib/utils';

export default function UsersPage() {
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [selectedUser, setSelectedUser] = useState<{
    user: User;
    apiKeys: APIKey[];
    jwtTokens: JWTToken[];
  } | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    if (user?.role !== 'admin') {
      router.push('/dashboard/');
      return;
    }
    loadUsers();
  }, [isAuthenticated, router, user, page]);

  const loadUsers = async () => {
    try {
      setLoading(true);
      const response = await apiClient.getUsers(page, 50);
      setUsers(response.data);
      setTotal(response.pagination.total);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const loadUserDetails = async (userId: number) => {
    try {
      const data = await apiClient.getUser(userId);
      setSelectedUser({
        user: data.user,
        apiKeys: data.api_keys,
        jwtTokens: data.jwt_tokens,
      });
    } catch (err) {
      alert(getErrorMessage(err));
    }
  };

  const handleToggleAccess = async (userId: number, currentAccess: boolean) => {
    if (!confirm(`Are you sure you want to ${currentAccess ? 'revoke' : 'grant'} access for this user?`)) return;
    try {
      await apiClient.updateUserAccess(userId, !currentAccess);
      loadUsers();
      if (selectedUser && selectedUser.user.id === userId) {
        loadUserDetails(userId);
      }
    } catch (err) {
      alert(getErrorMessage(err));
    }
  };

  const handleDeleteUser = async (userId: number, username: string) => {
    if (!confirm(`Are you sure you want to delete user "${username}"? This action cannot be undone and will delete all associated API keys and tokens.`)) return;
    try {
      await apiClient.deleteUser(userId);
      if (selectedUser && selectedUser.user.id === userId) {
        setSelectedUser(null);
      }
      loadUsers();
    } catch (err) {
      alert(getErrorMessage(err));
    }
  };

  if (!isAuthenticated || user?.role !== 'admin') return null;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="flex justify-between items-center mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white">User Management</h1>
        </div>

        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Users List */}
          <div>
            <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">Users</h2>
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
                            Username
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Email
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Role
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Access
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Via App
                          </th>
                          <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Created
                          </th>
                          <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                            Actions
                          </th>
                        </tr>
                      </thead>
                      <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                        {users.map((u) => (
                          <tr
                            key={u.id}
                            onClick={() => loadUserDetails(u.id)}
                            className={`cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors ${
                              selectedUser?.user.id === u.id ? 'bg-indigo-50 dark:bg-indigo-900/20' : ''
                            }`}
                          >
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                              {u.id}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                              {u.username}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {u.email}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <span
                                className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                  u.role === 'admin'
                                    ? 'bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-200'
                                    : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                }`}
                              >
                                {u.role}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <span
                                className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                  u.can_access_data
                                    ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-200'
                                    : 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-200'
                                }`}
                              >
                                {u.can_access_data ? 'Enabled' : 'Disabled'}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap">
                              <span
                                className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                                  u.created_via_app
                                    ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-200'
                                    : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                                }`}
                              >
                                {u.created_via_app ? 'Yes' : 'No'}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {formatDate(u.created_at)}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium" onClick={(e) => e.stopPropagation()}>
                              <button
                                onClick={() => router.push(`/progress/?userId=${u.id}`)}
                                className="px-3 py-1 text-xs rounded-md mr-2 bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900/20 dark:text-blue-200"
                              >
                                View Progress
                              </button>
                              <button
                                onClick={() => handleToggleAccess(u.id, u.can_access_data)}
                                className={`px-3 py-1 text-xs rounded-md mr-2 ${
                                  u.can_access_data
                                    ? 'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-900/20 dark:text-red-200'
                                    : 'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900/20 dark:text-green-200'
                                }`}
                              >
                                {u.can_access_data ? 'Revoke' : 'Grant'}
                              </button>
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
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
                    Page {page} of {Math.ceil(total / 50)}
                  </span>
                  <button
                    onClick={() => setPage(page + 1)}
                    disabled={page >= Math.ceil(total / 50)}
                    className="px-4 py-2 border rounded-md disabled:opacity-50"
                  >
                    Next
                  </button>
                </div>
              </>
            )}
          </div>

          {/* User Details */}
          <div>
            <h2 className="text-xl font-semibold mb-4 text-gray-900 dark:text-white">User Details</h2>
            {selectedUser ? (
              <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6 space-y-6">
                {/* User Info */}
                <div>
                  <div className="flex justify-between items-center mb-4">
                    <h3 className="text-lg font-medium text-gray-900 dark:text-white">User Information</h3>
                    <div className="space-x-2">
                      <button
                        onClick={() => router.push(`/progress/?userId=${selectedUser.user.id}`)}
                        className="px-3 py-1 text-xs bg-blue-600 text-white rounded-md hover:bg-blue-700"
                      >
                        View Progress
                      </button>
                      <button
                        onClick={() => handleDeleteUser(selectedUser.user.id, selectedUser.user.username)}
                        disabled={selectedUser.user.id === user?.id}
                        className="px-3 py-1 text-xs bg-red-600 text-white rounded-md hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
                        title={selectedUser.user.id === user?.id ? 'Cannot delete your own account' : 'Delete user'}
                      >
                        Delete User
                      </button>
                    </div>
                  </div>
                  <dl className="space-y-2">
                    <div className="flex justify-between">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">ID</dt>
                      <dd className="text-sm text-gray-900 dark:text-white">{selectedUser.user.id}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Username</dt>
                      <dd className="text-sm text-gray-900 dark:text-white">{selectedUser.user.username}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Email</dt>
                      <dd className="text-sm text-gray-900 dark:text-white">{selectedUser.user.email}</dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Role</dt>
                      <dd className="text-sm text-gray-900 dark:text-white">{selectedUser.user.role}</dd>
                    </div>
                    <div className="flex justify-between items-center">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Data Access</dt>
                      <dd className="text-sm">
                        <button
                          onClick={() => handleToggleAccess(selectedUser.user.id, selectedUser.user.can_access_data)}
                          className={`px-3 py-1 rounded-md text-xs font-medium transition-colors ${
                            selectedUser.user.can_access_data
                              ? 'bg-red-100 text-red-800 hover:bg-red-200 dark:bg-red-900/20 dark:text-red-200'
                              : 'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900/20 dark:text-green-200'
                          }`}
                        >
                          {selectedUser.user.can_access_data ? 'Disable Access' : 'Enable Access'}
                        </button>
                      </dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Created Via App</dt>
                      <dd className="text-sm">
                        <span
                          className={`px-2 py-1 rounded-full text-xs font-medium ${
                            selectedUser.user.created_via_app
                              ? 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-200'
                              : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                          }`}
                        >
                          {selectedUser.user.created_via_app ? 'Yes' : 'No'}
                        </span>
                      </dd>
                    </div>
                    <div className="flex justify-between">
                      <dt className="text-sm font-medium text-gray-500 dark:text-gray-400">Created</dt>
                      <dd className="text-sm text-gray-900 dark:text-white">{formatDate(selectedUser.user.created_at)}</dd>
                    </div>
                  </dl>
                </div>

                {/* API Keys */}
                <div>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                    API Keys ({selectedUser.apiKeys.length})
                  </h3>
                  {selectedUser.apiKeys.length > 0 ? (
                    <div className="space-y-2 max-h-48 overflow-y-auto">
                      {selectedUser.apiKeys.map((key) => (
                        <div
                          key={key.id}
                          className="p-3 bg-gray-50 dark:bg-gray-700 rounded-md border border-gray-200 dark:border-gray-600"
                        >
                          <div className="flex justify-between items-start">
                            <div>
                              <p className="text-sm font-medium text-gray-900 dark:text-white">{key.name}</p>
                              <p className="text-xs text-gray-500 dark:text-gray-400">
                                Created: {formatDate(key.created_at)}
                              </p>
                              {key.last_used_at && (
                                <p className="text-xs text-gray-500 dark:text-gray-400">
                                  Last used: {formatDate(key.last_used_at)}
                                </p>
                              )}
                            </div>
                            <span
                              className={`px-2 py-1 rounded-full text-xs ${
                                key.revoked_at
                                  ? 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-200'
                                  : 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-200'
                              }`}
                            >
                              {key.revoked_at ? 'Revoked' : 'Active'}
                            </span>
                          </div>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500 dark:text-gray-400">No API keys</p>
                  )}
                </div>

                {/* JWT Tokens */}
                <div>
                  <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">
                    Active JWT Tokens ({selectedUser.jwtTokens.length})
                  </h3>
                  {selectedUser.jwtTokens.length > 0 ? (
                    <div className="space-y-2 max-h-48 overflow-y-auto">
                      {selectedUser.jwtTokens.map((token) => (
                        <div
                          key={token.id}
                          className="p-3 bg-gray-50 dark:bg-gray-700 rounded-md border border-gray-200 dark:border-gray-600"
                        >
                          <p className="text-xs text-gray-500 dark:text-gray-400">
                            Created: {formatDate(token.created_at)}
                          </p>
                          <p className="text-xs text-gray-500 dark:text-gray-400">
                            Expires: {formatDate(token.expires_at)}
                          </p>
                        </div>
                      ))}
                    </div>
                  ) : (
                    <p className="text-sm text-gray-500 dark:text-gray-400">No active JWT tokens</p>
                  )}
                </div>
              </div>
            ) : (
              <div className="bg-white dark:bg-gray-800 shadow rounded-lg p-6 text-center text-gray-500 dark:text-gray-400">
                Select a user to view details
              </div>
            )}
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
