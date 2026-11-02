import type { User } from 'common/types';

/**
 * Reading the hidden-users record out of whatever storage holds.
 *
 * Kept apart from get-hidden-users.ts so it can be tested without a browser: that module binds
 * localStorage through common/local-storage at import. The cases worth having are the ones about
 * finding something other than what this widget wrote, since the key is readable and writable by
 * anything else on the page, and a shape that reaches the caller unchecked would be spread over
 * the comment list.
 */
export function parseHiddenUsers(raw: string | null): Record<string, User> {
  try {
    const hiddenUsers: Record<string, User> = JSON.parse(raw || '{}');

    // a list, a string or a null would all survive JSON.parse, and only a plain object can be
    // indexed by user id the way the caller does
    if (typeof hiddenUsers === 'object' && hiddenUsers !== null && !Array.isArray(hiddenUsers)) {
      return hiddenUsers;
    }
  } catch (e) {
    console.error('incorrect hidden user data in local storage', e);
  }

  return {};
}
