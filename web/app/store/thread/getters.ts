import { Comment } from '@app/common/types';
import { StaticStore } from '@app/common/static_store';

import { StoreState } from '../index';

export const getThreadIsCollapsed = (state: StoreState, comment: Comment): boolean => {
  const collapsed = state.collapsedThreads[comment.id];

  if (collapsed !== null && collapsed !== undefined) {
    return collapsed;
  }

  const score = comment.score || 0;

  return score <= StaticStore.config.critical_score;
};
