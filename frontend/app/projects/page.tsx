'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';
import ViewModal from '@/components/crud/ViewModal';

interface Project {
  id: number;
  external_id: string;
  name: string;
  data?: any;
  synced_at?: string;
  created_at?: string;
  updated_at?: string;
}

export default function ProjectsPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [projects, setProjects] = useState<Project[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [selectedProject, setSelectedProject] = useState<Project | null>(null);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);
  const [loadAll, setLoadAll] = useState(false);
  const pageSize = 20;

  useEffect(() => {
    if (!isAuthenticated) {
      router.push('/login/');
      return;
    }
    loadProjects();
  }, [isAuthenticated, router, page, loadAll]);

  const loadProjects = async () => {
    try {
      setLoading(true);
      setError('');
      if (loadAll) {
        // Load all data at once
        const result = await apiClient.getProjects(0, 10000);
        setProjects(result.data || []);
        setTotal(result.pagination?.total || 0);
      } else {
        // Load paginated data
        const offset = (page - 1) * pageSize;
        const result = await apiClient.getProjects(offset, pageSize);
        setProjects(result.data || []);
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

  const getPhaseCount = (project: Project): number => {
    if (!project.data || !project.data.phases) return 0;
    if (Array.isArray(project.data.phases)) return project.data.phases.length;
    return 0;
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6 flex justify-between items-start">
          <div>
            <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Projects</h1>
            <p className="text-gray-600 dark:text-gray-400">View project data from the repository</p>
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
                        Phases
                      </th>
                      <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
                    {projects.length === 0 ? (
                      <tr>
                        <td colSpan={5} className="px-6 py-4 text-center text-gray-500 dark:text-gray-400">
                          No projects found
                        </td>
                      </tr>
                    ) : (
                      projects.map((project) => {
                        const displayName = project.name || getMultilingualName(project.data) || project.external_id;
                        const phaseCount = getPhaseCount(project);
                        return (
                          <tr key={project.id} className="hover:bg-gray-50 dark:hover:bg-gray-800">
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                              {project.id}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              {project.external_id}
                            </td>
                            <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">
                              {displayName}
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                              <span className="px-2 py-1 text-xs font-semibold rounded-full bg-purple-100 text-purple-800 dark:bg-purple-900/20 dark:text-purple-300">
                                {phaseCount} {phaseCount === 1 ? 'phase' : 'phases'}
                              </span>
                            </td>
                            <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                              <button
                                onClick={() => setSelectedProject(project)}
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
                  Showing all {projects.length} projects
                </span>
              </div>
            )}
          </>
        )}

        {selectedProject && (
          <ViewModal
            entity={selectedProject as any}
            type="quest"
            onClose={() => setSelectedProject(null)}
          />
        )}
      </div>
    </DashboardLayout>
  );
}
