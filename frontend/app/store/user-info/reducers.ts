import { Comment } from '@app/common/types';

import { USER_INFO_SET, USER_INFO_ACTIONS } from './types';

export interface UserCommentsState {
  [key: string]: Comment[];
}

export const userComments = (state: UserCommentsState = {}, action: USER_INFO_ACTIONS): UserCommentsState => {
  switch (action.type) {
    case USER_INFO_SET: {
      return {
        ...state,
        [action.id]: action.comments,
      };
    }
    default:
      return state;
  }
};

export default { userComments };
