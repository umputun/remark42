import type { COMMENTS_ACTIONS } from './comments/types';
import type { POST_INFO_ACTIONS } from './post-info/types';
import type { THEME_ACTIONS } from './theme/types';
import type { THREAD_ACTIONS } from './thread/types';
import type { USER_ACTIONS } from './user/types';

/** Merged store actions */
export type ACTIONS = COMMENTS_ACTIONS | POST_INFO_ACTIONS | THEME_ACTIONS | THREAD_ACTIONS | USER_ACTIONS;
