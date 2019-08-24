import { PostInfo } from '@app/common/types';

import { StoreAction } from '../index';
import { POST_INFO_SET } from './types';
import api from '@app/common/api';
import { unsetCommentMode } from '../comments/actions';

export const setPostInfo = (info: PostInfo): StoreAction<void> => dispatch =>
  dispatch({
    type: POST_INFO_SET,
    info,
  });

/** set state of post: readonly or not */
export const setCommentsReadOnlyState = (state: boolean): StoreAction<Promise<boolean>> => async (
  dispatch,
  getState
) => {
  await (!state ? api.enableComments() : api.disableComments());
  const storeState = getState();
  dispatch(unsetCommentMode());
  dispatch({
    type: POST_INFO_SET,
    info: { ...storeState.info, read_only: state },
  });
  return state;
};

/** toggles state of post: readonly or not */
export const toggleCommentsReadOnlyState = (): StoreAction<Promise<boolean>> => async (dispatch, getState) => {
  const storeState = getState();
  const state = !storeState.info.read_only!;
  await (state ? api.enableComments() : api.disableComments());
  dispatch(unsetCommentMode());
  dispatch({
    type: POST_INFO_SET,
    info: { ...storeState.info, read_only: !state },
  });
  return !state;
};
