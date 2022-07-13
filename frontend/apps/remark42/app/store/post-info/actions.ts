import { PostInfo } from 'common/types';

import { StoreAction } from '../index';
import { POST_INFO_SET, POST_INFO_SET_ACTION } from './types';
import { disableComments, enableComments } from 'common/api';
import { unsetCommentMode } from '../comments/actions';

export function setPostInfo(info: PostInfo) {
  return {
    type: POST_INFO_SET,
    info,
  } as POST_INFO_SET_ACTION;
}

/** set state of post: readonly or not */
export function setCommentsReadOnlyState(read_only: boolean): StoreAction<Promise<void>> {
  return async (dispatch, getState) => {
    const { info } = getState();

    await (read_only ? disableComments() : enableComments());
    dispatch(unsetCommentMode());
    dispatch(setPostInfo({ ...info, read_only }));
  };
}
