import store from 'common/store';

export const getThreadIsCollapsed = (state, comment) => {
  let collapsed = state.collapsedThreads[comment.id];

  if (collapsed !== null && collapsed !== undefined) {
    return collapsed;
  }

  const config = store.get('config') || {};
  const score = comment.score || 0;

  return score <= config.critical_score;
};
