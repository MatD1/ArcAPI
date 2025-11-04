'use client';

import { useState, useEffect } from 'react';
import type { Quest, Item, SkillNode, HideoutModule } from '@/types';

type Entity = Quest | Item | SkillNode | HideoutModule;

interface EntityFormProps {
  entity: Partial<Entity> | null;
  type: 'quest' | 'item' | 'skill-node' | 'hideout-module';
  onSubmit: (data: Partial<Entity>) => Promise<void>;
  onCancel: () => void;
}

export default function EntityForm({ entity, type, onSubmit, onCancel }: EntityFormProps) {
  const [formData, setFormData] = useState<any>({});
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    if (entity) {
      const data: any = {
        external_id: (entity as any).external_id || '',
        name: (entity as any).name || '',
        description: (entity as any).description || '',
      };

      if (type === 'quest') {
        const m = entity as Quest;
        data.trader = m.trader || '';
        data.xp = m.xp || 0;
        data.objectives = m.objectives?.objectives || [];
        data.reward_item_ids = m.reward_item_ids?.reward_item_ids || [];
      } else if (type === 'item') {
        const i = entity as Item;
        data.type = i.type || '';
        data.image_url = i.image_url || '';
        data.image_filename = i.image_filename || '';
      } else if (type === 'skill-node') {
        const sn = entity as SkillNode;
        data.impacted_skill = sn.impacted_skill || '';
        data.category = sn.category || '';
        data.max_points = sn.max_points || 0;
        data.icon_name = sn.icon_name || '';
        data.is_major = sn.is_major || false;
        data.position = sn.position || { x: 0, y: 0 };
        data.known_value = sn.known_value?.known_value || [];
        data.prerequisite_node_ids = sn.prerequisite_node_ids?.prerequisite_node_ids || [];
      } else if (type === 'hideout-module') {
        const hm = entity as HideoutModule;
        data.max_level = hm.max_level || 0;
        data.levels = hm.levels?.levels || [];
      }

      setFormData(data);
    } else {
      // Default empty form
      const defaults: any = {
        external_id: '',
        name: '',
        description: '',
      };
      if (type === 'quest') {
        defaults.trader = '';
        defaults.xp = 0;
        defaults.objectives = [];
        defaults.reward_item_ids = [];
      } else if (type === 'item') {
        defaults.type = '';
        defaults.image_url = '';
        defaults.image_filename = '';
      } else if (type === 'skill-node') {
        defaults.impacted_skill = '';
        defaults.category = '';
        defaults.max_points = 0;
        defaults.icon_name = '';
        defaults.is_major = false;
        defaults.position = { x: 0, y: 0 };
        defaults.known_value = [];
        defaults.prerequisite_node_ids = [];
      } else if (type === 'hideout-module') {
        defaults.max_level = 0;
        defaults.levels = [];
      }
      setFormData(defaults);
    }
  }, [entity, type]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    try {
      const data: any = {
        external_id: formData.external_id,
        name: formData.name,
        description: formData.description,
      };

      if (type === 'quest') {
        data.trader = formData.trader;
        data.xp = parseInt(formData.xp) || 0;
        data.objectives = { objectives: Array.isArray(formData.objectives) ? formData.objectives : [] };
        data.reward_item_ids = { reward_item_ids: Array.isArray(formData.reward_item_ids) ? formData.reward_item_ids : [] };
      } else if (type === 'item') {
        data.type = formData.type;
        data.image_url = formData.image_url;
        data.image_filename = formData.image_filename;
      } else if (type === 'skill-node') {
        data.impacted_skill = formData.impacted_skill;
        data.category = formData.category;
        data.max_points = parseInt(formData.max_points) || 0;
        data.icon_name = formData.icon_name;
        data.is_major = formData.is_major || false;
        data.position = formData.position || { x: 0, y: 0 };
        data.known_value = { known_value: Array.isArray(formData.known_value) ? formData.known_value : [] };
        data.prerequisite_node_ids = { prerequisite_node_ids: Array.isArray(formData.prerequisite_node_ids) ? formData.prerequisite_node_ids : [] };
      } else if (type === 'hideout-module') {
        data.max_level = parseInt(formData.max_level) || 0;
        data.levels = { levels: Array.isArray(formData.levels) ? formData.levels : [] };
      }

      await onSubmit(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'An error occurred');
    } finally {
      setLoading(false);
    }
  };

  const updateField = (field: string, value: any) => {
    setFormData({ ...formData, [field]: value });
  };

  const addArrayItem = (field: string, defaultValue: any = '') => {
    const current = formData[field] || [];
    updateField(field, [...current, defaultValue]);
  };

  const removeArrayItem = (field: string, index: number) => {
    const current = formData[field] || [];
    updateField(field, current.filter((_: any, i: number) => i !== index));
  };

  const updateArrayItem = (field: string, index: number, value: any) => {
    const current = formData[field] || [];
    updateField(field, current.map((item: any, i: number) => (i === index ? value : item)));
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-4 max-h-[80vh] overflow-y-auto">
      {error && (
        <div className="rounded-md bg-red-50 dark:bg-red-900/20 p-4">
          <p className="text-sm text-red-800 dark:text-red-200">{error}</p>
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
          External ID *
        </label>
        <input
          type="text"
          required
          value={formData.external_id || ''}
          onChange={(e) => updateField('external_id', e.target.value)}
          className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Name *</label>
        <input
          type="text"
          required
          value={formData.name || ''}
          onChange={(e) => updateField('name', e.target.value)}
          className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
          Description
        </label>
        <textarea
          value={formData.description || ''}
          onChange={(e) => updateField('description', e.target.value)}
          rows={3}
          className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
        />
      </div>

      {/* Quest-specific fields */}
      {type === 'quest' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Trader</label>
            <input
              type="text"
              value={formData.trader || ''}
              onChange={(e) => updateField('trader', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">XP</label>
            <input
              type="number"
              value={formData.xp || 0}
              onChange={(e) => updateField('xp', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Objectives (one per line)
            </label>
            <textarea
              value={(formData.objectives || []).join('\n')}
              onChange={(e) => updateField('objectives', e.target.value.split('\n').filter(l => l.trim()))}
              rows={4}
              placeholder="Objective 1&#10;Objective 2"
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Reward Item IDs (JSON format)
            </label>
            <textarea
              value={JSON.stringify(formData.reward_item_ids || [], null, 2)}
              onChange={(e) => {
                try {
                  updateField('reward_item_ids', JSON.parse(e.target.value));
                } catch {
                  // Invalid JSON, ignore
                }
              }}
              rows={6}
              placeholder='[{"itemId": "item1", "quantity": 1}]'
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm font-mono text-xs"
            />
          </div>
        </>
      )}

      {/* Item-specific fields */}
      {type === 'item' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Type</label>
            <input
              type="text"
              value={formData.type || ''}
              onChange={(e) => updateField('type', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Image URL
            </label>
            <input
              type="url"
              value={formData.image_url || ''}
              onChange={(e) => updateField('image_url', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Image Filename
            </label>
            <input
              type="text"
              value={formData.image_filename || ''}
              onChange={(e) => updateField('image_filename', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
        </>
      )}

      {/* Skill Node-specific fields */}
      {type === 'skill-node' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Impacted Skill</label>
            <input
              type="text"
              value={formData.impacted_skill || ''}
              onChange={(e) => updateField('impacted_skill', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Category</label>
            <input
              type="text"
              value={formData.category || ''}
              onChange={(e) => updateField('category', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Max Points</label>
              <input
                type="number"
                value={formData.max_points || 0}
                onChange={(e) => updateField('max_points', e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Icon Name</label>
              <input
                type="text"
                value={formData.icon_name || ''}
                onChange={(e) => updateField('icon_name', e.target.value)}
                className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
              />
            </div>
          </div>
          <div className="flex items-center">
            <input
              type="checkbox"
              checked={formData.is_major || false}
              onChange={(e) => updateField('is_major', e.target.checked)}
              className="h-4 w-4 text-indigo-600 focus:ring-indigo-500 border-gray-300 rounded"
            />
            <label className="ml-2 block text-sm text-gray-700 dark:text-gray-300">Is Major</label>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Position (JSON: {"{x, y}"})</label>
            <input
              type="text"
              value={JSON.stringify(formData.position || { x: 0, y: 0 })}
              onChange={(e) => {
                try {
                  updateField('position', JSON.parse(e.target.value));
                } catch {
                  // Invalid JSON, ignore
                }
              }}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm font-mono text-xs"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Prerequisite Node IDs (one per line)
            </label>
            <textarea
              value={(formData.prerequisite_node_ids || []).join('\n')}
              onChange={(e) => updateField('prerequisite_node_ids', e.target.value.split('\n').filter(l => l.trim()))}
              rows={3}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Known Value (JSON array)
            </label>
            <textarea
              value={JSON.stringify(formData.known_value || [], null, 2)}
              onChange={(e) => {
                try {
                  updateField('known_value', JSON.parse(e.target.value));
                } catch {
                  // Invalid JSON, ignore
                }
              }}
              rows={4}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm font-mono text-xs"
            />
          </div>
        </>
      )}

      {/* Hideout Module-specific fields */}
      {type === 'hideout-module' && (
        <>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Max Level</label>
            <input
              type="number"
              value={formData.max_level || 0}
              onChange={(e) => updateField('max_level', e.target.value)}
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm"
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              Levels (JSON format)
            </label>
            <textarea
              value={JSON.stringify(formData.levels || [], null, 2)}
              onChange={(e) => {
                try {
                  updateField('levels', JSON.parse(e.target.value));
                } catch {
                  // Invalid JSON, ignore
                }
              }}
              rows={12}
              placeholder='[{"level": 1, "requirementItemIds": []}]'
              className="mt-1 block w-full rounded-md border-gray-300 dark:border-gray-600 shadow-sm focus:border-indigo-500 focus:ring-indigo-500 dark:bg-gray-700 dark:text-white sm:text-sm font-mono text-xs"
            />
          </div>
        </>
      )}

      <div className="flex justify-end space-x-3 pt-4 border-t border-gray-200 dark:border-gray-700">
        <button
          type="button"
          onClick={onCancel}
          className="px-4 py-2 border border-gray-300 dark:border-gray-600 rounded-md shadow-sm text-sm font-medium text-gray-700 dark:text-gray-300 bg-white dark:bg-gray-700 hover:bg-gray-50 dark:hover:bg-gray-600"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={loading}
          className="px-4 py-2 border border-transparent rounded-md shadow-sm text-sm font-medium text-white bg-indigo-600 hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 disabled:opacity-50"
        >
          {loading ? 'Saving...' : entity ? 'Update' : 'Create'}
        </button>
      </div>
    </form>
  );
}
