import { getItem } from 'common/local-storage';
import { LS_HIDDEN_USERS_KEY } from 'common/constants';

import { parseHiddenUsers } from './hidden-users';

/** The users this reader has hidden, as stored in the browser. */
export function getHiddenUsers() {
  return parseHiddenUsers(getItem(LS_HIDDEN_USERS_KEY));
}
