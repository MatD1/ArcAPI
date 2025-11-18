'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { useAuthStore } from '@/store/authStore';
import { apiClient, getErrorMessage } from '@/lib/api';

type ExportType =
  | 'quests'
  | 'items'
  | 'skillNodes'
  | 'hideoutModules'
  | 'enemyTypes'
  | 'alerts'
  | 'bots'
  | 'maps'
  | 'repoTraders'
  | 'projects';

export default function ExportPage() {
  const router = useRouter();
  const { isAuthenticated } = useAuthStore();
  const [exporting, setExporting] = useState<ExportType | null>(null);
  const [error, setError] = useState('');
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
    if (!isAuthenticated) {
      router.push('/login/');
    }
  }, [isAuthenticated, router]);

  if (!mounted || !isAuthenticated) {
    return null;
  }

  const exportLabels: Record<ExportType, string> = {
    quests: 'Quests',
    items: 'Items',
    skillNodes: 'Skill Nodes',
    hideoutModules: 'Hideout Modules',
    enemyTypes: 'Enemy Types',
    alerts: 'Alerts',
    bots: 'Bots',
    maps: 'Maps',
    repoTraders: 'Traders',
    projects: 'Projects',
  };

  const handleExport = async (type: ExportType) => {
    setError('');
    setExporting(type);

    try {
      const url = await apiClient.exportData(type);
      
      // Create a temporary link and click it to download
      const link = document.createElement('a');
      link.href = url;
      link.download = `${type}-${new Date().toISOString().split('T')[0]}.csv`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      
      // Revoke the object URL after download
      URL.revokeObjectURL(url);
    } catch (err) {
      setError(getErrorMessage(err));
    } finally {
      setExporting(null);
    }
  };

  const handleExportAll = async () => {
    if (!confirm('This will download 10 CSV files. Continue?')) {
      return;
    }

    const types: ExportType[] = [
      'quests',
      'items',
      'skillNodes',
      'hideoutModules',
      'enemyTypes',
      'alerts',
      'bots',
      'maps',
      'repoTraders',
      'projects',
    ];
    
    for (const type of types) {
      try {
        await handleExport(type);
        // Small delay between downloads to avoid overwhelming the browser
        await new Promise(resolve => setTimeout(resolve, 500));
      } catch (err) {
        console.error(`Failed to export ${type}:`, err);
      }
    }
  };

  return (
    <DashboardLayout>
      <div className="px-4 py-6 sm:px-0">
        <div className="mb-6">
          <h1 className="text-3xl font-bold text-gray-900 dark:text-white mb-2">Data Export</h1>
          <p className="text-gray-600 dark:text-gray-400">
            Export data as CSV files for importing into Appwrite or other systems
          </p>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-lg">
            <p className="text-red-800 dark:text-red-200">{error}</p>
          </div>
        )}

        <div className="mb-6 p-6 bg-white dark:bg-gray-800 rounded-lg shadow">
          <h2 className="text-xl font-semibold text-gray-900 dark:text-white mb-4">Export All Data</h2>
          <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
            Download all entity types as CSV files at once.
          </p>
          <button
            onClick={handleExportAll}
            disabled={exporting !== null}
            className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            {exporting ? `Exporting ${exportLabels[exporting]}...` : 'Export All Data'}
          </button>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {(Object.keys(exportLabels) as ExportType[]).map((type) => (
            <div
              key={type}
              className="p-6 bg-white dark:bg-gray-800 rounded-lg shadow hover:shadow-lg transition-shadow"
            >
              <div className="flex items-center justify-between mb-4">
                <h3 className="text-lg font-semibold text-gray-900 dark:text-white">
                  {exportLabels[type]}
                </h3>
                <div className="text-sm text-gray-500 dark:text-gray-400">CSV</div>
              </div>
              <p className="text-sm text-gray-600 dark:text-gray-400 mb-4">
                Export all {exportLabels[type].toLowerCase()} as a CSV file
              </p>
              <button
                onClick={() => handleExport(type)}
                disabled={exporting !== null}
                className="w-full px-4 py-2 bg-gray-100 dark:bg-gray-700 text-gray-900 dark:text-white rounded-md hover:bg-gray-200 dark:hover:bg-gray-600 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {exporting === type ? (
                  <span className="flex items-center justify-center">
                    <svg className="animate-spin -ml-1 mr-3 h-5 w-5 text-gray-900 dark:text-white" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                      <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                      <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                    </svg>
                    Exporting...
                  </span>
                ) : (
                  'Download CSV'
                )}
              </button>
            </div>
          ))}
        </div>

        <div className="mt-6 p-4 bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg">
          <h3 className="text-sm font-semibold text-blue-900 dark:text-blue-200 mb-2">Import Instructions</h3>
          <ol className="text-sm text-blue-800 dark:text-blue-300 space-y-1 list-decimal list-inside">
            <li>Download the CSV files you need</li>
            <li>Open your Appwrite console</li>
            <li>Navigate to your database and collection</li>
            <li>Use the Appwrite import feature or manually import the CSV data</li>
            <li>Ensure the collection has matching attributes for all CSV columns</li>
          </ol>
        </div>
      </div>
    </DashboardLayout>
  );
}

