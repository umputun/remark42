import { PostInfo } from '@app/common/types';

export const POST_INFO_SET = 'POST_INFO/SET';

export interface POST_INFO_SET_ACTION {
  type: typeof POST_INFO_SET;
  info: PostInfo;
}

export const POST_INFO_SET_READONLY = 'COMMENTS/SET_READONLY';

export interface POST_INFO_SET_READONLY_ACTION {
  type: typeof POST_INFO_SET_READONLY;
  readonly: boolean;
}

export type POST_INFO_ACTIONS = POST_INFO_SET_ACTION | POST_INFO_SET_READONLY_ACTION;
