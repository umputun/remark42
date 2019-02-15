import { createStore, applyMiddleware } from 'redux';
import { combineReducers } from 'redux';
import thunk, { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { Comment, User, PostInfo, Node, BlockedUser, Theme, Sorting } from '@app/common/types';

import storeReducers from './reducers';
import { COMMENTS_ACTIONS } from './comments/types';
import { POST_INFO_ACTIONS } from './post_info/types';
import { SORT_ACTIONS } from './sort/types';
import { THEME_ACTIONS } from './theme/types';
import { THREAD_ACTIONS } from './thread/types';
import { USER_ACTIONS } from './user/types';
import { USER_INFO_ACTIONS } from './user-info/types';

export type ACTIONS =
  | COMMENTS_ACTIONS
  | POST_INFO_ACTIONS
  | SORT_ACTIONS
  | THEME_ACTIONS
  | THREAD_ACTIONS
  | USER_ACTIONS
  | USER_INFO_ACTIONS;

export interface StoreState {
  sort: Sorting;
  comments: Node[];
  pinnedComments: Comment[];
  user: User | null;
  theme: Theme;
  info: PostInfo;
  bannedUsers: BlockedUser[];
  isBlockedVisible: boolean;
  collapsedThreads: {
    [key: string]: boolean;
  };
  /** used in user comments widget */
  userComments?: {
    [key: string]: Comment[];
  };
}

const reducers = combineReducers(storeReducers);
const middleware = applyMiddleware(thunk);

/**
 * Thunk Action shortcut
 */
export type StoreAction<R> = ThunkAction<R, StoreState, undefined, ACTIONS>;

/**
 * Thunk Dispatch shortcut
 */
export type StoreDispatch = ThunkDispatch<StoreState, undefined, ACTIONS>;

export default createStore(reducers, middleware);
