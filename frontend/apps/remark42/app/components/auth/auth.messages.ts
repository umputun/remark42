import { defineMessages } from 'common/intl';

export const messages = defineMessages<string>({
  signin: {
    id: 'auth.signin',
    defaultMessage: 'Sign in',
  },
  or: {
    id: 'auth.or',
    defaultMessage: 'or',
  },
  username: {
    id: 'auth.username',
    defaultMessage: 'Name',
  },
  usernameRestriction: {
    id: 'auth.symbols-restriction',
    defaultMessage: 'Name may only contain letters, numbers, underscores and spaces',
  },
  emailAddress: {
    id: 'auth.email-address',
    defaultMessage: 'Email address',
  },
  token: {
    id: 'token',
    defaultMessage: 'Paste the token from the email',
  },
  expiredToken: {
    id: 'token.expired',
    defaultMessage: 'Token is expired',
  },
  invalidToken: {
    id: 'token.invalid',
    defaultMessage: 'Token is invalid',
  },
  oauthSource: {
    id: 'auth.oauth-source',
    defaultMessage: 'Sign in with a social network',
  },
  oauthTitle: {
    id: 'auth.oauth-button',
    defaultMessage: 'Sign in with {provider}',
  },
  back: {
    id: 'auth.back',
    defaultMessage: 'Back',
  },
  loading: {
    id: 'auth.loading',
    defaultMessage: 'Loading...',
  },
  submit: {
    id: 'auth.submit',
    defaultMessage: 'Confirm',
  },
  openProfile: {
    id: 'auth.open-profile',
    defaultMessage: 'Open your profile',
  },
  signout: {
    id: 'auth.signout',
    defaultMessage: 'Sign out',
  },
  noProviders: {
    id: 'auth.no-providers',
    defaultMessage: 'No sign-in method available. Perhaps the site owner forgot to enable it?',
  },
});
