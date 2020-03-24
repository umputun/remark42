import { User } from '@app/common/types';
import { getItem } from '@app/common/local-storage';
import { LS_HIDDEN_USERS_KEY } from '@app/common/constants';

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
