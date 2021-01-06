import type { Comment } from 'common/types';
import { StaticStore } from 'common/static-store';

import { StoreState } from '../index';

export const getThreadIsCollapsed = (comment: Comment) => (state: StoreState): boolean => {
  const collapsed = state.collapsedThreads[comment.id];

  if (collapsed !== null && collapsed !== undefined) {
    return collapsed;
  }

  const score = comment.score || 0;

  return score <= StaticStore.config.critical_score;
};
