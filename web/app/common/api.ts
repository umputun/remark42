import { siteId, url } from './settings';
import { BASE_URL } from './constants';
import { Config, Comment, Tree, User, BlockedUser, Sorting, Provider, BlockTTL } from './types';
import fetcher from './fetcher';

/* common */

export const logIn = (provider: Provider) => {
  return new Promise<User | null>((resolve, reject) => {
    const newWindow = window.open(
      `${BASE_URL}/auth/${provider}/login?from=${encodeURIComponent(
        location.origin + location.pathname + '?selfClose'
      )}&site=${siteId}`
    );

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
          .then(user => {
            resolve(user);
          })
          .catch(() => {
            reject(new Error('User logIn Error'));
          });
      }
    }, checkMsDelay);
  });
};

export const logOut = (): Promise<void> => fetcher.get({ url: `/auth/logout`, overriddenApiBase: '' }) as Promise<void>;

export const getConfig = (): Promise<Config> => fetcher.get(`/config`) as Promise<Config>;

// TODO: looks like we can get url from settings here and below
export const getPostComments = (sort: Sorting): Promise<Tree> =>
  fetcher.get(`/find?site=${siteId}&url=${url}&sort=${sort}&format=tree`) as Promise<Tree>;

export const getLastComments = ({ siteId, max }: { siteId: string; max: number }): Promise<Comment[]> =>
  fetcher.get(`/last/${max}?site=${siteId}`) as Promise<Comment[]>;

export const getCommentsCount = (siteId: string, urls: string[]) =>
  fetcher.post({
    url: `/counts?site=${siteId}`,
    body: urls,
  }) as Promise<{ url: string; count: number }[]>;

export const getComment = ({ id }: { id: Comment['id'] }) => fetcher.get(`/id/${id}?url=${url}`) as Promise<Comment>;

export const getUserComments = ({ userId, limit }: { userId: User['id']; limit: number }) =>
  fetcher.get(`/comments?user=${userId}&limit=${limit}`) as Promise<{
    comments: Comment[];
    count: number;
  }>;

export const putCommentVote = ({ id, value }: { id: Comment['id']; value: number }) =>
  fetcher.put({
    url: `/vote/${id}?url=${url}&vote=${value}`,
    withCredentials: true,
  }) as Promise<void>;

export const addComment = ({ title, text, pid }: { title: string; text: string; pid?: Comment['id'] }) =>
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
  }) as Promise<Comment>;

export const updateComment = ({ text, id }: { text: string; id: Comment['id'] }) =>
  fetcher.put({
    url: `/comment/${id}?url=${url}`,
    body: {
      text,
    },
    withCredentials: true,
  }) as Promise<Comment>;

export const getPreview = (text: string) =>
  fetcher.post({
    url: '/preview',
    body: {
      text,
    },
    withCredentials: true,
  }) as Promise<string>;

export const getUser = () =>
  fetcher
    .get({
      url: '/user',
      withCredentials: true,
    })
    .catch(() => null) as Promise<User | null>;

/* GDPR */

export const deleteMe = () =>
  fetcher.post({
    url: `/deleteme?site=${siteId}`,
  }) as Promise<{
    user_id: string;
    link: string;
  }>;

export const approveDeleteMe = (token: string) =>
  fetcher.get({
    url: `/admin/deleteme?token=${token}`,
  }) as Promise<void>;

/* admin */
export const pinComment = (id: Comment['id']) =>
  fetcher.put({
    url: `/admin/pin/${id}?url=${url}&pin=1`,
    withCredentials: true,
  }) as Promise<void>;

export const unpinComment = (id: Comment['id']) =>
  fetcher.put({
    url: `/admin/pin/${id}?url=${url}&pin=0`,
    withCredentials: true,
  }) as Promise<void>;

export const setVerifyStatus = ({ id }: { id: User['id'] }) =>
  fetcher.put({
    url: `/admin/verify/${id}?verified=1`,
    withCredentials: true,
  }) as Promise<void>;

export const removeVerifyStatus = ({ id }: { id: User['id'] }) =>
  fetcher.put({
    url: `/admin/verify/${id}?verified=0`,
    withCredentials: true,
  }) as Promise<void>;

export const removeComment = ({ id }: { id: Comment['id'] }) =>
  fetcher.delete({
    url: `/admin/comment/${id}?url=${url}`,
    withCredentials: true,
  });

export const removeMyComment = ({ id }: { id: Comment['id'] }) =>
  fetcher.put({
    url: `/comment/${id}?url=${url}`,
    body: {
      delete: true,
    },
    withCredentials: true,
  }) as Promise<void>;

export const blockUser = ({ id, ttl }: { id: User['id']; ttl: BlockTTL }) =>
  fetcher.put({
    url: ttl === 'permanently' ? `/admin/user/${id}?block=1` : `/admin/user/${id}?block=1&ttl=${ttl}`,
    withCredentials: true,
  }) as Promise<{
    block: boolean;
    site_id: string;
    user_id: string;
  }>;

export const unblockUser = ({ id }: { id: User['id'] }) =>
  fetcher.put({
    url: `/admin/user/${id}?block=0`,
    withCredentials: true,
  }) as Promise<{
    block: boolean;
    site_id: string;
    user_id: string;
  }>;

export const getBlocked = () =>
  fetcher.get({
    url: '/admin/blocked',
    withCredentials: true,
  }) as Promise<BlockedUser[]>;

export const disableComments = () =>
  fetcher.put({
    url: `/admin/readonly?site=${siteId}&url=${url}&ro=1`,
    withCredentials: true,
  }) as Promise<void>;

export const enableComments = () =>
  fetcher.put({
    url: `/admin/readonly?site=${siteId}&url=${url}&ro=0`,
    withCredentials: true,
  }) as Promise<void>;

export default {
  logIn,
  logOut,
  getConfig,
  getPostComments,
  getLastComments,
  getCommentsCount,
  getComment,
  getUserComments,
  putCommentVote,
  addComment,
  updateComment,
  removeMyComment,
  getUser,
  getPreview,

  pinComment,
  unpinComment,
  setVerifyStatus,
  removeVerifyStatus,
  removeComment,
  blockUser,
  unblockUser,
  getBlocked,
  disableComments,
  enableComments,
};
