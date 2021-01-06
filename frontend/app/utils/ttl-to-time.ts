import { BlockTTL } from 'common/types';

export function ttlToDate(ttl: BlockTTL): Date {
  const date = new Date();
  switch (ttl) {
    case 'permanently':
      date.setFullYear(date.getFullYear() + 100);
      return date;
    case '43200m':
      date.setMonth(date.getMonth() + 1);
      return date;
    case '10080m':
      date.setDate(date.getDate() + 7);
      return date;
    case '1440m':
      date.setDate(date.getDate() + 1);
      return date;
    default:
      throw new Error('unknown block ttl');
  }
}

export function ttlToTime(ttl: BlockTTL): string {
  return ttlToDate(ttl).toISOString();
}
