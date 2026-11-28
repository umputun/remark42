import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useAppDispatch, type StoreDispatch } from 'store';
import { useIntl, type IntlShape } from 'common/intl';
import * as commonApi from 'common/api';
import type { Theme, User } from 'common/types';
import { signin } from 'store/user/actions';
import { useTheme } from 'hooks/useTheme';

import * as authApi from './auth.api';
import type { SignInApi, SignInDeps } from './sign-in.js';
import { mountSignIn } from './sign-in.js';

// each call goes through the module at the moment it is made, so a call replaced after the form
// mounted is the one it makes
const api: SignInApi = {
  anonymousSignin: (username) => authApi.anonymousSignin(username),
  emailSignin: (email, username) => authApi.emailSignin(email, username),
  verifyEmailSignin: (token) => authApi.verifyEmailSignin(token),
  oauthSignin: (url) => authApi.oauthSignin(url),
  getTelegramSigninParams: () => authApi.getTelegramSigninParams(),
  verifyTelegramSignin: (token) => authApi.verifyTelegramSignin(token),
  getSignedInUser: () => commonApi.getSignedInUser(),
  getUser: () => commonApi.getUser(),
};

/** What the plain sign-in form needs, from the locale, the theme and the store's dispatch. */
export function signInDeps(intl: IntlShape, theme: Theme, dispatch: StoreDispatch): SignInDeps {
  return { intl, theme, signin: (user: User) => dispatch(signin(user)), api };
}

/** What the plain sign-in form needs from the preact tree around it. */
export function useSignInDeps(): SignInDeps {
  return signInDeps(useIntl(), useTheme(), useAppDispatch());
}

/**
 * Mounts the plain sign-in form into a div preact renders without props or children, so a
 * re-render around it leaves the div's class and content to the plain component.
 */
export function Auth() {
  const deps = useSignInDeps();
  const root = useRef<HTMLDivElement>(null);
  const form = useRef<ReturnType<typeof mountSignIn> | null>(null);

  useLayoutEffect(() => {
    form.current = mountSignIn(root.current!, deps);
    return () => form.current?.destroy();
    // mounted once; the deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // every render, so a new theme or locale reaches the form
  useLayoutEffect(() => {
    form.current?.update(deps);
  });

  return <div ref={root} />;
}
