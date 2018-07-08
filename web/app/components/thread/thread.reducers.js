import { siteId, url } from 'common/settings';
import { THREAD_SET_COLLAPSE } from './thread.actions';
import getCollapsedComments from './getCollapsedComments';

const collapsedCommentIds = getCollapsedComments()
  .map(comment => comment.split('_'))
  .filter(components => components[0] === siteId && components[1] === url)
  .map(component => component[2]);

const initialState = collapsedCommentIds.reduce((acc, id) => ({ ...acc, [id]: true }), {});

export const collapsedThreads = (state = initialState, action) => {
  switch (action.type) {
    case THREAD_SET_COLLAPSE:
      return {
        ...state,
        [action.comment.id]: action.collapsed,
      };
    default:
      return state;
  }
};
