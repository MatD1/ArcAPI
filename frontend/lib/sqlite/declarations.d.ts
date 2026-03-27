declare module 'wa-sqlite/dist/wa-sqlite-async.mjs' {
  const factory: () => Promise<any>;
  export default factory;
}

declare module 'wa-sqlite/src/examples/IDBBatchAtomicVFS.js' {
  export class IDBBatchAtomicVFS {
    constructor(name: string);
  }
}
