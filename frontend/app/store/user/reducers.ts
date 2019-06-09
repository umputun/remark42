import { User, BlockedUser } from '@app/common/types';

import { StoreState } from '../index';
import {
  USER_SET,
  USER_BAN,
  USER_UNBAN,
  USER_ACTIONS,
  SETTINGS_VISIBLE_SET_ACTION,
  SETTINGS_VISIBLE_SET,
  USER_BANLIST_SET,
  USER_HIDELIST_SET,
  USER_HIDE,
  USER_UNHIDE,
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

export const hiddenUsers = (state: StoreState['hiddenUsers'] = {}, action: USER_ACTIONS): StoreState['hiddenUsers'] => {
  switch (action.type) {
    case USER_HIDELIST_SET: {
      return action.payload;
    }
    case USER_HIDE: {
      return { ...state, [action.user.id]: action.user };
    }
    case USER_UNHIDE: {
      if (!state.hasOwnProperty(action.id)) return state;
      const newState = { ...state };
      delete newState[action.id];
      return newState;
    }
    default:
      return state;
  }
};

export const isSettingsVisible = (
  state: StoreState['isSettingsVisible'] = false,
  action: SETTINGS_VISIBLE_SET_ACTION
): boolean => {
  switch (action.type) {
    case SETTINGS_VISIBLE_SET: {
      return action.state;
    }
    default:
      return state;
  }
};

export default { user, bannedUsers, hiddenUsers, isSettingsVisible };
