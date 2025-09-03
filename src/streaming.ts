import { config } from '@grafana/runtime';

/**
 * Calculate a unique key for the query.  The key is used to pick a channel and should
 * be unique for each distinct query execution plan.  This key is not secure and is only picked to avoid
 * possible collisions
 */
export async function getLiveStreamKey(datasourceUid: string, topic?: string): Promise<string> {
  const str = JSON.stringify({ topic });

  const orgId = config.bootData.user.orgId;
  const msgUint8 = new TextEncoder().encode(str); // encode as (utf-8) Uint8Array
  const hashBuffer = await crypto.subtle.digest('SHA-1', msgUint8); // hash the message
  const hashArray = Array.from(new Uint8Array(hashBuffer.slice(0, 8))); // first 8 bytes
  return `${datasourceUid}/${hashArray.map((b) => b.toString(16).padStart(2, '0')).join('')}/${orgId}`;
}
