import { createStore, applyMiddleware } from 'redux';
import { combineReducers } from 'redux';
import thunk, { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { Comment, User, PostInfo, Node, BlockedUser, Theme, Sorting, CommentMode } from '@app/common/types';

import storeReducers from './reducers';
import { ACTIONS } from './actions';

export interface StoreState {
  sort: Sorting;
  comments: Node[];
  pinnedComments: Comment[];
  /** Defines comment that is in reply or edit mode */
  activeComment: null | { id: Comment['id']; state: CommentMode };
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
