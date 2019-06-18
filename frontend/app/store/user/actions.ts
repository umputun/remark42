import api from '@app/common/api';
import { User, BlockedUser, AuthProvider, BlockTTL } from '@app/common/types';
import { ttlToTime } from '@app/utils/ttl-to-time';

import { StoreAction } from '../index';
import {
  USER_BAN,
  USER_SET,
  USER_UNBAN,
  USER_BANLIST_SET,
  USER_HIDELIST_SET_ACTION,
  USER_HIDELIST_SET,
  USER_HIDE_ACTION,
  USER_HIDE,
  USER_UNHIDE_ACTION,
  USER_UNHIDE,
  SETTINGS_VISIBLE_SET,
} from './types';
import { setComments, unsetCommentMode } from '../comments/actions';
import { setUserVerified as uSetUserVerified, filterTree } from '../comments/utils';
import { IS_STORAGE_AVAILABLE, LS_HIDDEN_USERS_KEY } from '@app/common/constants';
import { getItem } from '@app/common/local-storage';
import { Dispatch } from 'redux';

export const fetchUser = (): StoreAction<Promise<User | null>> => async dispatch => {
  const user = await api.getUser();
  dispatch({
    type: USER_SET,
    user,
  });
  return user;
};

export const logIn = (provider: AuthProvider): StoreAction<Promise<User | null>> => async dispatch => {
  const user = await api.logIn(provider);
  dispatch({
    type: USER_SET,
    user,
  });
  return user;
};

export const logout = (): StoreAction<Promise<void>> => async dispatch => {
  await api.logOut();
  dispatch(unsetCommentMode());
  dispatch({
    type: USER_SET,
    user: null,
  });
};

export const fetchBlockedUsers = (): StoreAction<Promise<BlockedUser[]>> => async dispatch => {
  const list = await api.getBlocked();
  dispatch({
    type: USER_BANLIST_SET,
    list,
  });
  return list;
};

export const blockUser = (
  id: User['id'],
  name: string,
  ttl: BlockTTL
): StoreAction<Promise<void>> => async dispatch => {
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

export const unblockUser = (id: User['id']): StoreAction<Promise<void>> => async dispatch => {
  await api.unblockUser(id);
  dispatch({
    type: USER_UNBAN,
    id,
  });
};

export const fetchHiddenUsers = (): StoreAction<void> => dispatch => {
  if (!IS_STORAGE_AVAILABLE) return;

  const hiddenUsers = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');
  return (dispatch as Dispatch<USER_HIDELIST_SET_ACTION>)({ type: USER_HIDELIST_SET, payload: hiddenUsers });
};

export const hideUser = (user: User): StoreAction<void> => (dispatch, getState) => {
  if (IS_STORAGE_AVAILABLE) {
    const hiddenUsers = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');
    hiddenUsers[user.id] = user;
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));
  }
  (dispatch as Dispatch<USER_HIDE_ACTION>)({ type: USER_HIDE, user });

  const comments = getState().comments;
  return dispatch(setComments(filterTree(comments, node => node.comment.user.id !== user.id)));
};

export const unhideUser = (userId: string): StoreAction<void> => dispatch => {
  if (IS_STORAGE_AVAILABLE) {
    const hiddenUsers = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');
    if (hiddenUsers.hasOwnProperty(userId)) {
      delete hiddenUsers[userId];
    }
    localStorage.setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));
  }
  return (dispatch as Dispatch<USER_UNHIDE_ACTION>)({ type: USER_UNHIDE, id: userId });
};

export const setVerifiedStatus = (id: User['id'], status: boolean): StoreAction<Promise<void>> => async (
  dispatch,
  getState
) => {
  if (status) {
    await api.setVerifyStatus(id);
  } else {
    await api.removeVerifyStatus(id);
  }
  const comments = getState().comments;
  dispatch(setComments(uSetUserVerified(comments, id, status)));
};

export const setSettingsVisibleState = (state: boolean): StoreAction<boolean> => dispatch => {
  dispatch({
    type: SETTINGS_VISIBLE_SET,
    state,
  });
  return state;
};
