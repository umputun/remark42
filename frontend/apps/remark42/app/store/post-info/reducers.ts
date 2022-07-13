import type { PostInfo } from 'common/types';
import { cmpRef } from 'utils/cmpRef';

import { POST_INFO_SET, POST_INFO_SET_ACTION } from './types';

const DefaultPostInfo: PostInfo = {
  url: '',
  count: 0,
  read_only: false,
  first_time: '',
  last_time: '',
};

export function info(state: PostInfo = DefaultPostInfo, action: POST_INFO_SET_ACTION): PostInfo {
  switch (action.type) {
    case POST_INFO_SET: {
      return cmpRef(state, action.info);
    }
    default:
      return state;
  }
}
