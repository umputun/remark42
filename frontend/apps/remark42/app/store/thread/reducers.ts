import type { THREAD_ACTIONS } from './types.ts';
import { THREAD_SET_COLLAPSE, THREAD_RESTORE_COLLAPSE } from './types.ts';

export interface CollapsedThreadsState {
  [key: string]: boolean;
}

export function collapsedThreads(state: CollapsedThreadsState = {}, action: THREAD_ACTIONS): CollapsedThreadsState {
  switch (action.type) {
    case THREAD_RESTORE_COLLAPSE: {
      return action.ids.reduce<CollapsedThreadsState>((acc, id) => {
        acc[id] = true;
        return acc;
      }, {});
    }
    case THREAD_SET_COLLAPSE: {
      return {
        ...state,
        [action.id]: action.collapsed,
      };
    }
    default:
      return state;
  }
}
