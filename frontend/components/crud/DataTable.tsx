'use client';

import { useState } from 'react';
import type { Mission, Item, SkillNode, HideoutModule } from '@/types';
import { formatDate } from '@/lib/utils';

type Entity = Mission | Item | SkillNode | HideoutModule;

interface DataTableProps {
  data: Entity[];
  onEdit: (item: Entity) => void;
  onDelete: (id: number) => void;
  type: 'mission' | 'item' | 'skill-node' | 'hideout-module';
}

export default function DataTable({ data, onEdit, onDelete, type }: DataTableProps) {
  const getDisplayFields = (item: Entity) => {
    const base = { id: item.id, external_id: (item as any).external_id, name: (item as any).name };
    if (type === 'item') {
      return { ...base, image_url: (item as Item).image_url };
    }
    return base;
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
              <tr key={item.id}>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-900 dark:text-white">
                  {item.id}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {fields.external_id}
                </td>
                <td className="px-6 py-4 text-sm text-gray-900 dark:text-white">{fields.name}</td>
                {type === 'item' && fields.image_url && (
                  <td className="px-6 py-4 text-sm">
                    <a
                      href={fields.image_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400"
                    >
                      View
                    </a>
                  </td>
                )}
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500 dark:text-gray-400">
                  {formatDate(item.updated_at)}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
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
    </div>
  );
}

