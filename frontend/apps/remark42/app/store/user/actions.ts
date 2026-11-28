import * as api from 'common/api';
import { dropSession } from 'common/fetcher';
import { logout } from 'components/auth/auth.api';
import type { User, BlockedUser, BlockTTL } from 'common/types';
import { ttlToTime } from 'utils/ttl-to-time';
import { getHiddenUsers } from 'utils/get-hidden-users';
import { LS_HIDDEN_USERS_KEY } from 'common/constants';
import { setItem } from 'common/local-storage';

import type { StoreAction } from '../index';
import type { USER_SET_ACTION } from './types';
import {
  USER_BAN,
  USER_SET,
  USER_UNBAN,
  USER_BANLIST_SET,
  USER_HIDELIST_SET,
  USER_HIDE,
  USER_UNHIDE,
  USER_SUBSCRIPTION_SET,
} from './types';
import { reloadComments, unsetCommentMode } from '../comments/actions';
import { COMMENTS_PATCH } from '../comments/types';
import { resetPersistedEmail } from 'components/auth/auth.utils';

export function setUser(user: User | null = null): USER_SET_ACTION {
  return {
    type: USER_SET,
    user,
  };
}

/**
 * Loads the comments again for a reader who has just signed in or out, since votes and the
 * actions offered on a comment depend on who reads it. Nothing waits for it: the reader has
 * changed either way, and a failed request leaves the comments loaded before it on screen
 */
const reloadCommentsForReader = (): StoreAction => (dispatch) => {
  dispatch(reloadComments()).catch((e: unknown) => {
    console.error('Remark42: failed to reload comments', e);
  });
};

// tells the other widget frames of this site, on this page or in another tab, that the reader
// signed in ('signin') or out ('signout') here. They share the session but show the reader they
// loaded with
let readerChannel: BroadcastChannel | null = null;
// counts the notices, so only the answer to the newest one is shown
let readerProbe = 0;

/** Makes this frame show the reader that signed in or out in another one, and tell the others */
export const followReaderAcrossFrames = (): StoreAction => (dispatch, getState) => {
  if (typeof BroadcastChannel !== 'function') {
    return;
  }
  try {
    readerChannel = new BroadcastChannel('remark42-reader');
  } catch {
    // Firefox refuses it to a frame it denies storage, and the widget works on without it
    return;
  }
  readerChannel.onmessage = (event: MessageEvent) => {
    // a sign-out elsewhere ends the session here too, the copy of the token this frame holds included
    if (event.data === 'signout') {
      dropSession();
    }
    const probe = ++readerProbe;
    api.getUser().then(
      (user) => {
        // an earlier notice's answer can come back after a later one's
        if (probe !== readerProbe) {
          return;
        }
        const before = getState().user?.id;
        dispatch(setUser(user));
        if (user?.id !== before) {
          dispatch(unsetCommentMode());
          dispatch(reloadCommentsForReader());
        }
      },
      () => undefined
    );
  };
};

/**
 * Ends the session and then shows the reader as signed out. The same for the button above the
 * comments and for a sign-out asked for in the profile: the profile is a document of its own, which
 * is removed as it closes, and it leaves the request to this one
 */
export function signout(): StoreAction<Promise<void>> {
  return async (dispatch) => {
    await logout();
    resetPersistedEmail();
    dispatch(setUser());
    dispatch(unsetCommentMode());
    dispatch(reloadCommentsForReader());
    readerChannel?.postMessage('signout');
  };
}

export const fetchUser = (): StoreAction<Promise<User | null>> => async (dispatch) => {
  const user = await api.getUser();
  dispatch(setUser(user));
  return user;
};

export const signin =
  (user: User): StoreAction<Promise<void>> =>
  async (dispatch) => {
    dispatch(setUser(user));
    dispatch(reloadCommentsForReader());
    readerChannel?.postMessage('signin');
  };

export const fetchBlockedUsers = (): StoreAction<Promise<BlockedUser[]>> => async (dispatch) => {
  const list = (await api.getBlocked()) || [];

  dispatch({ type: USER_BANLIST_SET, list });

  return list;
};

export const blockUser =
  (id: User['id'], name: string, ttl: BlockTTL): StoreAction<Promise<void>> =>
  async (dispatch) => {
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

export const unblockUser =
  (id: User['id']): StoreAction<Promise<void>> =>
  async (dispatch, getState) => {
    await api.unblockUser(id);
    dispatch({ type: USER_UNBAN, id });
    const comments = Object.values(getState().comments.allComments);
    const userComments = comments.filter((comment) => comment.user.id === id);

    if (!userComments.length) return;
    const user = userComments[0].user;

    dispatch({
      type: COMMENTS_PATCH,
      ids: userComments.map((c) => c.id),
      patch: { user: { ...user, block: false } },
    });
  };

export const fetchHiddenUsers = (): StoreAction => (dispatch) => {
  const hiddenUsers = getHiddenUsers();

  dispatch({ type: USER_HIDELIST_SET, payload: hiddenUsers });
};

export const hideUser =
  (user: User): StoreAction =>
  (dispatch, getState) => {
    const hiddenUsers = getHiddenUsers();

    hiddenUsers[user.id] = user;
    setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));

    const ids = Object.values(getState().comments.allComments)
      .filter((c) => c.user.id === user.id)
      .map((c) => c.id);

    dispatch({ type: USER_HIDE, user });
    dispatch({ type: COMMENTS_PATCH, ids, patch: { hidden: true } });
  };

export const unhideUser =
  (userId: string): StoreAction =>
  (dispatch, _getState) => {
    const hiddenUsers = getHiddenUsers();

    if (Object.prototype.hasOwnProperty.call(hiddenUsers, userId)) {
      delete hiddenUsers[userId];
    }

    setItem(LS_HIDDEN_USERS_KEY, JSON.stringify(hiddenUsers));
    dispatch({ type: USER_UNHIDE, id: userId });

    // no need for comments patch as comments will be refetched after action
  };

export const setVerifiedStatus =
  (id: User['id'], status: boolean): StoreAction<Promise<void>> =>
  async (dispatch, getState) => {
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
