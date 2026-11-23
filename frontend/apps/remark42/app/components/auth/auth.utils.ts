import { isJwtExpired } from 'utils/jwt';
import { StaticStore } from 'common/static-store';
import type { FormProvider, OAuthProvider } from 'common/types';

import type { messages } from './auth.messages';
import { setItem, getItem } from 'common/local-storage';
import { LS_EMAIL_KEY } from 'common/constants';

export function getProviders(): [OAuthProvider[], FormProvider[]] {
  const oauthProviders: OAuthProvider[] = [];
  const formProviders: FormProvider[] = [];

  StaticStore.config.auth_providers.forEach((p) => {
    p === 'email' || p === 'anonymous'
      ? formProviders.push(p as FormProvider)
      : oauthProviders.push(p as OAuthProvider);
  });

  return [oauthProviders, formProviders];
}

export function getTokenInvalidReason(token: string): null | keyof typeof messages {
  try {
    if (isJwtExpired(token)) {
      return 'expiredToken';
    }
  } catch (e) {
    return 'invalidToken';
  }

  return null;
}

/**
 * Whether the text is a whole signed token. The expiry is read from the payload, which parses
 * as soon as it is complete, so without this a token still being typed or pasted in pieces would
 * count as ready before its signature is there. 43 characters is the shortest signature a JWT
 * carries, the HMAC-SHA256 one the backend issues
 */
export function isCompleteToken(token: string): boolean {
  return /^[\w-]+\.[\w-]+\.[\w-]{43,}$/.test(token);
}

export function persistEmail(email: string) {
  setItem(LS_EMAIL_KEY, email);
}

export function getPersistedEmail() {
  return getItem(LS_EMAIL_KEY) || '';
}

export function resetPersistedEmail() {
  setItem(LS_EMAIL_KEY, '');
}
