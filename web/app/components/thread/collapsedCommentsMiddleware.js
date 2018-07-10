import { siteId, url } from 'common/settings';
import { THREAD_SET_COLLAPSE } from './thread.actions';
import getCollapsedComments from './getCollapsedComments';
import saveCollapsedComments from './saveCollapsedComments';

const collapsedCommentsMiddleware = ({ getState }) => next => action => {
  if (action.type === THREAD_SET_COLLAPSE) {
    const state = getState();
    const currentCollapsed = state[action.comment.id];

    if (action.collapsed !== currentCollapsed) {
      const lsCollapsedID = `${siteId}_${url}_${action.comment.id}`;
      let collapsedComments = getCollapsedComments();

      if (action.collapsed) {
        collapsedComments = [...new Set(collapsedComments.concat(lsCollapsedID))];
      } else {
        collapsedComments = collapsedComments.filter(id => id !== lsCollapsedID);
      }

      saveCollapsedComments(collapsedComments);
    }
  }

  next(action);
};

export default collapsedCommentsMiddleware;
