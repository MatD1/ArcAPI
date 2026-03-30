import { drizzle } from 'drizzle-orm/sqlite-proxy';
import * as schema from './schema';

let worker: Worker | null = null;
let idCounter = 0;
const callbacks = new Map<number, (res: any) => void>();
let isWorkerReady = false;
let workerError: Error | null = null;

if (typeof window !== 'undefined') {
  try {
    worker = new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' });
    
    worker.onmessage = (e) => {
      const { id, result, error } = e.data;
      
      // Mark worker as ready after first message
      if (!isWorkerReady) {
        isWorkerReady = true;
        console.debug('[SQLite DB] Worker initialized and responding');
      }
      
      const resolve = callbacks.get(id);
      if (resolve) {
        callbacks.delete(id);
        if (error) {
          console.error('[SQLite DB] Query execution error:', error);
        }
        resolve(result);
      }
    };
    
    worker.onerror = (error) => {
      console.error('[SQLite DB] Worker error:', error.message, error.filename, error.lineno);
      workerError = new Error(`SQLite worker error: ${error.message}`);
    };
  } catch (error) {
    console.error('[SQLite DB] Failed to create worker:', error);
    workerError = error instanceof Error ? error : new Error(String(error));
  }
}

async function callWorker(method: string, sql: string, params: any[]) {
  if (!worker) {
    const msg = 'SQLite worker not available';
    console.error('[SQLite DB]', msg);
    return [];
  }
  
  if (workerError) {
    console.error('[SQLite DB] Worker has encountered an error:', workerError.message);
  }
  
  const id = ++idCounter;
  return new Promise((resolve) => {
    callbacks.set(id, resolve);
    try {
      worker!.postMessage({ id, method, sql, params });
    } catch (error) {
      callbacks.delete(id);
      console.error('[SQLite DB] Failed to post message to worker:', error);
      resolve(undefined);
    }
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
