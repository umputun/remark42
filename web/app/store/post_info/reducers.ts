import { PostInfo } from '@app/common/types';
import { POST_INFO_SET, POST_INFO_SET_ACTION } from './types';

/* eslint-disable @typescript-eslint/camelcase */
const DefaultPostInfo: PostInfo = {
  url: '',
  count: 0,
  read_only: false,
  first_time: '',
  last_time: '',
};
/* eslint-enable @typescript-eslint/camelcase */

export const info = (state: PostInfo = DefaultPostInfo, action: POST_INFO_SET_ACTION): PostInfo => {
  switch (action.type) {
    case POST_INFO_SET: {
      return action.info;
    }
    default:
      return state;
  }
};

export default { info };
