import { Comment } from 'common/types';

export const THREAD_RESTORE_COLLAPSE = 'THREAD/COLLAPSE_RESTORE';
export interface THREAD_RESTORE_COLLAPSE_ACTION {
  type: typeof THREAD_RESTORE_COLLAPSE;
  ids: Comment['id'][];
}

export const THREAD_SET_COLLAPSE = 'THREAD/COLLAPSE_SET';
export interface THREAD_SET_COLLAPSE_ACTION {
  type: typeof THREAD_SET_COLLAPSE;
  id: Comment['id'];
  collapsed: boolean;
}

export type THREAD_ACTIONS = THREAD_RESTORE_COLLAPSE_ACTION | THREAD_SET_COLLAPSE_ACTION;
