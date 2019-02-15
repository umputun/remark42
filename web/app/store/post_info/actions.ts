import { PostInfo } from '@app/common/types';

import { StoreAction } from '../index';
import { POST_INFO_SET } from './types';

export const setPostInfo = (info: PostInfo): StoreAction<void> => dispatch =>
  dispatch({
    type: POST_INFO_SET,
    info,
  });
