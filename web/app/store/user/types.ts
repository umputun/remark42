import { User, BlockedUser } from '@app/common/types';

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

export const BLOCKED_VISIBLE_SET = 'BLOCKED_VISIBLE/SET';

export interface BLOCKED_VISIBLE_SET_ACTION {
  type: typeof BLOCKED_VISIBLE_SET;
  state: boolean;
}

export type USER_ACTIONS =
  | USER_SET_ACTION
  | USER_BANLIST_SET_ACTION
  | USER_BAN_ACTION
  | USER_UNBAN_ACTION
  | BLOCKED_VISIBLE_SET_ACTION;
