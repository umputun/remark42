import { Comment } from './types';
import fetcher from './fetcher';

export default function getLastComments(siteId: string, max: number): Promise<Comment[]> {
  return fetcher.get(`/last/${max}?site=${siteId}`);
}
