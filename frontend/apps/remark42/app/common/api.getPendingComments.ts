import { Comment } from './types';
import { apiFetcher } from './fetcher';

export function getPendingComments(siteId: string): Promise<Comment[]> {
  return apiFetcher.get('/admin/pending', { site: siteId });
}
