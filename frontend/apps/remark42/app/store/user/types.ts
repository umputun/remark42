import { User, BlockedUser } from 'common/types';

export const USER_SET = 'USER/SET';

export interface USER_SET_ACTION {
  type: typeof USER_SET;
  user: User | null;
}

/**
 * Set list of banned users
 */
export const USER_BANLIST_SET = 'USER/BANLIST_SET';
export interface USER_BANLIST_SET_ACTION {
  type: typeof USER_BANLIST_SET;
  list: BlockedUser[];
}

export const USER_BAN = 'USER/BAN';
export interface USER_BAN_ACTION {
  type: typeof USER_BAN;
  user: BlockedUser;
}

export const USER_UNBAN = 'USER/UNBAN';
export interface USER_UNBAN_ACTION {
  type: typeof USER_UNBAN;
  id: User['id'];
}

/**
 * Set list of hidden users
 */
export const USER_HIDELIST_SET = 'USER/HIDELIST_SET';
export interface USER_HIDELIST_SET_ACTION {
  type: typeof USER_HIDELIST_SET;
  payload: { [id: string]: User };
}

export const USER_HIDE = 'USER/HIDE';
export interface USER_HIDE_ACTION {
  type: typeof USER_HIDE;
  user: User;
}

export const USER_UNHIDE = 'USER/UNHIDE';
export interface USER_UNHIDE_ACTION {
  type: typeof USER_UNHIDE;
  id: User['id'];
}

export const USER_SUBSCRIPTION_SET = 'USER_SUBSCRIPTION/SET';

export interface USER_SUBSCRIPTION_SET_ACTION {
  type: typeof USER_SUBSCRIPTION_SET;
  payload: boolean;
}

export type USER_ACTIONS =
  | USER_SET_ACTION
  | USER_BANLIST_SET_ACTION
  | USER_BAN_ACTION
  | USER_UNBAN_ACTION
  | USER_HIDELIST_SET_ACTION
  | USER_HIDE_ACTION
  | USER_UNHIDE_ACTION
  | USER_SUBSCRIPTION_SET_ACTION;
