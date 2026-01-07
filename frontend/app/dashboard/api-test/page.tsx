'use client';

import { useState, useEffect } from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { apiClient } from '@/lib/api';

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

interface Header {
  key: string;
  value: string;
}

interface PresetEndpoint {
  label: string;
  method: HttpMethod;
  endpoint: string;
  category: string;
  body?: string;
}

export default function APITestPage() {
  const [method, setMethod] = useState<HttpMethod>('GET');
  const [endpoint, setEndpoint] = useState('/api/v1/missions');
  const [headers, setHeaders] = useState<Header[]>([
    { key: 'Authorization', value: '' },
    { key: 'X-API-Key', value: '' },
  ]);
  const [body, setBody] = useState('');
  const [loading, setLoading] = useState(false);
  const [response, setResponse] = useState<{
    status: number;
    statusText: string;
    headers: Record<string, string>;
    data: any;
    error?: string;
  } | null>(null);

  useEffect(() => {
    // Load auth tokens (JWT from localStorage, API key from memory only)
    const jwtToken = localStorage.getItem('jwt_token');
    const apiKey = apiClient.getApiKey(); // Get from memory, not localStorage

    setHeaders([
      { key: 'Authorization', value: jwtToken ? `Bearer ${jwtToken}` : '' },
      { key: 'X-API-Key', value: apiKey || '' },
    ]);
  }, []);

  const handleAddHeader = () => {
    setHeaders([...headers, { key: '', value: '' }]);
  };

  const handleRemoveHeader = (index: number) => {
    setHeaders(headers.filter((_, i) => i !== index));
  };

  const handleHeaderChange = (index: number, field: 'key' | 'value', value: string) => {
    const newHeaders = [...headers];
    newHeaders[index][field] = value;
    setHeaders(newHeaders);
  };

  const handleSendRequest = async () => {
    setLoading(true);
    setResponse(null);

    try {
      // Build headers object
      const headersObj: Record<string, string> = {
        'Content-Type': 'application/json',
      };

      headers.forEach((header) => {
        if (header.key.trim() && header.value.trim()) {
          headersObj[header.key.trim()] = header.value.trim();
        }
      });

      // Build URL
      const baseURL = typeof window !== 'undefined' ? window.location.origin : '';
      const url = endpoint.startsWith('http') ? endpoint : `${baseURL}${endpoint}`;

      // Parse body if present
      let bodyData: any = null;
      if (body.trim() && (method === 'POST' || method === 'PUT' || method === 'PATCH')) {
        try {
          bodyData = JSON.parse(body);
        } catch {
          throw new Error('Invalid JSON in request body');
        }
      }

      // Make request
      const options: RequestInit = {
        method,
        headers: headersObj,
      };

      if (bodyData) {
        options.body = JSON.stringify(bodyData);
      }

      const startTime = Date.now();
      const res = await fetch(url, options);
      const responseTime = Date.now() - startTime;

      // Get response data
      let responseData: any;
      const contentType = res.headers.get('content-type');
      if (contentType && contentType.includes('application/json')) {
        responseData = await res.json();
      } else {
        responseData = await res.text();
      }

      // Build response headers object
      const responseHeaders: Record<string, string> = {};
      res.headers.forEach((value, key) => {
        responseHeaders[key] = value;
      });

      setResponse({
        status: res.status,
        statusText: res.statusText,
        headers: responseHeaders,
        data: responseData,
      });
    } catch (error) {
      setResponse({
        status: 0,
        statusText: 'Error',
        headers: {},
        data: null,
        error: error instanceof Error ? error.message : 'Unknown error occurred',
      });
    } finally {
      setLoading(false);
    }
  };

  const formatJSON = (obj: any): string => {
    try {
      return JSON.stringify(obj, null, 2);
    } catch {
      return String(obj);
    }
  };

  const presetEndpoints = [
    // Auth Endpoints
    { label: 'Login (API Key)', method: 'POST' as HttpMethod, endpoint: '/api/v1/auth/login', category: 'Auth', body: '{"api_key": "your-api-key"}' },
    { label: 'Get Current User', method: 'GET' as HttpMethod, endpoint: '/api/v1/me', category: 'Auth' },
    { label: 'Refresh Token', method: 'POST' as HttpMethod, endpoint: '/api/v1/auth/refresh', category: 'Auth', body: '{"refresh_token": "your-refresh-token"}' },
    { label: 'Token Exchange', method: 'POST' as HttpMethod, endpoint: '/api/v1/auth/token', category: 'Auth', body: '{"code": "auth-code", "code_verifier": "code-verifier"}' },
    { label: 'Authentik Token Exchange', method: 'POST' as HttpMethod, endpoint: '/api/v1/auth/authentik/token', category: 'Auth', body: '{"code": "auth-code", "code_verifier": "code-verifier", "redirect_uri": "https://..."}' },
    { label: 'Authentik Register', method: 'POST' as HttpMethod, endpoint: '/api/v1/auth/authentik/register', category: 'Auth', body: '{"access_token": "authentik-token"}' },
    
    // Quests/Missions (Read)
    { label: 'List Quests', method: 'GET' as HttpMethod, endpoint: '/api/v1/quests', category: 'Quests' },
    { label: 'Get Quest', method: 'GET' as HttpMethod, endpoint: '/api/v1/quests/1', category: 'Quests' },
    { label: 'List Missions', method: 'GET' as HttpMethod, endpoint: '/api/v1/missions', category: 'Quests' },
    { label: 'Get Mission', method: 'GET' as HttpMethod, endpoint: '/api/v1/missions/1', category: 'Quests' },
    
    // Quests/Missions (Write - Admin)
    { label: 'Create Quest', method: 'POST' as HttpMethod, endpoint: '/api/v1/quests', category: 'Quests (Write)', body: '{"name": "Quest Name", "external_id": "quest-123"}' },
    { label: 'Update Quest', method: 'PUT' as HttpMethod, endpoint: '/api/v1/quests/1', category: 'Quests (Write)', body: '{"name": "Updated Name"}' },
    { label: 'Delete Quest', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/quests/1', category: 'Quests (Write)' },
    
    // Items (Read)
    { label: 'List Items', method: 'GET' as HttpMethod, endpoint: '/api/v1/items', category: 'Items' },
    { label: 'Get Item', method: 'GET' as HttpMethod, endpoint: '/api/v1/items/1', category: 'Items' },
    { label: 'Get Required Items', method: 'GET' as HttpMethod, endpoint: '/api/v1/items/required', category: 'Items' },
    { label: 'Get Blueprints', method: 'GET' as HttpMethod, endpoint: '/api/v1/items/blueprints', category: 'Items' },
    
    // Items (Write - Admin)
    { label: 'Create Item', method: 'POST' as HttpMethod, endpoint: '/api/v1/items', category: 'Items (Write)', body: '{"name": "Item Name", "external_id": "item-123"}' },
    { label: 'Update Item', method: 'PUT' as HttpMethod, endpoint: '/api/v1/items/1', category: 'Items (Write)', body: '{"name": "Updated Name"}' },
    { label: 'Delete Item', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/items/1', category: 'Items (Write)' },
    
    // Skill Nodes (Read)
    { label: 'List Skill Nodes', method: 'GET' as HttpMethod, endpoint: '/api/v1/skill-nodes', category: 'Skill Nodes' },
    { label: 'Get Skill Node', method: 'GET' as HttpMethod, endpoint: '/api/v1/skill-nodes/1', category: 'Skill Nodes' },
    
    // Skill Nodes (Write - Admin)
    { label: 'Create Skill Node', method: 'POST' as HttpMethod, endpoint: '/api/v1/skill-nodes', category: 'Skill Nodes (Write)', body: '{"name": "Skill Name", "external_id": "skill-123"}' },
    { label: 'Update Skill Node', method: 'PUT' as HttpMethod, endpoint: '/api/v1/skill-nodes/1', category: 'Skill Nodes (Write)', body: '{"name": "Updated Name"}' },
    { label: 'Delete Skill Node', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/skill-nodes/1', category: 'Skill Nodes (Write)' },
    
    // Hideout Modules (Read)
    { label: 'List Hideout Modules', method: 'GET' as HttpMethod, endpoint: '/api/v1/hideout-modules', category: 'Hideout Modules' },
    { label: 'Get Hideout Module', method: 'GET' as HttpMethod, endpoint: '/api/v1/hideout-modules/1', category: 'Hideout Modules' },
    
    // Hideout Modules (Write - Admin)
    { label: 'Create Hideout Module', method: 'POST' as HttpMethod, endpoint: '/api/v1/hideout-modules', category: 'Hideout Modules (Write)', body: '{"name": "Module Name", "external_id": "module-123"}' },
    { label: 'Update Hideout Module', method: 'PUT' as HttpMethod, endpoint: '/api/v1/hideout-modules/1', category: 'Hideout Modules (Write)', body: '{"name": "Updated Name"}' },
    { label: 'Delete Hideout Module', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/hideout-modules/1', category: 'Hideout Modules (Write)' },
    
    // Enemy Types (Read)
    { label: 'List Enemy Types', method: 'GET' as HttpMethod, endpoint: '/api/v1/enemy-types', category: 'Enemy Types' },
    { label: 'Get Enemy Type', method: 'GET' as HttpMethod, endpoint: '/api/v1/enemy-types/1', category: 'Enemy Types' },
    
    // Enemy Types (Write - Admin)
    { label: 'Create Enemy Type', method: 'POST' as HttpMethod, endpoint: '/api/v1/enemy-types', category: 'Enemy Types (Write)', body: '{"name": "Enemy Name", "external_id": "enemy-123"}' },
    { label: 'Update Enemy Type', method: 'PUT' as HttpMethod, endpoint: '/api/v1/enemy-types/1', category: 'Enemy Types (Write)', body: '{"name": "Updated Name"}' },
    { label: 'Delete Enemy Type', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/enemy-types/1', category: 'Enemy Types (Write)' },
    
    // Alerts (Read)
    { label: 'List Alerts', method: 'GET' as HttpMethod, endpoint: '/api/v1/alerts', category: 'Alerts' },
    { label: 'Get Active Alerts', method: 'GET' as HttpMethod, endpoint: '/api/v1/alerts/active', category: 'Alerts' },
    { label: 'Get Alert', method: 'GET' as HttpMethod, endpoint: '/api/v1/alerts/1', category: 'Alerts' },
    
    // Alerts (Write - Admin)
    { label: 'Create Alert', method: 'POST' as HttpMethod, endpoint: '/api/v1/alerts', category: 'Alerts (Write)', body: '{"title": "Alert Title", "message": "Alert Message"}' },
    { label: 'Update Alert', method: 'PUT' as HttpMethod, endpoint: '/api/v1/alerts/1', category: 'Alerts (Write)', body: '{"title": "Updated Title"}' },
    { label: 'Delete Alert', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/alerts/1', category: 'Alerts (Write)' },
    
    // Bots, Maps, Traders, Projects (Read)
    { label: 'List Bots', method: 'GET' as HttpMethod, endpoint: '/api/v1/bots', category: 'Other' },
    { label: 'Get Bot', method: 'GET' as HttpMethod, endpoint: '/api/v1/bots/1', category: 'Other' },
    { label: 'List Maps', method: 'GET' as HttpMethod, endpoint: '/api/v1/maps', category: 'Other' },
    { label: 'Get Map', method: 'GET' as HttpMethod, endpoint: '/api/v1/maps/1', category: 'Other' },
    { label: 'List Traders', method: 'GET' as HttpMethod, endpoint: '/api/v1/traders', category: 'Other' },
    { label: 'List Repo Traders', method: 'GET' as HttpMethod, endpoint: '/api/v1/repo-traders', category: 'Other' },
    { label: 'Get Repo Trader', method: 'GET' as HttpMethod, endpoint: '/api/v1/repo-traders/1', category: 'Other' },
    { label: 'List Projects', method: 'GET' as HttpMethod, endpoint: '/api/v1/projects', category: 'Other' },
    { label: 'Get Project', method: 'GET' as HttpMethod, endpoint: '/api/v1/projects/1', category: 'Other' },
    
    // Progress (User)
    { label: 'Get My Quest Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/progress/quests', category: 'Progress' },
    { label: 'Update Quest Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/progress/quests/quest-123', category: 'Progress', body: '{"completed": true}' },
    { label: 'Get My Hideout Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/progress/hideout-modules', category: 'Progress' },
    { label: 'Update Hideout Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/progress/hideout-modules/module-123', category: 'Progress', body: '{"unlocked": true, "level": 1}' },
    { label: 'Get My Skill Node Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/progress/skill-nodes', category: 'Progress' },
    { label: 'Update Skill Node Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/progress/skill-nodes/skill-123', category: 'Progress', body: '{"unlocked": true, "level": 1}' },
    { label: 'Get My Blueprint Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/progress/blueprints', category: 'Progress' },
    { label: 'Update Blueprint Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/progress/blueprints/item-123', category: 'Progress', body: '{"consumed": true}' },
    
    // Admin - API Keys
    { label: 'Create API Key', method: 'POST' as HttpMethod, endpoint: '/api/v1/admin/api-keys', category: 'Admin', body: '{"name": "My API Key"}' },
    { label: 'List API Keys', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/api-keys', category: 'Admin' },
    { label: 'Revoke API Key', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/admin/api-keys/1', category: 'Admin' },
    
    // Admin - JWT Tokens
    { label: 'List JWT Tokens', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/jwts', category: 'Admin' },
    { label: 'Revoke JWT', method: 'POST' as HttpMethod, endpoint: '/api/v1/admin/jwts/revoke', category: 'Admin', body: '{"token": "jwt-token"}' },
    
    // Admin - Users
    { label: 'List Users', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users', category: 'Admin' },
    { label: 'Get User', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users/1', category: 'Admin' },
    { label: 'Update User Access', method: 'PUT' as HttpMethod, endpoint: '/api/v1/admin/users/1/access', category: 'Admin', body: '{"can_access_data": true}' },
    { label: 'Update User Role', method: 'PUT' as HttpMethod, endpoint: '/api/v1/admin/users/1/role', category: 'Admin', body: '{"role": "admin"}' },
    { label: 'Delete User', method: 'DELETE' as HttpMethod, endpoint: '/api/v1/admin/users/1', category: 'Admin' },
    { label: 'Update User Profile', method: 'PUT' as HttpMethod, endpoint: '/api/v1/users/1/profile', category: 'Admin', body: '{"username": "newusername"}' },
    
    // Admin - Progress Management
    { label: 'Get All User Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress', category: 'Admin' },
    { label: 'Get User Quest Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/quests', category: 'Admin' },
    { label: 'Update User Quest Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/quests/quest-123', category: 'Admin', body: '{"completed": true}' },
    { label: 'Get User Hideout Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/hideout-modules', category: 'Admin' },
    { label: 'Update User Hideout Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/hideout-modules/module-123', category: 'Admin', body: '{"unlocked": true, "level": 1}' },
    { label: 'Get User Skill Node Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/skill-nodes', category: 'Admin' },
    { label: 'Update User Skill Node Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/skill-nodes/skill-123', category: 'Admin', body: '{"unlocked": true, "level": 1}' },
    { label: 'Get User Blueprint Progress', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/blueprints', category: 'Admin' },
    { label: 'Update User Blueprint Progress', method: 'PUT' as HttpMethod, endpoint: '/api/v1/admin/users/1/progress/blueprints/item-123', category: 'Admin', body: '{"consumed": true}' },
    
    // Admin - Sync
    { label: 'Force Sync', method: 'POST' as HttpMethod, endpoint: '/api/v1/admin/sync/force', category: 'Admin' },
    { label: 'Get Sync Status', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/sync/status', category: 'Admin' },
    
    // Admin - Logs
    { label: 'Query Logs', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/logs', category: 'Admin' },
    
    // Admin - Export
    { label: 'Export Quests', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/quests', category: 'Admin' },
    { label: 'Export Items', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/items', category: 'Admin' },
    { label: 'Export Skill Nodes', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/skill-nodes', category: 'Admin' },
    { label: 'Export Hideout Modules', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/hideout-modules', category: 'Admin' },
    { label: 'Export Enemy Types', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/enemy-types', category: 'Admin' },
    { label: 'Export Alerts', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/alerts', category: 'Admin' },
    { label: 'Export Bots', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/bots', category: 'Admin' },
    { label: 'Export Maps', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/maps', category: 'Admin' },
    { label: 'Export Traders', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/traders', category: 'Admin' },
    { label: 'Export Projects', method: 'GET' as HttpMethod, endpoint: '/api/v1/admin/export/projects', category: 'Admin' },
    
    // Admin - Utilities
    { label: 'Cleanup Duplicate Hideout Modules', method: 'POST' as HttpMethod, endpoint: '/api/v1/admin/hideout-modules/cleanup-duplicates', category: 'Admin' },
  ];

  const [selectedCategory, setSelectedCategory] = useState<string>('All');
  const [searchQuery, setSearchQuery] = useState('');

  const categories = ['All', ...Array.from(new Set(presetEndpoints.map(e => e.category)))];
  
  const filteredEndpoints = presetEndpoints.filter(endpoint => {
    const matchesCategory = selectedCategory === 'All' || endpoint.category === selectedCategory;
    const matchesSearch = !searchQuery || 
      endpoint.label.toLowerCase().includes(searchQuery.toLowerCase()) ||
      endpoint.endpoint.toLowerCase().includes(searchQuery.toLowerCase());
    return matchesCategory && matchesSearch;
  });

  const loadPreset = (preset: PresetEndpoint) => {
    setMethod(preset.method);
    setEndpoint(preset.endpoint);
    setBody(preset.body || '');
  };

  return (
    <DashboardLayout>
      <div className="px-4 sm:px-6 lg:px-8">
        <div className="mb-6">
          <h1 className="text-2xl font-bold text-gray-900 dark:text-white">API Testing</h1>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
            Test any API endpoint and view the response
          </p>
        </div>

        <div className="bg-white dark:bg-gray-800 shadow rounded-lg">
          <div className="p-6">
            {/* Preset Endpoints */}
            <div className="mb-6">
              <div className="flex items-center justify-between mb-3">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
                  Quick Presets ({filteredEndpoints.length} endpoints)
                </label>
              </div>
              
              {/* Category Filter */}
              <div className="mb-3">
                <div className="flex flex-wrap gap-2 mb-2">
                  {categories.map((category) => (
                    <button
                      key={category}
                      onClick={() => setSelectedCategory(category)}
                      className={`px-3 py-1 text-xs rounded ${
                        selectedCategory === category
                          ? 'bg-indigo-600 text-white'
                          : 'bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 hover:bg-gray-200 dark:hover:bg-gray-600'
                      }`}
                    >
                      {category}
                    </button>
                  ))}
                </div>
              </div>

              {/* Search */}
              <div className="mb-3">
                <input
                  type="text"
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  placeholder="Search endpoints..."
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>

              {/* Endpoints List */}
              <div className="max-h-64 overflow-y-auto border border-gray-200 dark:border-gray-700 rounded-md p-2">
                <div className="space-y-1">
                  {filteredEndpoints.map((preset, idx) => (
                    <button
                      key={idx}
                      onClick={() => loadPreset(preset)}
                      className="w-full text-left px-3 py-2 text-xs bg-gray-50 dark:bg-gray-800 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center justify-between group"
                    >
                      <div className="flex-1 min-w-0">
                        <div className="font-medium truncate">{preset.label}</div>
                        <div className="text-gray-500 dark:text-gray-400 truncate text-xs mt-0.5">
                          {preset.method} {preset.endpoint}
                        </div>
                      </div>
                      <span className="ml-2 px-2 py-0.5 text-xs bg-indigo-100 dark:bg-indigo-900 text-indigo-700 dark:text-indigo-300 rounded opacity-0 group-hover:opacity-100 transition-opacity">
                        {preset.category}
                      </span>
                    </button>
                  ))}
                </div>
                {filteredEndpoints.length === 0 && (
                  <div className="text-center py-4 text-sm text-gray-500 dark:text-gray-400">
                    No endpoints found
                  </div>
                )}
              </div>
            </div>

            {/* Request Method and URL */}
            <div className="mb-4">
              <div className="flex gap-2">
                <select
                  value={method}
                  onChange={(e) => setMethod(e.target.value as HttpMethod)}
                  className="px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="GET">GET</option>
                  <option value="POST">POST</option>
                  <option value="PUT">PUT</option>
                  <option value="PATCH">PATCH</option>
                  <option value="DELETE">DELETE</option>
                </select>
                <input
                  type="text"
                  value={endpoint}
                  onChange={(e) => setEndpoint(e.target.value)}
                  placeholder="/api/v1/missions"
                  className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
                />
                <button
                  onClick={handleSendRequest}
                  disabled={loading}
                  className="px-6 py-2 bg-indigo-600 text-white rounded-md hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {loading ? 'Sending...' : 'Send'}
                </button>
              </div>
            </div>

            {/* Headers */}
            <div className="mb-4">
              <div className="flex items-center justify-between mb-2">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">Headers</label>
                <button
                  onClick={handleAddHeader}
                  className="text-sm text-indigo-600 hover:text-indigo-700 dark:text-indigo-400"
                >
                  + Add Header
                </button>
              </div>
              <div className="space-y-2">
                {headers.map((header, index) => (
                  <div key={index} className="flex gap-2">
                    <input
                      type="text"
                      value={header.key}
                      onChange={(e) => handleHeaderChange(index, 'key', e.target.value)}
                      placeholder="Header name"
                      className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
                    />
                    <input
                      type="text"
                      value={header.value}
                      onChange={(e) => handleHeaderChange(index, 'value', e.target.value)}
                      placeholder="Header value"
                      className="flex-1 px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 focus:ring-indigo-500 focus:border-indigo-500"
                    />
                    <button
                      onClick={() => handleRemoveHeader(index)}
                      className="px-3 py-2 text-red-600 hover:text-red-700 dark:text-red-400"
                    >
                      ×
                    </button>
                  </div>
                ))}
              </div>
            </div>

            {/* Request Body */}
            {(method === 'POST' || method === 'PUT' || method === 'PATCH') && (
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                  Request Body (JSON)
                </label>
                <textarea
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  rows={10}
                  placeholder='{"key": "value"}'
                  className="w-full px-3 py-2 border border-gray-300 dark:border-gray-600 rounded-md bg-white dark:bg-gray-700 text-gray-900 dark:text-white placeholder-gray-400 font-mono text-sm focus:ring-indigo-500 focus:border-indigo-500"
                />
              </div>
            )}

            {/* Response */}
            {response && (
              <div className="mt-6 border-t border-gray-200 dark:border-gray-700 pt-6">
                <h3 className="text-lg font-medium text-gray-900 dark:text-white mb-4">Response</h3>
                
                {/* Status */}
                <div className="mb-4">
                  <span
                    className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${
                      response.status >= 200 && response.status < 300
                        ? 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200'
                        : response.status >= 400
                        ? 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200'
                        : 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200'
                    }`}
                  >
                    {response.status} {response.statusText}
                  </span>
                  {response.error && (
                    <span className="ml-2 text-sm text-red-600 dark:text-red-400">{response.error}</span>
                  )}
                </div>

                {/* Response Headers */}
                {Object.keys(response.headers).length > 0 && (
                  <div className="mb-4">
                    <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Headers</h4>
                    <pre className="bg-gray-100 dark:bg-gray-900 p-3 rounded text-xs overflow-x-auto">
                      {formatJSON(response.headers)}
                    </pre>
                  </div>
                )}

                {/* Response Body */}
                <div>
                  <h4 className="text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Body</h4>
                  <pre className="bg-gray-100 dark:bg-gray-900 p-4 rounded text-xs overflow-x-auto max-h-96">
                    {response.error
                      ? response.error
                      : typeof response.data === 'string'
                      ? response.data
                      : formatJSON(response.data)}
                  </pre>
                </div>
              </div>
            )}
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}
