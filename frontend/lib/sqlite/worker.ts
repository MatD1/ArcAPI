import SQLiteESMFactory from 'wa-sqlite/dist/wa-sqlite-async.mjs';
import * as SQLite from 'wa-sqlite';
import { IDBBatchAtomicVFS } from 'wa-sqlite/src/examples/IDBBatchAtomicVFS.js';

let sqliteDb: any;
let sqlite: any;

async function init() {
  const module = await SQLiteESMFactory({
    locateFile: (path: string, prefix: string) => {
      if (path.endsWith('.wasm')) {
        // The wa-sqlite WASM file is bundled by Next.js and placed in /_next/static/media/
        // with a content hash. We need to locate it in the correct place.
        // 
        // Next.js does: node_modules/wa-sqlite/dist/wa-sqlite-async.wasm
        //           -> /_next/static/media/wa-sqlite-async.<hash>.wasm
        //
        // The locateFile callback receives the filename but not the hash,
        // so we need to search for the file in the _next directory or use a known path.
        
        try {
          // Best approach: Try to use import.meta.url if available (works in modern bundlers)
          // This gives us the URL of the current module and we can navigate from there
          if (typeof import.meta !== 'undefined' && import.meta.url) {
            try {
              const workerUrl = new URL(import.meta.url);
              // Navigate up to _next/static/media where the WASM is
              // The worker is typically at /_next/static/chunks/
              // We need to get to /_next/static/media/
              return new URL(`../${path}`, workerUrl).href;
            } catch (e) {
              console.warn('Failed to use import.meta.url for WASM path:', e);
            }
          }
          
          // Fallback: Construct path based on origin
          // The WASM file is in the /_next/static/media/ directory
          if (typeof self !== 'undefined' && self.location) {
            // For worker context, use the origin + known path to WASM directory
            // Note: The filename will have a hash added by Next.js, but the locateFile
            // callback will be called with just the base name, so we need to search
            // for files matching the pattern or use a stable path.
            
            // Try the most likely location first
            return `${self.location.origin}/_next/static/media/${path}`;
          }
          
          return prefix + path;
        } catch (error) {
          console.error('[SQLite Worker] Failed to construct WASM URL:', path, error);
          // Last resort fallback
          return prefix + path;
        }
      }
      return prefix + path;
    }
  });
  sqlite = SQLite.Factory(module);
  
  const vfs = new IDBBatchAtomicVFS('arcapi-vfs');
  sqlite.vfs_register(vfs, true);

  sqliteDb = await sqlite.open_v2('arcapi.db');
}

// Initialize and propagate errors
let initPromise: Promise<void>;
let initError: Error | null = null;

initPromise = init().catch((error) => {
  initError = error;
  console.error('[SQLite Worker] Failed to initialize SQLite:', error);
  throw error;
});

self.onmessage = async (e) => {
  const { id, sql, params, method } = e.data;

  try {
    // Wait for initialization, but also check for errors
    await initPromise;
    
    if (initError) {
      throw new Error(`SQLite initialization failed: ${initError.message}`);
    }

    let result;
    if (method === 'all') {
      result = await sqlite.exec(sqliteDb, sql, (row: any, columns: any) => {
        const obj: any = {};
        columns.forEach((col: any, i: any) => obj[col] = row[i]);
        return obj;
      }, params);
    } else if (method === 'run') {
       result = await sqlite.exec(sqliteDb, sql, null, params);
    } else if (method === 'values') {
       result = await sqlite.exec(sqliteDb, sql, (row: any) => row, params);
    }

    self.postMessage({ id, result });
  } catch (error: any) {
    console.error('[SQLite Worker] Error executing query:', error);
    self.postMessage({ id, error: error.message });
  }
};
