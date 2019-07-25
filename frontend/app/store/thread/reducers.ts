import { THREAD_GET_COLLAPSE_ACTION, THREAD_SET_COLLAPSE, THREAD_SET_COLLAPSE_ACTION } from './types';
import { getCollapsedComments } from './utils';

const collapsedCommentIds = getCollapsedComments();

export interface CollapsedThreadsState {
  [key: string]: boolean;
}

const initialState: CollapsedThreadsState = collapsedCommentIds.reduce((acc: { [key: string]: boolean }, id) => {
  acc[id] = true;
  return acc;
}, {});

export const collapsedThreads = (
  state: CollapsedThreadsState = initialState,
  action: THREAD_GET_COLLAPSE_ACTION | THREAD_SET_COLLAPSE_ACTION
): CollapsedThreadsState => {
  switch (action.type) {
    case THREAD_SET_COLLAPSE:
      return {
        ...state,
        [action.id]: action.collapsed,
      };
    default:
      return state;
  }
};

export default {
  collapsedThreads,
};
