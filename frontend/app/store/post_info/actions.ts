import { PostInfo } from '@app/common/types';

import { StoreAction } from '../index';
import { POST_INFO_SET, POST_INFO_SET_ACTION } from './types';
import api from '@app/common/api';
import { unsetCommentMode } from '../comments/actions';

export function setPostInfo(info: PostInfo) {
  return {
    type: POST_INFO_SET,
    info,
  } as POST_INFO_SET_ACTION;
}

/** set state of post: readonly or not */
export const setCommentsReadOnlyState = (read_only: boolean): StoreAction<Promise<boolean>> => async (
  dispatch,
  getState
) => {
  const { info } = getState();

  await (read_only ? api.disableComments() : api.enableComments());
  dispatch(unsetCommentMode());
  dispatch(setPostInfo({ ...info, read_only }));

  return read_only;
};

/** toggles state of post: readonly or not */
export const toggleCommentsReadOnlyState = (): StoreAction<Promise<boolean>> => async (dispatch, getState) => {
  const { info } = getState();
  const { read_only } = info;

  await (read_only ? api.disableComments() : api.enableComments());
  dispatch(unsetCommentMode());
  dispatch(setPostInfo({ ...info, read_only }));

  return read_only!;
};
