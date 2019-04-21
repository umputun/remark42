import { BlockTTL } from '@app/common/types';

export function ttlToTime(ttl: BlockTTL): string {
  const now = new Date();
  if (ttl === 'permanently') {
    now.setFullYear(now.getFullYear() + 100);
    return now.toISOString();
  }
  if (ttl === '43200m') {
    now.setMonth(now.getMonth() + 1);
    return now.toISOString();
  }
  if (ttl === '10080m') {
    now.setDate(now.getDate() + 7);
    return now.toISOString();
  }
  if (ttl === '1440m') {
    now.setDate(now.getDate() + 1);
    return now.toISOString();
  }
  throw new Error('unknown block ttl');
}
