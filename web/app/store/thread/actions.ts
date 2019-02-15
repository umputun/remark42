import { Comment } from '@app/common/types';
import { siteId, url } from '@app/common/settings';

import { StoreAction } from '../index';
import { THREAD_SET_COLLAPSE } from './types';
import { saveCollapsedComments } from './utils';

export const setCollapse = (id: Comment['id']): StoreAction<void> => (dispatch, getState) => {
  const collapsed = !getState().collapsedThreads[id];
  dispatch({
    type: THREAD_SET_COLLAPSE,
    id,
    collapsed,
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
