import { defineMessages } from 'react-intl';

export const messages = defineMessages({
  signin: {
    id: 'auth.signin',
    defaultMessage: 'Sign In',
  },
  or: {
    id: 'auth.or',
    defaultMessage: 'or',
  },
  username: {
    id: 'auth.username',
    defaultMessage: 'Username',
  },
  usernameRestriction: {
    id: 'auth.symbols-restriction',
    defaultMessage: 'Username must contain only letters, numbers, underscores or spaces',
  },
  userNotFound: {
    id: 'auth.user-not-found',
    defaultMessage: 'No user was found',
  },
  emailAddress: {
    id: 'auth.email-address',
    defaultMessage: 'Email Address',
  },
  token: {
    id: 'token',
    defaultMessage: 'Token',
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
    defaultMessage: 'Use Social Network',
  },
  oauthTitle: {
    id: 'auth.oauth-button',
    defaultMessage: 'Sign In with {provider}',
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
    defaultMessage: 'Submit',
  },
  telegramLink: {
    id: 'auth.telegram-link',
    defaultMessage: 'by the link',
  },
  telegramCheck: {
    id: 'auth.telegram-check',
    defaultMessage: 'Check',
  },
  telegramMessage1: {
    id: 'auth.telegram-message-1',
    defaultMessage: 'Open the Telegram',
  },
  telegramOptionalQR: {
    id: 'auth.telegram-optional-qr',
    defaultMessage: 'or by scanning the QR code',
  },
  telegramMessage2: {
    id: 'auth.telegram-message-2',
    defaultMessage: 'and click “Start” there.',
  },
  telegramMessage3: {
    id: 'auth.telegram-message-3',
    defaultMessage: 'Afterwards, click “Check” below.',
  },
  openProfile: {
    id: 'auth.open-profile',
    defaultMessage: 'Open My Profile',
  },
  signout: {
    id: 'auth.signout',
    defaultMessage: 'Sign Out',
  },
});
