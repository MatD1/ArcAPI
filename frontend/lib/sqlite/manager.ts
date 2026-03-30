import { db } from './db';
import * as sqliteSchema from './schema';
import { get, set } from 'idb-keyval';

// Check if we need to hydrate or if there's a new version
export async function checkHydration(api: any) {
  const lastSync = await get('db_last_sync');
  const now = Date.now();
  
  // Refetch if more than 24 hours old or never synced
  if (!lastSync || (now - (lastSync as number) > 86400000)) {
    console.log('Needs hydration - fetching snapshot...');
    await hydrate(api);
  }
}

async function hydrate(api: any) {
  try {
    const response = await api.get('/sync/snapshot');
    const snapshot = response.data;

    // We can clear tables before re-inserting for a clean slate
    // and to handle record removals from the backend.

    // Quests
    if (snapshot.quests) {
      for (const quest of snapshot.quests) {
         await db.insert(sqliteSchema.quests).values({
           ...quest,
           synced_at: new Date(quest.synced_at),
         }).onConflictDoUpdate({
           target: sqliteSchema.quests.external_id,
           set: { ...quest, synced_at: new Date(quest.synced_at) }
         });
      }
    }

    // Items
    if (snapshot.items) {
      for (const item of snapshot.items) {
         await db.insert(sqliteSchema.items).values({
           ...item,
           synced_at: new Date(item.synced_at),
         }).onConflictDoUpdate({
           target: sqliteSchema.items.external_id,
           set: { ...item, synced_at: new Date(item.synced_at) }
         });
      }
    }

    // Hideout Modules
    if (snapshot.hideout_modules) {
      for (const module of snapshot.hideout_modules) {
         await db.insert(sqliteSchema.hideoutModules).values({
           ...module,
           synced_at: new Date(module.synced_at),
         }).onConflictDoUpdate({
           target: sqliteSchema.hideoutModules.external_id,
           set: { ...module, synced_at: new Date(module.synced_at) }
         });
      }
    }
    
    // Skill Nodes
    if (snapshot.skill_nodes) {
       for (const node of snapshot.skill_nodes) {
          await db.insert(sqliteSchema.skillNodes).values({
            ...node,
            synced_at: new Date(node.synced_at),
          }).onConflictDoUpdate({
            target: sqliteSchema.skillNodes.external_id,
            set: { ...node, synced_at: new Date(node.synced_at) }
          });
       }
    }

    // Bots
    if (snapshot.bots) {
      for (const bot of snapshot.bots) {
        await db.insert(sqliteSchema.bots).values({
          ...bot,
          synced_at: new Date(bot.synced_at),
        }).onConflictDoUpdate({
          target: sqliteSchema.bots.external_id,
          set: { ...bot, synced_at: new Date(bot.synced_at) }
        });
      }
    }

    // Maps
    if (snapshot.maps) {
      for (const map of snapshot.maps) {
        await db.insert(sqliteSchema.maps).values({
          ...map,
          synced_at: new Date(map.synced_at),
        }).onConflictDoUpdate({
          target: sqliteSchema.maps.external_id,
          set: { ...map, synced_at: new Date(map.synced_at) }
        });
      }
    }

    // Traders
    if (snapshot.traders) {
      for (const trader of snapshot.traders) {
        await db.insert(sqliteSchema.traders).values({
          ...trader,
          synced_at: new Date(trader.synced_at),
        }).onConflictDoUpdate({
          target: sqliteSchema.traders.external_id,
          set: { ...trader, synced_at: new Date(trader.synced_at) }
        });
      }
    }

    // Projects
    if (snapshot.projects) {
      for (const project of snapshot.projects) {
        await db.insert(sqliteSchema.projects).values({
          ...project,
          synced_at: new Date(project.synced_at),
        }).onConflictDoUpdate({
          target: sqliteSchema.projects.external_id,
          set: { ...project, synced_at: new Date(project.synced_at) }
        });
      }
    }

    await set('db_last_sync', Date.now());
    console.log('Hydration complete!');
  } catch (error: any) {
    if (error.response?.status === 401) {
      console.warn('Hydration skipped - unauthorized access to snapshot (public access required)');
    } else {
      console.error('Hydration failed:', error);
    }
  }
}

// Add progress to offline outbox
export async function addToOutbox(type: any, targetId: string, action: any, payload: any) {
  await db.insert(sqliteSchema.outbox).values({
    type,
    target_id: targetId,
    action,
    payload,
    created_at: new Date(),
  });
}
