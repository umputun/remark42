import { Comment } from '@app/common/types';
import { siteId, url } from '@app/common/settings';

import { StoreAction } from '../index';
import { THREAD_SET_COLLAPSE } from './types';
import { saveCollapsedComments } from './utils';

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
