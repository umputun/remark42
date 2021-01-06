import { User } from 'common/types';
import { getItem } from 'common/local-storage';
import { LS_HIDDEN_USERS_KEY } from 'common/constants';

export default function getHiddenUsers() {
  try {
    const hiddenUsers: Record<string, User> = JSON.parse(getItem(LS_HIDDEN_USERS_KEY) || '{}');

    if (typeof hiddenUsers === 'object' && hiddenUsers !== null && !Array.isArray(hiddenUsers)) {
      return hiddenUsers;
    }
  } catch (e) {
    console.error('incorrect hidden user data in local storage', e); // eslint-disable-line no-console
  }

  return {};
}
