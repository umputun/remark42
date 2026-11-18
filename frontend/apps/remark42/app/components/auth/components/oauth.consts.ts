import { webAssetUrl } from 'common/web-asset';

export const OAUTH_DATA = {
  apple: {
    name: 'Apple',
    icons: {
      light: webAssetUrl('apple-light.svg'),
      dark: webAssetUrl('apple-dark.svg'),
    },
  },
  facebook: webAssetUrl('facebook.svg'),
  twitter: {
    name: 'X',
    icons: {
      light: webAssetUrl('twitter-light.svg'),
      dark: webAssetUrl('twitter-dark.svg'),
    },
  },
  patreon: webAssetUrl('patreon.svg'),
  discord: webAssetUrl('discord.svg'),
  google: webAssetUrl('google.svg'),
  microsoft: webAssetUrl('microsoft.svg'),
  yandex: webAssetUrl('yandex.svg'),
  dev: webAssetUrl('dev.svg'),
  custom: webAssetUrl('custom.svg'),
  github: {
    name: 'GitHub',
    icons: {
      light: webAssetUrl('github-light.svg'),
      dark: webAssetUrl('github-dark.svg'),
    },
  },
  telegram: webAssetUrl('telegram.svg'),
} as const;

export const OAUTH_PROVIDERS = Object.keys(OAUTH_DATA);
