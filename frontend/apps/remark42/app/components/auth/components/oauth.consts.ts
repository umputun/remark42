export const OAUTH_DATA = {
  apple: {
    name: 'Apple',
    icons: {
      light: require('assets/social/apple-light.svg').default as string,
      dark: require('assets/social/apple-dark.svg').default as string,
    },
  },
  facebook: require('assets/social/facebook.svg').default as string,
  twitter: {
    name: 'X',
    icons: {
      light: require('assets/social/twitter-light.svg').default as string,
      dark: require('assets/social/twitter-dark.svg').default as string,
    },
  },
  patreon: require('assets/social/patreon.svg').default as string,
  discord: require('assets/social/discord.svg').default as string,
  google: require('assets/social/google.svg').default as string,
  microsoft: require('assets/social/microsoft.svg').default as string,
  yandex: require('assets/social/yandex.svg').default as string,
  dev: require('assets/social/dev.svg').default as string,
  custom: require('assets/social/custom.svg').default as string,
  github: {
    name: 'GitHub',
    icons: {
      light: require('assets/social/github-light.svg').default as string,
      dark: require('assets/social/github-dark.svg').default as string,
    },
  },
  telegram: require('assets/social/telegram.svg').default as string,
} as const;

export const OAUTH_PROVIDERS = Object.keys(OAUTH_DATA);
