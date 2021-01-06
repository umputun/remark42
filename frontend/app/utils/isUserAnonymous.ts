import { User } from 'common/types';

/**
 * Defines whether current client is logged in via `Anonymous provider`
 */
export function isUserAnonymous(user: User | null) {
  return user === null || user.id.substr(0, 10) === 'anonymous_';
}
