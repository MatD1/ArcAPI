'use client';

import { useEffect } from 'react';
import type { Quest, Item, SkillNode, HideoutModule } from '@/types';
import { formatDate } from '@/lib/utils';
import { getMultilingualText, getMultilingualArray } from '@/lib/i18n';

type Entity = Quest | Item | SkillNode | HideoutModule;

interface ViewModalProps {
  entity: Entity | null;
  type: 'quest' | 'item' | 'skill-node' | 'hideout-module';
  onClose: () => void;
}

export default function ViewModal({ entity, type, onClose }: ViewModalProps) {
  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        onClose();
      }
    };

    if (entity) {
      document.addEventListener('keydown', handleEscape);
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleEscape);
      document.body.style.overflow = 'unset';
    };
  }, [entity, onClose]);

  if (!entity) return null;

  const renderValue = (value: any): React.ReactNode => {
    if (value === null || value === undefined) {
      return <span className="text-gray-400 italic">Not set</span>;
    }

    if (typeof value === 'boolean') {
      return value ? 'Yes' : 'No';
    }

    // Handle arrays (e.g., objectives)
    if (Array.isArray(value)) {
      if (value.length === 0) {
        return <span className="text-gray-400 italic">Empty</span>;
      }
      return (
        <ul className="list-disc list-inside space-y-1">
          {value.map((item, idx) => (
            <li key={idx} className="break-words">{String(item)}</li>
          ))}
        </ul>
      );
    }

    if (typeof value === 'object') {
      // Check if it's a multilingual object (has language codes as keys)
      const keys = Object.keys(value);
      const languageCodes = ['en', 'de', 'es', 'fr', 'it', 'ja', 'kr', 'no', 'pl', 'pt', 'ru', 'tr', 'uk', 'zh-CN', 'zh-TW', 'da', 'hr', 'sr'];
      const isMultilingual = keys.some(key => languageCodes.includes(key));
      
      if (isMultilingual) {
        // Display as a simple list of language: text
        return (
          <div className="space-y-1">
            {Object.entries(value).map(([lang, text]) => (
              <div key={lang} className="text-sm">
                <span className="font-medium text-gray-500 dark:text-gray-400">{lang}:</span>{' '}
                <span className="break-words">{String(text)}</span>
              </div>
            ))}
          </div>
        );
      }
      
      // Otherwise render as JSON
      return (
        <pre className="bg-gray-100 dark:bg-gray-800 p-3 rounded text-xs overflow-x-auto">
          {JSON.stringify(value, null, 2)}
        </pre>
      );
    }

    if (typeof value === 'string' && (value.startsWith('http://') || value.startsWith('https://'))) {
      return (
        <a
          href={value}
          target="_blank"
          rel="noopener noreferrer"
          className="text-indigo-600 hover:text-indigo-900 dark:text-indigo-400 break-all"
        >
          {value}
        </a>
      );
    }

    return <span className="break-words">{String(value)}</span>;
  };

  const renderEntityDetails = () => {
    const data = (entity as any).data || {};
    
    // Extract multilingual name and description
    let displayName = (entity as any).name;
    if (!displayName || displayName === '') {
      displayName = getMultilingualText(data.name);
    }
    
    let displayDescription = (entity as any).description;
    if (!displayDescription || displayDescription === '') {
      displayDescription = getMultilingualText(data.description);
    }

    const commonFields = [
      { label: 'ID', value: entity.id },
      { label: 'External ID', value: (entity as any).external_id },
      { label: 'Name', value: displayName || (entity as any).external_id },
      { label: 'Description', value: displayDescription },
    ];

    const fields: Array<{ label: string; value: any }> = [...commonFields];

    if (type === 'quest') {
      const m = entity as Quest;
      
      // Extract multilingual objectives
      let objectives: string[] = [];
      if (m.objectives && m.objectives.objectives) {
        objectives = getMultilingualArray(m.objectives.objectives);
      } else if (data.objectives && Array.isArray(data.objectives)) {
        objectives = getMultilingualArray(data.objectives);
      }
      
      fields.push(
        { label: 'Trader', value: m.trader || data.trader },
        { label: 'XP', value: m.xp !== undefined ? m.xp : data.xp },
        { label: 'Objectives', value: objectives.length > 0 ? objectives : m.objectives },
        { label: 'Reward Item IDs', value: m.reward_item_ids || data.rewardItemIds }
      );
    } else if (type === 'item') {
      const i = entity as Item;
      fields.push(
        { label: 'Type', value: i.type },
        { label: 'Image URL', value: i.image_url },
        { label: 'Image Filename', value: i.image_filename }
      );
    } else if (type === 'skill-node') {
      const sn = entity as SkillNode;
      
      // Extract multilingual impacted skill
      let impactedSkill = sn.impacted_skill;
      if (!impactedSkill && data.impactedSkill) {
        if (typeof data.impactedSkill === 'object') {
          impactedSkill = getMultilingualText(data.impactedSkill);
        } else {
          impactedSkill = data.impactedSkill;
        }
      }
      
      fields.push(
        { label: 'Impacted Skill', value: impactedSkill },
        { label: 'Category', value: sn.category || data.category },
        { label: 'Max Points', value: sn.max_points !== undefined ? sn.max_points : data.maxPoints },
        { label: 'Icon Name', value: sn.icon_name || data.iconName },
        { label: 'Is Major', value: sn.is_major !== undefined ? sn.is_major : data.isMajor },
        { label: 'Position', value: sn.position || data.position },
        { label: 'Known Value', value: sn.known_value || data.knownValue },
        { label: 'Prerequisite Node IDs', value: sn.prerequisite_node_ids || data.prerequisiteNodeIds }
      );
    } else if (type === 'hideout-module') {
      const hm = entity as HideoutModule;
      fields.push(
        { label: 'Max Level', value: hm.max_level !== undefined ? hm.max_level : data.maxLevel },
        { label: 'Levels', value: hm.levels || data.levels }
      );
    }

    fields.push(
      { label: 'Synced At', value: entity.synced_at ? formatDate(entity.synced_at) : null },
      { label: 'Created At', value: formatDate(entity.created_at) },
      { label: 'Updated At', value: formatDate(entity.updated_at) },
      { label: 'Full Data (JSON)', value: (entity as any).data }
    );

    return fields;
  };

  const fields = renderEntityDetails();

  return (
    <div
      className="fixed inset-0 z-50 overflow-y-auto"
      onClick={onClose}
      role="dialog"
      aria-modal="true"
      aria-labelledby="modal-title"
    >
      <div className="flex min-h-screen items-center justify-center p-4">
        {/* Backdrop */}
        <div className="fixed inset-0 bg-black bg-opacity-50 transition-opacity" />

        {/* Modal */}
        <div
          className="relative w-full max-w-4xl max-h-[90vh] bg-white dark:bg-gray-800 rounded-lg shadow-xl overflow-hidden"
          onClick={(e) => e.stopPropagation()}
        >
          {/* Header */}
          <div className="sticky top-0 bg-gray-50 dark:bg-gray-900 px-6 py-4 border-b border-gray-200 dark:border-gray-700 flex items-center justify-between">
            <h3 className="text-lg font-semibold text-gray-900 dark:text-white capitalize">
              {type.replace('-', ' ')} Details
            </h3>
            <button
              onClick={onClose}
              className="text-gray-400 hover:text-gray-500 dark:hover:text-gray-300 focus:outline-none"
              aria-label="Close"
            >
              <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>

          {/* Content */}
          <div className="p-6 overflow-y-auto max-h-[calc(90vh-120px)]">
            {/* Image preview for items */}
            {type === 'item' && (entity as Item).image_url && (
              <div className="mb-6 flex justify-center">
                <img
                  src={(entity as Item).image_url}
                  alt={getMultilingualText((entity as any).data?.name) || (entity as Item).name || 'Item image'}
                  className="max-h-64 max-w-full object-contain rounded-lg border border-gray-200 dark:border-gray-700"
                  onError={(e) => {
                    const target = e.target as HTMLImageElement;
                    target.style.display = 'none';
                  }}
                />
              </div>
            )}

            {/* Fields */}
            <div className="space-y-4">
              {fields.map((field, index) => (
                <div key={index} className="border-b border-gray-200 dark:border-gray-700 pb-4 last:border-0">
                  <dt className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">{field.label}</dt>
                  <dd className="text-sm text-gray-900 dark:text-white">{renderValue(field.value)}</dd>
                </div>
              ))}
            </div>
          </div>

          {/* Footer */}
          <div className="sticky bottom-0 bg-gray-50 dark:bg-gray-900 px-6 py-4 border-t border-gray-200 dark:border-gray-700 flex justify-end">
            <button
              onClick={onClose}
              className="px-4 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
            >
              Close
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

