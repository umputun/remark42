import { createStore, applyMiddleware, AnyAction, compose } from 'redux';
import { combineReducers } from 'redux';
import thunk, { ThunkAction, ThunkDispatch } from 'redux-thunk';
import { Comment, User, PostInfo, Node, BlockedUser, Theme, Sorting, CommentMode } from '@app/common/types';
import { ProviderState } from './provider/reducers';

import storeReducers from './reducers';
import { ACTIONS } from './actions';

export interface StoreState {
  /** Comments sort */
  sort: Sorting;
  /** Comments list */
  comments: Node[];
  /** List of pinned comments */
  pinnedComments: Comment[];
  /** Defines comment that is in reply or edit mode */
  activeComment: null | { id: Comment['id']; state: CommentMode };
  /** Logged in user */
  user: User | null;
  /** Remark's styling theme */
  theme: Theme;
  /** Current post information */
  info: PostInfo;
  /** List of banned users */
  bannedUsers: BlockedUser[];
  /** List of hidden users */
  hiddenUsers: { [id: string]: User };
  /** Whether list of blocked users should be visible */
  isSettingsVisible: boolean;
  /** Map of collapsed threads */
  collapsedThreads: {
    [key: string]: boolean;
  };
  /** used in user comments widget */
  userComments?: {
    [key: string]: Comment[];
  };
  /** stores info about provider used for login */
  provider: ProviderState;
}

const reducers = combineReducers<StoreState>(storeReducers);
const middleware = applyMiddleware(thunk);

/**
 * Thunk Action shortcut
 */
export type StoreAction<R, A extends AnyAction = ACTIONS> = ThunkAction<R, StoreState, undefined, A>;

/**
 * Thunk Dispatch shortcut
 */
export type StoreDispatch = ThunkDispatch<StoreState, undefined, ACTIONS>;

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const composeEnhancers = (window as any).__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
  ? // eslint-disable-next-line @typescript-eslint/no-explicit-any
    (window as any).__REDUX_DEVTOOLS_EXTENSION_COMPOSE__
  : compose;
const store = createStore(reducers, composeEnhancers(middleware));

if (process.env.NODE_ENV === 'development') {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  (window as any).ReduxStore = store;
}

export default store;
