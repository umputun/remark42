import { siteId, url } from './settings';

import fetcher from './fetcher';

// TODO: rename actions

/* common */

export const logOut = () => fetcher.get({ url: `/auth/logout`, overriddenApiBase: '' });

export const getConfig = () => fetcher.get(`/config`);

// TODO: looks like we can get url from settings here and below
export const getPostComments = ({ sort, url }) => fetcher.get(`/find?url=${url}&sort=${sort}&format=tree`);

export const getLastComments = ({ siteId, max }) => fetcher.get(`/last/${max}?site=${siteId}`);

export const getCommentsCount = ({ urls, siteId }) => fetcher.post({
  url: `/counts?site=${siteId}`,
  body: urls,
});

export const getComment = ({ id }) => fetcher.get(`/id/${id}?url=${url}`);

export const getUserComments = ({ user, limit }) => fetcher.get(`/comments?user=${user}&limit=${limit}`);

export const putCommentVote = ({ id, url, value }) => fetcher.put({
  url: `/vote/${id}?url=${url}&vote=${value}`,
  withCredentials: true,
});

export const addComment = ({ text, pid }) => fetcher.post({
  url: '/comment',
  body: {
    text,
    locator: {
      site: siteId,
      url,
    },
    ...(pid ? { pid } : {}),
  },
  withCredentials: true,
});

export const updateComment = ({ text, id }) => fetcher.put({
  url: `/comment/${id}?url=${url}`,
  body: {
    text,
  },
  withCredentials: true,
});

export const getPreview = ({ text }) => fetcher.post({
  url: '/preview',
  body: {
    text,
  },
  withCredentials: true,
});

export const getUser = () => fetcher.get({
  url: '/user',
  withCredentials: true
});

/* admin */
export const pinComment = ({ id, url }) => fetcher.put({
  url: `/admin/pin/${id}?url=${url}&pin=1`,
  withCredentials: true,
});

export const unpin = ({ id, url }) => fetcher.put({
  url: `/admin/pin/${id}?url=${url}&pin=0`,
  withCredentials: true,
});

export const verify = ({ id }) => fetcher.put({
  url: `/admin/verify/${id}?verified=1`,
  withCredentials: true,
});

export const unverify = ({ id }) => fetcher.put({
  url: `/admin/verify/${id}?verified=0`,
  withCredentials: true,
});

export const remove = ({ id }) => fetcher.delete({
  url: `/admin/comment/${id}?url=${url}`,
  withCredentials: true,
});

export const blockUser = ({ id }) => fetcher.put({
  url: `/admin/user/${id}?block=1`,
  withCredentials: true,
});

export const unblockUser = ({ id }) => fetcher.put({
  url: `/admin/user/${id}?block=0`,
  withCredentials: true,
});

export const getBlocked = () => fetcher.get({
  url: '/admin/blocked',
  withCredentials: true,
});

export default {
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
  getUser,
  getPreview,

  pinComment,
  unpin,
  verify,
  unverify,
  remove,
  blockUser,
  unblockUser,
  getBlocked,
};
