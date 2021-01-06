import { Comment, User } from 'common/types';

export const USER_INFO_SET = 'USER-INFO/SET';

export interface USER_INFO_SET_ACTION {
  type: typeof USER_INFO_SET;
  id: User['id'];
  comments: Comment[];
}

export type USER_INFO_ACTIONS = USER_INFO_SET_ACTION;
