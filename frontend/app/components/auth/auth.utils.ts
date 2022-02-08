import { useMemo, useState } from 'preact/hooks';
import { useIntl } from 'react-intl';

import { isJwtExpired } from 'utils/jwt';
import { errorMessages, RequestError } from 'utils/errorUtils';
import { StaticStore } from 'common/static-store';
import type { FormProvider, OAuthProvider } from 'common/types';

import { OAUTH_PROVIDERS } from './components/oauth.consts';
import { messages } from './auth.messsages';

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

export function useErrorMessage(): [string | null, (e: unknown) => void] {
  const intl = useIntl();
  const [invalidReason, setInvalidReason] = useState<string | null>(null);

  return useMemo(() => {
    let errorMessage = invalidReason;

    if (invalidReason && messages[invalidReason]) {
      errorMessage = intl.formatMessage(messages[invalidReason]);
    }

    if (invalidReason && errorMessages[invalidReason]) {
      errorMessage = intl.formatMessage(errorMessages[invalidReason]);
    }

    function setError(err: unknown): void {
      if (err === null) {
        setInvalidReason(null);
        return;
      }

      if (typeof err === 'string') {
        setInvalidReason(err);
        return;
      }

      const errorReason = err instanceof RequestError ? err.error : err instanceof Error ? err.message : 'error.0';

      setInvalidReason(errorReason);
    }

    return [errorMessage, setError];
  }, [intl, invalidReason]);
}
