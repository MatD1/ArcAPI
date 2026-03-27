import SQLiteESMFactory from 'wa-sqlite/dist/wa-sqlite-async.mjs';
import * as SQLite from 'wa-sqlite';
import { IDBBatchAtomicVFS } from 'wa-sqlite/src/examples/IDBBatchAtomicVFS.js';

let sqliteDb: any;
let sqlite: any;

async function init() {
  const module = await SQLiteESMFactory();
  sqlite = SQLite.Factory(module);
  
  const vfs = new IDBBatchAtomicVFS('arcapi-vfs');
  sqlite.vfs_register(vfs, true);

  sqliteDb = await sqlite.open_v2('arcapi.db');
}

const initPromise = init();

self.onmessage = async (e) => {
  await initPromise;
  const { id, sql, params, method } = e.data;

  try {
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
    self.postMessage({ id, error: error.message });
  }
};
