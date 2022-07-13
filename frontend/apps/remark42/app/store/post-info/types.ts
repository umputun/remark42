import { PostInfo } from 'common/types';

export const POST_INFO_SET = 'POST_INFO/SET';

export interface POST_INFO_SET_ACTION {
  type: typeof POST_INFO_SET;
  info: PostInfo;
}

export type POST_INFO_ACTIONS = POST_INFO_SET_ACTION;
