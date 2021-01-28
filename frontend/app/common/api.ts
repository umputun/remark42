import { siteId, url } from './settings';
import { BASE_URL, API_BASE } from './constants';
import { Config, Comment, Tree, User, BlockedUser, Sorting, AuthProvider, BlockTTL, Image } from './types';
import fetcher from './fetcher';

/* common */
const __loginAnonymously = (username: string): Promise<User | null> => {
  const url = `/auth/anonymous/login?user=${encodeURIComponent(username)}&aud=${siteId}&from=${encodeURIComponent(
    `${window.location.origin}${window.location.pathname}?selfClose`
  )}`;
  return fetcher.get<User>({ url, withCredentials: true, overriddenApiBase: '' });
};

const __loginViaEmail = (token: string): Promise<User | null> => {
  const url = `/auth/email/login?token=${token}`;
  return fetcher.get<User>({ url, withCredentials: true, overriddenApiBase: '' });
};

/**
 * First step of two of `email` authorization
 *
 * @param username userrname
 * @param address email address
 */
export const sendEmailVerificationRequest = (username: string, address: string): Promise<void> => {
  const url = `/auth/email/login?id=${siteId}&user=${encodeURIComponent(username)}&address=${encodeURIComponent(
    address
  )}`;
  return fetcher.get({ url, withCredentials: true, overriddenApiBase: '' });
};

export const logIn = (provider: AuthProvider): Promise<User | null> => {
  if (provider.name === 'anonymous') return __loginAnonymously(provider.username);
  if (provider.name === 'email') return __loginViaEmail(provider.token);

  return new Promise<User | null>((resolve, reject) => {
    const url = `${BASE_URL}/auth/${provider.name}/login?from=${encodeURIComponent(
      `${window.location.origin}${window.location.pathname}?selfClose`
    )}&site=${siteId}`;

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

export const logOut = (): Promise<void> =>
  fetcher.get({ url: `/auth/logout`, overriddenApiBase: '', withCredentials: true });

export const getConfig = (): Promise<Config> => fetcher.get(`/config`);

export const getPostComments = (sort: Sorting) =>
  fetcher.get<Tree>({
    url: `/find?site=${siteId}&url=${url}&sort=${sort}&format=tree`,
    withCredentials: true,
  });

export const getCommentsCount = (siteId: string, urls: string[]): Promise<{ url: string; count: number }[]> =>
  fetcher.post({
    url: `/counts?site=${siteId}`,
    body: urls,
  });

export const getComment = (id: Comment['id']): Promise<Comment> =>
  fetcher.get({ url: `/id/${id}?url=${url}`, withCredentials: true });

export const getUserComments = (
  userId: User['id'],
  limit: number
): Promise<{
  comments: Comment[];
  count: number;
}> =>
  fetcher.get({
    url: `/comments?user=${userId}&limit=${limit}`,
    withCredentials: true,
  });

export const putCommentVote = ({ id, value }: { id: Comment['id']; value: number }): Promise<void> =>
  fetcher.put({
    url: `/vote/${id}?url=${url}&vote=${value}`,
    withCredentials: true,
  });

export const addComment = ({
  title,
  text,
  pid,
}: {
  title: string;
  text: string;
  pid?: Comment['id'];
}): Promise<Comment> =>
  fetcher.post({
    url: '/comment',
    body: {
      title,
      text,
      locator: {
        site: siteId,
        url,
      },
      ...(pid ? { pid } : {}),
    },
    withCredentials: true,
  });

export const updateComment = ({ text, id }: { text: string; id: Comment['id'] }): Promise<Comment> =>
  fetcher.put({
    url: `/comment/${id}?url=${url}`,
    body: {
      text,
    },
    withCredentials: true,
  });

export const getPreview = (text: string): Promise<string> =>
  fetcher.post({
    url: '/preview',
    body: {
      text,
    },
    withCredentials: true,
  });

export const getUser = (): Promise<User | null> =>
  fetcher
    .get<User | null>({
      url: '/user',
      withCredentials: true,
      logError: false,
    })
    .catch(() => null);

/* GDPR */

export const deleteMe = (): Promise<{
  user_id: string;
  link: string;
}> =>
  fetcher.post({
    url: `/deleteme?site=${siteId}`,
    withCredentials: true,
  });

export const approveDeleteMe = (token: string): Promise<void> =>
  fetcher.get({
    url: `/admin/deleteme?token=${token}`,
    withCredentials: true,
  });

/* admin */
export const pinComment = (id: Comment['id']): Promise<void> =>
  fetcher.put({
    url: `/admin/pin/${id}?url=${url}&pin=1`,
    withCredentials: true,
  });

export const unpinComment = (id: Comment['id']): Promise<void> =>
  fetcher.put({
    url: `/admin/pin/${id}?url=${url}&pin=0`,
    withCredentials: true,
  });

export const setVerifiedStatus = (id: User['id']): Promise<void> =>
  fetcher.put({
    url: `/admin/verify/${id}?verified=1`,
    withCredentials: true,
  });

export const removeVerifiedStatus = (id: User['id']): Promise<void> =>
  fetcher.put({
    url: `/admin/verify/${id}?verified=0`,
    withCredentials: true,
  });

export const removeComment = (id: Comment['id']) =>
  fetcher.delete({
    url: `/admin/comment/${id}?url=${url}`,
    withCredentials: true,
  });

export const removeMyComment = (id: Comment['id']): Promise<void> =>
  fetcher.put({
    url: `/comment/${id}?url=${url}`,
    body: {
      delete: true,
    } as object,
    withCredentials: true,
  });

export const blockUser = (
  id: User['id'],
  ttl: BlockTTL
): Promise<{
  block: boolean;
  site_id: string;
  user_id: string;
}> =>
  fetcher.put({
    url: ttl === 'permanently' ? `/admin/user/${id}?block=1` : `/admin/user/${id}?block=1&ttl=${ttl}`,
    withCredentials: true,
  });

export const unblockUser = (
  id: User['id']
): Promise<{
  block: boolean;
  site_id: string;
  user_id: string;
}> =>
  fetcher.put({
    url: `/admin/user/${id}?block=0`,
    withCredentials: true,
  });

export const getBlocked = (): Promise<BlockedUser[] | null> =>
  fetcher.get({
    url: '/admin/blocked',
    withCredentials: true,
  });

export const disableComments = (): Promise<void> =>
  fetcher.put({
    url: `/admin/readonly?site=${siteId}&url=${url}&ro=1`,
    withCredentials: true,
  });

export const enableComments = (): Promise<void> =>
  fetcher.put({
    url: `/admin/readonly?site=${siteId}&url=${url}&ro=0`,
    withCredentials: true,
  });

export const uploadImage = (image: File): Promise<Image> => {
  const data = new FormData();
  data.append('file', image);

  return fetcher
    .post<{ id: string }>({
      url: `/picture`,
      withCredentials: true,
      contentType: 'multipart/form-data',
      body: data,
    })
    .then((resp) => ({
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
  fetcher.post({
    url: `/email/subscribe?site=${siteId}&address=${encodeURIComponent(emailAddress)}`,
    withCredentials: true,
  });

/**
 * Confirmation of email subscription to updates
 * @param token confirmation token from email
 */
export const emailConfirmationForSubscribe = (token: string) =>
  fetcher.post({ url: `/email/confirm?site=${siteId}&tkn=${encodeURIComponent(token)}`, withCredentials: true });

/**
 * Decline current subscription to updates
 */
export const unsubscribeFromEmailUpdates = () => fetcher.delete({ url: `/email`, withCredentials: true });
