import { h } from 'preact';
import { useLayoutEffect, useRef } from 'preact/hooks';

import { useIntl, type IntlShape } from 'common/intl';
import type { Theme, User } from 'common/types';
import type { StoreState } from 'store';
import { useSelector } from 'store/context';
import { useTheme } from 'hooks/useTheme';
import { telegramSubscribe, telegramCurrentSubscribtion, telegramUnsubcribe } from 'common/api';
import { isUserAnonymous } from 'utils/isUserAnonymous';

import { mountSubscribeByTelegram } from './subscribe-by-telegram-element.js';

/** The props and deps of the plain Telegram subscription, for this reader, theme and locale. */
export function subscribeByTelegramArgs(user: User | null, theme: Theme, intl: IntlShape) {
  const props: Parameters<typeof mountSubscribeByTelegram>[1] = {
    userId: user?.id ?? '',
    anonymous: isUserAnonymous(user),
    theme,
  };
  const deps: Parameters<typeof mountSubscribeByTelegram>[2] = {
    intl,
    subscribe: () => telegramSubscribe(),
    check: (token) => telegramCurrentSubscribtion({ token }),
    unsubscribe: () => telegramUnsubcribe(),
    // read when the panel opens, as the preact form read it: where the browser blocks storage in a
    // frame, reading it throws, and the closed button must still draw
    get storage() {
      return sessionStorage;
    },
  };

  return { props, deps };
}

/** The props and deps of the plain Telegram subscription, from the preact tree around it. */
export function useSubscribeByTelegramArgs() {
  const theme = useTheme();
  const intl = useIntl();
  const user = useSelector<StoreState, User | null>(({ user }) => user);
  return subscribeByTelegramArgs(user, theme, intl);
}

/**
 * Mounts the plain Telegram subscription into a div preact renders without props or children, so a
 * re-render of the form leaves the div's attributes and content to the plain component.
 */
export function SubscribeByTelegram() {
  const { props, deps } = useSubscribeByTelegramArgs();
  const root = useRef<HTMLDivElement>(null);
  const subscription = useRef<ReturnType<typeof mountSubscribeByTelegram> | null>(null);

  useLayoutEffect(() => {
    subscription.current = mountSubscribeByTelegram(root.current!, props, deps);
    return () => subscription.current?.destroy();
    // mounted once; the props and deps of every later render reach it through update below
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useLayoutEffect(() => {
    subscription.current?.update(props, deps);
  });

  return <div ref={root} />;
}
