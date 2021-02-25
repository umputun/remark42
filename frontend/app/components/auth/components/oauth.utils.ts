import { OAuthProvider, Theme } from 'common/types';
import capitalizeFirstLetter from 'utils/capitalize-first-letter';

import { OAUTH_DATA } from './oauth.consts';

export function getButtonVariant(num: number) {
  if (num === 2) {
    return 'name';
  }

  if (num === 1) {
    return 'full';
  }

  return 'icon';
}

export function getProviderData(provider: OAuthProvider, theme: Theme) {
  const data = OAUTH_DATA[provider];

  if (typeof data !== 'string') {
    return {
      name: data.name,
      icon: data.icons[theme],
    };
  }

  return { name: capitalizeFirstLetter(provider), icon: data };
}
