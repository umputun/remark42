import { User, BlockedUser } from '@app/common/types';

import { StoreState } from '../index';
import {
  USER_SET,
  USER_BAN,
  USER_UNBAN,
  USER_ACTIONS,
  BLOCKED_VISIBLE_SET_ACTION,
  BLOCKED_VISIBLE_SET,
  USER_BANLIST_SET,
} from './types';

export const user = (state: StoreState['user'] = null, action: USER_ACTIONS): User | null => {
  switch (action.type) {
    case USER_SET: {
      return action.user;
    }
    default:
      return state;
  }
};

export const bannedUsers = (state: StoreState['bannedUsers'] = [], action: USER_ACTIONS): BlockedUser[] => {
  switch (action.type) {
    case USER_BANLIST_SET: {
      return action.list;
    }
    case USER_BAN: {
      if (state.find(u => u.id === action.user.id) !== undefined) {
        return state;
      }
      return [action.user, ...state];
    }
    case USER_UNBAN: {
      const index = state.findIndex(u => u.id === action.id);
      if (index === -1) {
        return state;
      }
      return [...state.slice(0, index), ...state.slice(index + 1)];
    }
    default:
      return state;
  }
};

export const isBlockedVisible = (
  state: StoreState['isBlockedVisible'] = false,
  action: BLOCKED_VISIBLE_SET_ACTION
): boolean => {
  switch (action.type) {
    case BLOCKED_VISIBLE_SET: {
      return action.state;
    }
    default:
      return state;
  }
};

export default { user, bannedUsers, isBlockedVisible };
