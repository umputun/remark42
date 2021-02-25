import { isJwtExpired } from 'utils/jwt';
import { StaticStore } from 'common/static-store';
import type { FormProvider, OAuthProvider } from 'common/types';

import { OAUTH_PROVIDERS } from './components/oauth.consts';
import messages from './auth.messsages';

export function getProviders(): [OAuthProvider[], FormProvider[]] {
  const oauthProviders: OAuthProvider[] = [];
  const formProviders: FormProvider[] = [];

  StaticStore.config.auth_providers.forEach((p) => {
    OAUTH_PROVIDERS.includes(p) ? oauthProviders.push(p as OAuthProvider) : formProviders.push(p as FormProvider);
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
