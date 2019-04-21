import { User } from '@app/common/types';

/**
 * Defines whether current client is logged in via `Anonymous provider`
 */
export function isUserAnonymous(user?: User | null): boolean {
  return user! && user!.id.substr(0, 10) === 'anonymous_';
}
