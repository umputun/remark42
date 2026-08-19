import { defineMessages } from 'react-intl';

export const messages = defineMessages<string>({
  signin: {
    id: 'auth.signin',
    defaultMessage: 'Sign In',
  },
  signinorpickname: {
    id: 'auth.signinorpickname',
    defaultMessage: 'Sign in or pick name',
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
    defaultMessage: 'Username must contain only letters, dots, numbers, dashes, underscores or spaces',
  },
  usernameRestrictionLetters: {
    id: 'auth.symbols-restriction-letters',
    defaultMessage: 'Username may only contain letters, dots, dashes and spaces',
  },
  userNotFound: {
    id: 'auth.user-not-found',
    defaultMessage: 'No user was found',
  },
  emailAddress: {
    id: 'auth.email-address',
    defaultMessage: 'Email Address',
  },
  notRequired: {
    id: 'auth.not-required',
    defaultMessage: 'not required',
  },
  token: {
    id: 'token',
    defaultMessage: 'Copy and paste the token from the email',
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
  openProfile: {
    id: 'auth.open-profile',
    defaultMessage: 'Open My Profile',
  },
  signout: {
    id: 'auth.signout',
    defaultMessage: 'Sign Out',
  },
  noProviders: {
    id: 'auth.no-providers',
    defaultMessage: 'No providers available. Maybe you forgot to enable them?',
  },
});
