'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import type { JWTToken } from '@/types';
import { formatDate } from '@/lib/utils';

export default function JWTTokensPage() {
  const router = useRouter();
  const { isAuthenticated, user } = useAuthStore();
  const [tokens, setTokens] = useState<JWTToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [revokingToken, setRevokingToken] = useState<string | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    if (user?.role !== 'admin') {
      router.push('/dashboard/');
      return;
    }
    loadTokens();
  }, [isAuthenticated, router, user]);

  const loadTokens = async () => {
    try {
      setLoading(true);
      const data = await apiClient.getJWTTokens();
      setTokens(data);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setLoading(false);
    }
  };

  const handleRevoke = async () => {
    if (!revokingToken) return;
    if (!confirm('Are you sure you want to revoke this JWT token?')) {
      setRevokingToken(null);
      return;
    }

    try {
      await apiClient.revokeJWT(revokingToken);
      setRevokingToken(null);
      loadTokens();
    } catch (err) {
      alert(getErrorMessage(err));
    }
  };

  const isExpired = (expiresAt: string) => {
    return new Date(expiresAt) < new Date();
  };

  if (!isAuthenticated || user?.role !== 'admin') return null;

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-6">JWT Tokens</h1>

        {error && (
          <div className="mb-4 p-4 bg-red-50 dark:bg-red-900/20 rounded-md">
            <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        {revokingToken && (
          <div className="mb-6 p-6 bg-yellow-50 dark:bg-yellow-900/20 rounded-lg border border-yellow-200 dark:border-yellow-800">
            <h3 className="font-semibold mb-2 text-yellow-800 dark:text-yellow-200">
              Revoke Token
            </h3>
            <p className="text-sm text-yellow-700 dark:text-yellow-300 mb-4">
              Enter the JWT token to revoke:
            </p>
            <div className="space-y-3">
              <input
                type="text"
                value={revokingToken}
                onChange={(e) => setRevokingToken(e.target.value)}
                placeholder="Paste JWT token here"
                className="w-full p-2 border rounded-md dark:bg-gray-700 dark:text-white"
              />
              <div className="flex justify-end space-x-3">
                <button
                  onClick={() => setRevokingToken(null)}
                  className="px-4 py-2 border rounded-md"
                >
                  Cancel
                </button>
                <button
                  onClick={handleRevoke}
                  className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700"
                >
                  Revoke Token
                </button>
              </div>
            </div>
          </div>
        )}

        <div className="mb-4 flex justify-end">
          <button
            onClick={() => setRevokingToken('')}
            className="px-4 py-2 bg-red-600 text-white rounded-md hover:bg-red-700"
          >
            Revoke Token
          </button>
        </div>

        {loading ? (
          <div className="text-center py-8">Loading...</div>
        ) : (
          <div className="bg-white dark:bg-gray-800 shadow rounded-lg overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
              <thead className="bg-gray-50 dark:bg-gray-800">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    ID
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    User ID
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    Expires At
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    Created
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase">
                    Status
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200 dark:divide-gray-700">
                {tokens.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                      No active tokens found
                    </td>
                  </tr>
                ) : (
                  tokens.map((token) => {
                    const expired = isExpired(token.expires_at);
                    const revoked = !!token.revoked_at;
                    return (
                      <tr key={token.id}>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                          {token.id}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {token.user_id}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {formatDate(token.expires_at)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                          {formatDate(token.created_at)}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap">
                          <span
                            className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                              revoked
                                ? 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-200'
                                : expired
                                ? 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-200'
                                : 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-200'
                            }`}
                          >
                            {revoked ? 'Revoked' : expired ? 'Expired' : 'Active'}
                          </span>
                        </td>
                      </tr>
                    );
                  })
                )}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

