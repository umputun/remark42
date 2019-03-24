import { Comment } from '@app/common/types';

import { StoreState } from '../index';
import { USER_INFO_SET, USER_INFO_ACTIONS } from './types';

export const userComments = (
  state: StoreState['userComments'] = {},
  action: USER_INFO_ACTIONS
): { [key: string]: Comment[] } => {
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
