import { THREAD_GET_COLLAPSE_ACTION, THREAD_SET_COLLAPSE, THREAD_SET_COLLAPSE_ACTION } from './types';
import { getCollapsedComments } from './utils';
import { StoreState } from '../index';

const collapsedCommentIds = getCollapsedComments();

const initialState: StoreState['collapsedThreads'] = collapsedCommentIds.reduce(
  (acc: { [key: string]: boolean }, id) => {
    acc[id] = true;
    return acc;
  },
  {}
);

export const collapsedThreads = (
  state: StoreState['collapsedThreads'] = initialState,
  action: THREAD_GET_COLLAPSE_ACTION | THREAD_SET_COLLAPSE_ACTION
): { [key: string]: boolean } => {
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
