'use client';

import { useState, useEffect } from 'react';
import DashboardLayout from '@/components/layout/DashboardLayout';
import { apiClient } from '@/lib/api';

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH';

interface Header {
  key: string;
  value: string;
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
    // Load auth tokens from localStorage
    const jwtToken = localStorage.getItem('jwt_token');
    const apiKey = localStorage.getItem('api_key');

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
    { label: 'List Missions', method: 'GET' as HttpMethod, endpoint: '/api/v1/missions' },
    { label: 'Get Mission', method: 'GET' as HttpMethod, endpoint: '/api/v1/missions/1' },
    { label: 'List Items', method: 'GET' as HttpMethod, endpoint: '/api/v1/items' },
    { label: 'List Skill Nodes', method: 'GET' as HttpMethod, endpoint: '/api/v1/skill-nodes' },
    { label: 'List Hideout Modules', method: 'GET' as HttpMethod, endpoint: '/api/v1/hideout-modules' },
  ];

  const loadPreset = (preset: typeof presetEndpoints[0]) => {
    setMethod(preset.method);
    setEndpoint(preset.endpoint);
    setBody('');
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
              <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                Quick Presets
              </label>
              <div className="flex flex-wrap gap-2">
                {presetEndpoints.map((preset, idx) => (
                  <button
                    key={idx}
                    onClick={() => loadPreset(preset)}
                    className="px-3 py-1 text-xs bg-gray-100 dark:bg-gray-700 text-gray-700 dark:text-gray-300 rounded hover:bg-gray-200 dark:hover:bg-gray-600"
                  >
                    {preset.label}
                  </button>
                ))}
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

