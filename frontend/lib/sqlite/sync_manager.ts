import { db } from './db';
import * as schema from './schema';
import { eq, asc } from 'drizzle-orm';

// Periodic sync of the offline outbox to the backend
export async function syncOutbox(api: any) {
  // Use explicit select to avoid type issues with schema.outbox
  const pending = await db.select().from(schema.outbox).orderBy(asc(schema.outbox.created_at));
  
  if (pending.length === 0) return;

  console.log(`Syncing ${pending.length} pending items from outbox...`);

  for (const item of pending) {
    try {
      let route = '';
      if (item.type === 'quest_progress') route = `/progress/quests/${item.target_id}`;
      else if (item.type === 'hideout_module_progress') route = `/progress/hideout-modules/${item.target_id}`;
      else if (item.type === 'skill_node_progress') route = `/progress/skill-nodes/${item.target_id}`;
      else if (item.type === 'blueprint_progress') route = `/progress/blueprints/${item.target_id}`;

      if (route) {
        await api.put(route, item.payload);
        
        // Success - remove from outbox
        await db.delete(schema.outbox).where(eq(schema.outbox.id, item.id as number));
      }
    } catch (error) {
      console.error(`Failed to sync item ${item.id}:`, error);
      
      // Update retry count
      await db.update(schema.outbox)
        .set({ retry_count: (item.retry_count as number || 0) + 1 })
        .where(eq(schema.outbox.id, item.id as number));
        
      // Stop syncing to keep order (FIFO)
      break; 
    }
  }
}

// Can be called periodically or when connection is restored
export function startOutboxSync(api: any, intervalMs = 60000) {
  if (typeof window === 'undefined') return;
  
  // Initial sync
  syncOutbox(api).catch(console.error);
  
  // Periodic sync
  setInterval(() => syncOutbox(api).catch(console.error), intervalMs);
  
  // Connection listener
  window.addEventListener('online', () => syncOutbox(api).catch(console.error));
}
