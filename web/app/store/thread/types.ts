import { Comment } from '@app/common/types';

export const THREAD_GET_COLLAPSE = 'THREAD/COLLAPSE_GET';
export interface THREAD_GET_COLLAPSE_ACTION {
  type: typeof THREAD_GET_COLLAPSE;
}

export const THREAD_SET_COLLAPSE = 'THREAD/COLLAPSE_SET';
export interface THREAD_SET_COLLAPSE_ACTION {
  type: typeof THREAD_SET_COLLAPSE;
  id: Comment['id'];
  collapsed: boolean;
}

export type THREAD_ACTIONS = THREAD_GET_COLLAPSE_ACTION | THREAD_SET_COLLAPSE_ACTION;
