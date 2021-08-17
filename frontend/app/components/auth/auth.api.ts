import type { User, TelegramParams } from 'common/types';

import { authFetcher } from 'common/fetcher';
import { siteId } from 'common/settings';

const EMAIL_SIGNIN_ENDPOINT = '/email/login';
const TELEGRAM_SIGNIN_ENDPOINT = '/telegram/login';

export function anonymousSignin(user: string): Promise<User> {
  return authFetcher.get<User>('/anonymous/login', { user, aud: siteId });
}

/**
 * First step of two of `email` authorization
 */
export function emailSignin(email: string, username: string): Promise<unknown> {
  return authFetcher.get(EMAIL_SIGNIN_ENDPOINT, { address: email, user: username });
}

/**
 * Second step of two of `email` authorization
 */
export function verifyEmailSignin(token: string): Promise<User> {
  return authFetcher.get(EMAIL_SIGNIN_ENDPOINT, { token });
}

/**
 * First step of two of `telegram` authorization
 */
export function getTelegramSigninParams(): Promise<TelegramParams> {
  return authFetcher.get(TELEGRAM_SIGNIN_ENDPOINT);
}

/**
 * Second step of two of `telegram` authorization
 */
export function verifyTelegramSignin(token: string): Promise<User> {
  return authFetcher.get(TELEGRAM_SIGNIN_ENDPOINT, { token });
}

export function logout(): Promise<void> {
  return authFetcher.get('/logout');
}
