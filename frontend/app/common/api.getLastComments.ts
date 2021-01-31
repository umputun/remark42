import { Comment } from './types';
import { apiFetcher } from './fetcher';

export default function getLastComments(siteId: string, max: number): Promise<Comment[]> {
  return apiFetcher.get(`/last/${max}`, { site: siteId });
}
