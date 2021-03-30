import { defineMessages } from 'react-intl';

const messages = defineMessages({
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
});

export default messages;
