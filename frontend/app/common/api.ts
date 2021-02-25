import { siteId, url } from './settings';
import { BASE_URL, API_BASE } from './constants';
import { Config, Comment, Tree, User, BlockedUser, Sorting, BlockTTL, Image } from './types';
import { apiFetcher, adminFetcher } from './fetcher';

/* API methods */

export const getConfig = (): Promise<Config> => apiFetcher.get('/config');

export const getPostComments = (sort: Sorting) => apiFetcher.get<Tree>('/find', { url, sort, format: 'tree' });

export const getComment = (id: Comment['id']): Promise<Comment> => apiFetcher.get(`/id/${id}`, { url });

export const getUserComments = (
  userId: User['id'],
  limit: number
): Promise<{
  comments: Comment[];
  count: number;
}> => apiFetcher.get('/comments', { user: userId, limit });

export const putCommentVote = ({ id, value }: { id: Comment['id']; value: number }): Promise<void> =>
  apiFetcher.put(`/vote/${id}`, { url, vote: value });

export const addComment = ({
  title,
  text,
  pid,
}: {
  title: string;
  text: string;
  pid?: Comment['id'];
}): Promise<Comment> =>
  apiFetcher.post(
    '/comment',
    {},
    {
      title,
      text,
      locator: { site: siteId, url },
      ...(pid ? { pid } : {}),
    }
  );

export const updateComment = ({ text, id }: { text: string; id: Comment['id'] }): Promise<Comment> =>
  apiFetcher.put(`/comment/${id}`, { url }, { text });

export const removeMyComment = (id: Comment['id']): Promise<void> =>
  apiFetcher.put(`/comment/${id}`, { url }, { delete: true });

export const getPreview = (text: string): Promise<string> => apiFetcher.post('/preview', {}, { text });

export const getUser = (): Promise<User | null> => apiFetcher.get<User | null>('/user').catch(() => null);

export const uploadImage = (image: File): Promise<Image> => {
  const data = new FormData();
  data.append('file', image);

  return apiFetcher.post<{ id: string }>('/picture', {}, data).then((resp) => ({
    name: image.name,
    size: image.size,
    type: image.type,
    url: `${BASE_URL + API_BASE}/picture/${resp.id}`,
  }));
};

/* Subscription methods */

/**
 * Start process of email subscription to updates
 * @param emailAddress email for subscription
 */
export const emailVerificationForSubscribe = (emailAddress: string) =>
  apiFetcher.post('/email/subscribe', { address: emailAddress });

/**
 * Confirmation of email subscription to updates
 * @param token confirmation token from email
 */
export const emailConfirmationForSubscribe = (token: string) => apiFetcher.post('/email/confirm', { tkn: token });

/**
 * Decline current subscription to updates
 */
export const unsubscribeFromEmailUpdates = () => apiFetcher.delete('/email');

/* GDPR Methods */

export const deleteMe = (): Promise<{ user_id: string; link: string }> => apiFetcher.post('/deleteme');

/* Admin Methods */
// TODO: move these methods to separate chunk as well as all admin inteface features
export const approveDeleteMe = (token: string): Promise<void> => adminFetcher.get('/deleteme', { token });

export const pinComment = (id: Comment['id']): Promise<void> => adminFetcher.put(`/pin/${id}`, { url, pin: 1 });

export const unpinComment = (id: Comment['id']): Promise<void> => adminFetcher.put(`/pin/${id}`, { url, pin: 0 });

export const setVerifiedStatus = (id: User['id']): Promise<void> => adminFetcher.put(`/verify/${id}`, { verified: 1 });

export const removeVerifiedStatus = (id: User['id']): Promise<void> =>
  adminFetcher.put(`/verify/${id}`, { verified: 0 });

export const removeComment = (id: Comment['id']): Promise<void> => adminFetcher.delete(`/comment/${id}`, { url });

export const blockUser = (
  id: User['id'],
  ttl: BlockTTL
): Promise<{
  block: boolean;
  site_id: string;
  user_id: string;
}> => adminFetcher.put(`/user/${id}`, { block: 1, ttl: ttl === 'permanently' ? ttl : undefined });

export const unblockUser = (
  id: User['id']
): Promise<{
  block: boolean;
  site_id: string;
  user_id: string;
}> => adminFetcher.put(`/user/${id}`, { block: 0 });

export const getBlocked = (): Promise<BlockedUser[] | null> => adminFetcher.get('/blocked');

export const disableComments = (): Promise<void> => adminFetcher.put('/readonly', { url, ro: 1 });

export const enableComments = (): Promise<void> => adminFetcher.put('/readonly', { url, ro: 0 });
