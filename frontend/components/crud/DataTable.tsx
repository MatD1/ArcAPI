'use client';

import { useState } from 'react';
import type { Quest, Item, SkillNode, HideoutModule } from '@/types';
import { formatDate } from '@/lib/utils';
import ViewModal from './ViewModal';

type Entity = Quest | Item | SkillNode | HideoutModule;

// Mission is deprecated, use Quest instead
type Mission = Quest;

interface DataTableProps {
  data: Entity[];
  onEdit: (item: Entity) => void;
  onDelete: (id: number) => void;
  type: 'quest' | 'item' | 'skill-node' | 'hideout-module';
}

export default function DataTable({ data, onEdit, onDelete, type }: DataTableProps) {
  const [viewingEntity, setViewingEntity] = useState<Entity | null>(null);
  const getDisplayFields = (item: Entity) => {
    const base = { id: item.id, external_id: (item as any).external_id, name: (item as any).name };
    if (type === 'item') {
      return { ...base, image_url: (item as Item).image_url } as typeof base & { image_url?: string };
    }
    return base as typeof base & { image_url?: string };
  };

  return (
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
            {type === 'item' && (
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wider">
                Image URL
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
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {fields.external_id}
                </td>
                <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">{fields.name}</td>
                {type === 'item' && (
                  <td className="px-6 py-4 text-sm">
                    {fields.image_url ? (
                      <img
                        src={fields.image_url}
                        alt={fields.name || 'Item image'}
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
                  <button
                    onClick={() => onEdit(item)}
                    className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 mr-4"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => onDelete(item.id)}
                    className="text-red-600 hover:text-red-900 dark:text-red-400"
                  >
                    Delete
                  </button>
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

