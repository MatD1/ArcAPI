'use client';

import { useState } from 'react';
import type { Quest, Item, SkillNode, HideoutModule, EnemyType, Alert } from '@/types';
import { formatDate } from '@/lib/utils';
import { getMultilingualText } from '@/lib/i18n';
import ViewModal from './ViewModal';

type Entity = Quest | Item | SkillNode | HideoutModule | EnemyType | Alert;

// Mission is deprecated, use Quest instead
type Mission = Quest;

interface DataTableProps {
  data: Entity[];
  onEdit?: (item: Entity) => void;
  onDelete?: (id: number) => void;
  type: 'quest' | 'item' | 'skill-node' | 'hideout-module' | 'enemy-type' | 'alert';
}

export default function DataTable({ data, onEdit, onDelete, type }: DataTableProps) {
  const [viewingEntity, setViewingEntity] = useState<Entity | null>(null);
  const getDisplayFields = (item: Entity) => {
    // Extract multilingual name
    let displayName = (item as any).name;
    if (!displayName || displayName === '') {
      // Try to get from data.name if it's a multilingual object
      const data = (item as any).data;
      if (data && data.name && typeof data.name === 'object') {
        displayName = getMultilingualText(data.name);
      }
    }
    
    const base = { 
      id: item.id, 
      external_id: type === 'alert' ? undefined : (item as any).external_id, 
      name: displayName || (type === 'alert' ? 'Unnamed Alert' : (item as any).external_id) // Fallback
    };
    if (type === 'item' || type === 'enemy-type') {
      return { ...base, image_url: (item as Item | EnemyType).image_url } as typeof base & { image_url?: string };
    }
    return base as typeof base & { image_url?: string };
  };

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'info':
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900/20 dark:text-blue-300';
      case 'warning':
        return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/20 dark:text-yellow-300';
      case 'error':
        return 'bg-red-100 text-red-800 dark:bg-red-900/20 dark:text-red-300';
      case 'critical':
        return 'bg-red-200 text-red-900 dark:bg-red-900/40 dark:text-red-200';
      default:
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300';
    }
  };

  return (
    <div className="overflow-x-auto">
      <table className="min-w-full divide-y divide-gray-200 dark:divide-gray-700">
        <thead className="bg-gray-50 dark:bg-gray-800">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              ID
            </th>
            {type !== 'alert' && (
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                External ID
              </th>
            )}
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Name
            </th>
            {type === 'alert' && (
              <>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Severity
                </th>
                <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                  Status
                </th>
              </>
            )}
            {(type === 'item' || type === 'enemy-type') && (
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Image
              </th>
            )}
            <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Updated
            </th>
            <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
              Actions
            </th>
          </tr>
        </thead>
        <tbody className="bg-white dark:bg-gray-900 divide-y divide-gray-200 dark:divide-gray-700">
          {data.map((item) => {
            const fields = getDisplayFields(item);
            return (
              <tr
                key={item.id}
                onClick={() => setViewingEntity(item)}
                className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 transition-colors"
              >
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                  {item.id}
                </td>
                {type !== 'alert' && (
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                    {fields.external_id}
                  </td>
                )}
                <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">{fields.name}</td>
                {type === 'alert' && (
                  <>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${getSeverityColor((item as Alert).severity)}`}>
                        {(item as Alert).severity.toUpperCase()}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      <span className={`px-2 py-1 text-xs font-semibold rounded-full ${
                        (item as Alert).is_active
                          ? 'bg-green-100 text-green-800 dark:bg-green-900/20 dark:text-green-300'
                          : 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-300'
                      }`}>
                        {(item as Alert).is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                  </>
                )}
                {(type === 'item' || type === 'enemy-type') && (
                  <td className="px-6 py-4 text-sm">
                    {fields.image_url ? (
                      <img
                        src={fields.image_url}
                        alt={fields.name || (type === 'enemy-type' ? 'Enemy image' : 'Item image')}
                        className="h-12 w-12 object-contain rounded"
                        onError={(e) => {
                          // Replace image with a broken image icon or placeholder
                          const target = e.target as HTMLImageElement;
                          target.style.display = 'none';
                          const placeholder = document.createElement('div');
                          placeholder.className = 'h-12 w-12 flex items-center justify-center bg-gray-200 dark:bg-gray-700 rounded text-xs text-gray-500';
                          placeholder.textContent = 'No img';
                          if (fields.image_url) {
                            placeholder.title = fields.image_url;
                          }
                          target.parentElement?.appendChild(placeholder);
                        }}
                      />
                    ) : (
                      <div className="h-12 w-12 flex items-center justify-center bg-gray-200 dark:bg-gray-700 rounded text-xs text-gray-500">
                        No img
                      </div>
                    )}
                  </td>
                )}
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {formatDate(item.updated_at)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium" onClick={(e) => e.stopPropagation()}>
                  <button
                    onClick={() => setViewingEntity(item)}
                    className="text-blue-600 hover:text-blue-900 dark:text-blue-400 mr-4"
                    title="View Details"
                  >
                    View
                  </button>
                  {onEdit && (
                    <button
                      onClick={() => onEdit(item)}
                      className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 mr-4"
                      title="Edit"
                    >
                      Edit
                    </button>
                  )}
                  {onDelete && (
                    <button
                      onClick={() => onDelete(item.id)}
                      className="text-red-600 hover:text-red-900 dark:text-red-400"
                      title="Delete"
                    >
                      Delete
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      
      {/* View Modal */}
      <ViewModal entity={viewingEntity} type={type} onClose={() => setViewingEntity(null)} />
    </div>
  );
}

