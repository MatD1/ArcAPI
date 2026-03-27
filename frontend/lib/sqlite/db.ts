import { drizzle } from 'drizzle-orm/sqlite-proxy';
import * as schema from './schema';

let worker: Worker | null = null;
let idCounter = 0;
const callbacks = new Map<number, (res: any) => void>();

if (typeof window !== 'undefined') {
  worker = new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' });
  
  worker.onmessage = (e) => {
    const { id, result, error } = e.data;
    const resolve = callbacks.get(id);
    if (resolve) {
      callbacks.delete(id);
      if (error) {
        console.error('SQL Error:', error);
      }
      resolve(result);
    }
  };
}

async function callWorker(method: string, sql: string, params: any[]) {
  if (!worker) return [];
  const id = ++idCounter;
  return new Promise((resolve) => {
    callbacks.set(id, resolve);
    worker!.postMessage({ id, method, sql, params });
  });
}

export const db = drizzle(
  async (sql, params, method) => {
    const res = await callWorker(method, sql, params);
    return { rows: res as any[] };
  },
  { schema }
);

// Helper to ensure tables exist
export async function ensureTables() {
  // Simple table creation - usually handled by migrations, but for WASM
  // we can use Drizzle-generated SQL or just raw CREATE TABLEs for now.
  // In a production app, we'd use drizzle-kit generated migrations.
  
  // For this MVP, we'll manually ensure tables or use schema-to-sql tools.
  // Actually, for a snapshot-based app, we can just DROP/CREATE on full reload.
}
