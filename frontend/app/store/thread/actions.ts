import { Comment } from 'common/types';
import { siteId, url } from 'common/settings';

import { StoreAction } from '../index';
import { THREAD_SET_COLLAPSE, THREAD_RESTORE_COLLAPSE_ACTION, THREAD_RESTORE_COLLAPSE } from './types';
import { saveCollapsedComments, getCollapsedComments } from './utils';

export const restoreCollapsedThreads = (): THREAD_RESTORE_COLLAPSE_ACTION => ({
  type: THREAD_RESTORE_COLLAPSE,
  ids: getCollapsedComments(),
});

export const setCollapse = (id: Comment['id'], value: boolean): StoreAction<void> => (dispatch, getState) => {
  dispatch({
    type: THREAD_SET_COLLAPSE,
    id,
    collapsed: value,
  });
  saveCollapsedComments(
    siteId!,
    url!,
    Object.entries(getState().collapsedThreads).reduce((acc: string[], [key, value]) => {
      if (value) {
        acc.push(key);
      }
      return acc;
    }, [])
  );
};
