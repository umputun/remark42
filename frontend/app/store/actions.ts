import { COMMENTS_ACTIONS } from './comments/types';
import { POST_INFO_ACTIONS } from './post_info/types';
import { SORT_ACTIONS } from './sort/types';
import { THEME_ACTIONS } from './theme/types';
import { THREAD_ACTIONS } from './thread/types';
import { USER_ACTIONS } from './user/types';
import { USER_INFO_ACTIONS } from './user-info/types';

/** Merged store actions */
export type ACTIONS =
  | COMMENTS_ACTIONS
  | POST_INFO_ACTIONS
  | SORT_ACTIONS
  | THEME_ACTIONS
  | THREAD_ACTIONS
  | USER_ACTIONS
  | USER_INFO_ACTIONS;
