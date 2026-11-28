import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useIntl } from 'common/intl';
import type { User } from 'common/types';
import type { StoreState } from 'store';
import { useDispatch, useSelector } from 'store/context';
import { setUserSubscribed } from 'store/user/actions';
import { useTheme } from 'hooks/useTheme';
import {
  emailVerificationForSubscribe,
  emailConfirmationForSubscribe,
  unsubscribeFromEmailUpdates,
  getEmailSubscription,
} from 'common/api';
import { isUserAnonymous } from 'utils/isUserAnonymous';

import { mountSubscribeByEmail } from './subscribe-by-email-element.js';

export { CONFIRMATION_POLL_DURATION, CONFIRMATION_POLL_INTERVAL } from './subscribe-by-email-element.js';

/** The props and deps of the plain email subscription, from the preact tree around it. */
export function useSubscribeByEmailArgs() {
  const theme = useTheme();
  const intl = useIntl();
  const dispatch = useDispatch();
  const user = useSelector<StoreState, User | null>(({ user }) => user);
  const props: Parameters<typeof mountSubscribeByEmail>[1] = {
    subscribed: Boolean(user?.email_subscription),
    userId: user?.id,
    anonymous: isUserAnonymous(user),
    theme,
  };
  const deps: Parameters<typeof mountSubscribeByEmail>[2] = {
    intl,
    verify: (address) => emailVerificationForSubscribe(address),
    confirm: (token) => emailConfirmationForSubscribe(token),
    unsubscribe: () => unsubscribeFromEmailUpdates(),
    subscription: () => getEmailSubscription(),
    setSubscribed: (on) => dispatch(setUserSubscribed(on)),
  };

  return { props, deps };
}

/**
 * Mounts the plain email subscription into a div preact renders without props or children, so a
 * re-render of the form leaves the div's attributes and content to the plain component.
 */
export function SubscribeByEmail() {
  const { props, deps } = useSubscribeByEmailArgs();
  const root = useRef<HTMLDivElement>(null);
  const subscription = useRef<ReturnType<typeof mountSubscribeByEmail> | null>(null);

  useLayoutEffect(() => {
    subscription.current = mountSubscribeByEmail(root.current!, props, deps);
    return () => subscription.current?.destroy();
    // mounted once; the props and deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useLayoutEffect(() => {
    subscription.current?.update(props, deps);
  });

  return <div ref={root} />;
}
