export const OAUTH_DATA = {
  facebook: require('./assets/facebook.svg').default as string,
  twitter: require('./assets/twitter.svg').default as string,
  google: require('./assets/google.svg').default as string,
  microsoft: require('./assets/microsoft.svg').default as string,
  yandex: require('./assets/yandex.svg').default as string,
  dev: require('./assets/dev.svg').default as string,
  github: {
    name: 'GitHub',
    icons: {
      light: require('./assets/github-light.svg').default as string,
      dark: require('./assets/github-dark.svg').default as string,
    },
  },
} as const;

export const OAUTH_PROVIDERS = Object.keys(OAUTH_DATA);
