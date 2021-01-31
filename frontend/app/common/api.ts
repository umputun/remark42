import { siteId, url } from './settings';
import { BASE_URL, API_BASE } from './constants';
import { Config, Comment, Tree, User, BlockedUser, Sorting, AuthProvider, BlockTTL, Image } from './types';
import fetcher, { stringifyUrl } from './fetcher';

/* common */
const __loginAnonymously = (username: string): Promise<User | null> =>
  fetcher.get<User>('/auth/anonymous/login', {
    user: username,
    aud: siteId,
    from: `${window.location.origin}${window.location.pathname}?selfClose`,
  });

const __loginViaEmail = (token: string): Promise<User | null> => fetcher.get<User>('/auth/email/login', { token });

/**
 * First step of two of `email` authorization
 *
 * @param username userrname
 * @param address email address
 */
export const sendEmailVerificationRequest = (username: string, address: string): Promise<void> =>
  fetcher.get('/auth/email/login', { id: siteId, user: username, address });

export const logIn = (provider: AuthProvider): Promise<User | null> => {
  if (provider.name === 'anonymous') return __loginAnonymously(provider.username);
  if (provider.name === 'email') return __loginViaEmail(provider.token);

  return new Promise<User | null>((resolve, reject) => {
    const url = stringifyUrl(`/auth/${provider.name}/login`, {
      from: `${window.location.origin}${window.location.pathname}?selfClose`,
    });
    const newWindow = window.open(url);
    let secondsPass = 0;
    const checkMsDelay = 300;
    const checkInterval = setInterval(() => {
      let shouldProceed;
      secondsPass += checkMsDelay;
      try {
        shouldProceed = (newWindow && newWindow.closed) || secondsPass > 30000;
      } catch (e) {}

      if (shouldProceed) {
        clearInterval(checkInterval);

        getUser()
          .then((user) => {
            resolve(user);
          })
          .catch(() => {
            reject(new Error('User logIn Error'));
          });
      }
    }, checkMsDelay);
  });
};

export const logOut = (): Promise<void> => fetcher.get('/auth/logout');

export const getConfig = (): Promise<Config> => fetcher.get('/config');

export const getPostComments = (sort: Sorting) => fetcher.get<Tree>('/find', { url, sort, format: 'tree' });

export const getComment = (id: Comment['id']): Promise<Comment> => fetcher.get(`/id/${id}`, { url });

export const getUserComments = (
  userId: User['id'],
  limit: number
): Promise<{
  comments: Comment[];
  count: number;
}> => fetcher.get('/comments', { user: userId, limit });

export const putCommentVote = ({ id, value }: { id: Comment['id']; value: number }): Promise<void> =>
  fetcher.put(`/vote/${id}`, { url, vote: value });

export const addComment = ({
  title,
  text,
  pid,
}: {
  title: string;
  text: string;
  pid?: Comment['id'];
}): Promise<Comment> =>
  fetcher.post(
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
  fetcher.put(`/comment/${id}`, { url }, { text });

export const getPreview = (text: string): Promise<string> => fetcher.post('/preview', {}, { text });

export const getUser = (): Promise<User | null> => fetcher.get<User | null>('/user').catch(() => null);

/* GDPR */

export const deleteMe = (): Promise<{ user_id: string; link: string }> => fetcher.post('/deleteme');

export const approveDeleteMe = (token: string): Promise<void> => fetcher.get('/admin/deleteme', { token });

/* admin */
export const pinComment = (id: Comment['id']): Promise<void> => fetcher.put(`/admin/pin/${id}`, { url, pin: 1 });

export const unpinComment = (id: Comment['id']): Promise<void> => fetcher.put(`/admin/pin/${id}`, { url, pin: 0 });

export const setVerifiedStatus = (id: User['id']): Promise<void> => fetcher.put(`/admin/verify/${id}`, { verified: 1 });

export const removeVerifiedStatus = (id: User['id']): Promise<void> =>
  fetcher.put(`/admin/verify/${id}`, { verified: 0 });

export const removeComment = (id: Comment['id']): Promise<void> => fetcher.delete(`/admin/comment/${id}`, { url });

export const removeMyComment = (id: Comment['id']): Promise<void> =>
  fetcher.put(`/comment/${id}`, { url }, { delete: true });

export const blockUser = (
  id: User['id'],
  ttl: BlockTTL
): Promise<{
  block: boolean;
  site_id: string;
  user_id: string;
}> => fetcher.put(`/admin/user/${id}`, { block: 1, ttl: ttl === 'permanently' ? ttl : undefined });

export const unblockUser = (
  id: User['id']
): Promise<{
  block: boolean;
  site_id: string;
  user_id: string;
}> => fetcher.put(`/admin/user/${id}`, { block: 0 });

export const getBlocked = (): Promise<BlockedUser[] | null> => fetcher.get('/admin/blocked');

export const disableComments = (): Promise<void> => fetcher.put('/admin/readonly', { url, ro: 1 });

export const enableComments = (): Promise<void> => fetcher.put('/admin/readonly', { url, ro: 0 });

export const uploadImage = (image: File): Promise<Image> => {
  const data = new FormData();
  data.append('file', image);

  return fetcher.post<{ id: string }>('/picture', {}, data).then((resp) => ({
    name: image.name,
    size: image.size,
    type: image.type,
    url: `${BASE_URL + API_BASE}/picture/${resp.id}`,
  }));
};

/**
 * Start process of email subscription to updates
 * @param emailAddress email for subscription
 */
export const emailVerificationForSubscribe = (emailAddress: string) =>
  fetcher.post('/email/subscribe', { address: emailAddress });

/**
 * Confirmation of email subscription to updates
 * @param token confirmation token from email
 */
export const emailConfirmationForSubscribe = (token: string) => fetcher.post('/email/confirm', { tkn: token });

/**
 * Decline current subscription to updates
 */
export const unsubscribeFromEmailUpdates = () => fetcher.delete('/email');
