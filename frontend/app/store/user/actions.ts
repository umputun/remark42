import * as api from 'common/api';
import { User, BlockedUser, BlockTTL } from 'common/types';
import { ttlToTime } from 'utils/ttl-to-time';
import getHiddenUsers from 'utils/get-hidden-users';
import { LS_HIDDEN_USERS_KEY } from 'common/constants';
import { setItem } from 'common/local-storage';

import { StoreAction } from '../index';
import {
  USER_BAN,
  USER_SET,
  USER_UNBAN,
  USER_BANLIST_SET,
  USER_HIDELIST_SET,
  USER_HIDE,
  USER_UNHIDE,
  USER_SUBSCRIPTION_SET,
  USER_SET_ACTION,
} from './types';
import { fetchComments } from '../comments/actions';
import { COMMENTS_PATCH } from '../comments/types';

export function setUser(user: User | null = null): USER_SET_ACTION {
  return {
    type: USER_SET,
    user,
  };
}

export const fetchUser = (): StoreAction<Promise<User | null>> => async (dispatch) => {
  const user = await api.getUser();
  dispatch(setUser(user));
  return user;
};

export const signin = (user: User): StoreAction<Promise<void>> => async (dispatch) => {
  dispatch(setUser(user));
  dispatch(fetchComments());
};

export const fetchBlockedUsers = (): StoreAction<Promise<BlockedUser[]>> => async (dispatch) => {
  const list = (await api.getBlocked()) || [];

  dispatch({ type: USER_BANLIST_SET, list });

  return list;
};

export const blockUser = (id: User['id'], name: string, ttl: BlockTTL): StoreAction<Promise<void>> => async (
  dispatch
) => {
  await api.blockUser(id, ttl);
  dispatch({
    type: USER_BAN,
    user: {
      id,
      name,
      time: ttlToTime(ttl),
    },
  });
};

export const unblockUser = (id: User['id']): StoreAction<Promise<void>> => async (dispatch, getState) => {
  await api.unblockUser(id);
  dispatch({ type: USER_UNBAN, id });
  const comments = Object.values(getState().comments.allComments);
  const userComments = comments.filter((comment) => comment.user.id === id);

  if (!userComments.length) return;
  const user = comments[0].user;

  dispatch({
    type: COMMENTS_PATCH,
    ids: userComments.map((c) => c.id),
    patch: { user: { ...user, block: false } },
  });
};

export const fetchHiddenUsers = (): StoreAction<void> => (dispatch) => {
  const hiddenUsers = getHiddenUsers();

  dispatch({ type: USER_HIDELIST_SET, payload: hiddenUsers });
};

export const hideUser = (user: User): StoreAction<void> => (dispatch, getState) => {
  const hiddenUsers = getHiddenUsers();

  hiddenUsers[user.id] = user;
  setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));

  const ids = Object.values(getState().comments.allComments)
    .filter((c) => c.user.id === user.id)
    .map((c) => c.id);

  dispatch({ type: USER_HIDE, user });
  dispatch({ type: COMMENTS_PATCH, ids, patch: { hidden: true } });
};

export const unhideUser = (userId: string): StoreAction<void> => (dispatch, _getState) => {
  const hiddenUsers = getHiddenUsers();

  if (Object.prototype.hasOwnProperty.call(hiddenUsers, userId)) {
    delete hiddenUsers[userId];
  }

  setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));
  dispatch({ type: USER_UNHIDE, id: userId });

  // no need for comments patch as comments will be refetched after action
};

export const setVerifiedStatus = (id: User['id'], status: boolean): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  if (status) {
    await api.setVerifiedStatus(id);
  } else {
    await api.removeVerifiedStatus(id);
  }
  const comments = Object.values(getState().comments.allComments);
  const userComments = comments.filter((c) => c.user.id === id);

  if (!userComments.length) return;
  const user = userComments[0].user;

  dispatch({
    type: COMMENTS_PATCH,
    ids: userComments.map((c) => c.id),
    patch: { user: { ...user, verified: status } },
  });
};

export const setUserSubscribed = (isSubscribed: boolean) => ({
  type: USER_SUBSCRIPTION_SET,
  payload: isSubscribed,
});
