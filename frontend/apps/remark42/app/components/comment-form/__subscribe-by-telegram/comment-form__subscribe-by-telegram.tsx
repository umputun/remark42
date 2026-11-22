import clsx from 'clsx';
import type { FunctionComponent } from 'preact';
import { h, Fragment } from 'preact';
import { useState, useEffect, useRef } from 'preact/hooks';
import { useSelector } from 'store/context';

import { useIntl, defineMessages } from 'common/intl';
import type { User } from 'common/types';
import { siteId } from 'common/settings';
import type { StoreState } from 'store';
import type { FetcherError, RequestError } from 'utils/errorUtils';
import { extractErrorMessageFromResponse } from 'utils/errorUtils';
import { useTheme } from 'hooks/useTheme';
import { useSessionStorage } from 'hooks/useSessionState';
import { telegramSubscribe, telegramCurrentSubscribtion, telegramUnsubcribe } from 'common/api';
import { Dropdown } from 'components/dropdown';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { TelegramLink } from 'components/telegram/telegram-link';
import { Preloader } from 'components/preloader';
import { Button } from 'components/button';

import styles from './comment-form__subscribe-by-telegram.module.css';

const messages = defineMessages({
  haveSubscribed: {
    id: 'subscribeByTelegram.have-been-subscribed',
    defaultMessage: 'You have subscribed to Telegram updates',
  },
  haveUnsubscribed: {
    id: 'subscribeByTelegram.have-been-unsubscribed',
    defaultMessage: 'You have unsubscribed from Telegram updates',
  },
  unsubscribe: {
    id: 'subscribeByTelegram.unsubscribe',
    defaultMessage: 'Unsubscribe',
  },
  resubscribe: {
    id: 'subscribeByTelegram.resubscribe',
    defaultMessage: 'Resubscribe',
  },
  subscribeByTelegram: {
    id: 'subscribeByTelegram.subscribe-by-telegram',
    defaultMessage: 'Subscribe by Telegram',
  },
  onlyRegisteredUsers: {
    id: 'subscribeByTelegram.only-registered-users',
    defaultMessage: 'Only available to registered users',
  },
  telegram: {
    id: 'subscribeByTelegram.telegram',
    defaultMessage: 'Telegram',
  },
});

type STEP = 'initial' | 'subscribed' | 'unsubscribed';

/** how long the server keeps a subscription link, see telegramSubscribeCtrl in the backend */
const LINK_LIFETIME = 10 * 60 * 1000;

/**
 * the longest an open panel goes without comparing the link's deadline with the clock. One timer
 * set for the whole lifetime would not count the time a computer spends asleep
 */
const EXPIRY_CHECK_INTERVAL = 10 * 1000;

/** a subscription link and the time the server forgets it */
type TelegramLinkState = { token: string; bot: string; expires: number };

export const SubscribeByTelegramForm: FunctionComponent = () => {
  const intl = useIntl();
  const userId = useSelector<StoreState, string>(({ user }) => user?.id ?? '');
  // sessionStorage outlives a sign-out, and the server issues a link to one user of one site, so
  // the state is kept per site and user and the next user of the tab gets a link of their own
  const owner = `${encodeURIComponent(siteId)}/${encodeURIComponent(userId)}`;
  const [step, setStep] = useSessionStorage<STEP>(`telegram-subscription-step/${owner}`, 'initial');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [telegram, setTelegram] = useSessionStorage<TelegramLinkState | null>(
    `telegram-subscription-telegram/${owner}`,
    null
  );

  const fetchTelegram = async (showLoading = true): Promise<boolean> => {
    if (showLoading) {
      setLoading(true);
    }
    try {
      // counted from before the request, so the link is never shown past the server's own deadline
      const expires = Date.now() + LINK_LIFETIME;
      const { token, bot } = await telegramSubscribe();
      setTelegram({ token, bot, expires });
      return true;
    } catch (e) {
      if ((e as RequestError).error === 'already subscribed') {
        setStep('subscribed');
        return false;
      }
      if ((e as RequestError).code === 409) {
        setStep('subscribed');
        return false;
      }
      setError(extractErrorMessageFromResponse(e as FetcherError, intl));
      return false;
    } finally {
      if (showLoading) {
        setLoading(false);
      }
    }
  };

  useEffect(() => {
    if (telegram) {
      return;
    }
    void fetchTelegram();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [telegram]);

  useEffect(() => {
    // the link is on screen only in this step
    if (!telegram || step !== 'initial') {
      return;
    }
    // a link the server has forgotten is dropped, at once when it is already that old and by the
    // timer when it gets there with the panel open. The effect above then asks for a new one.
    // An error on screen came from checking the dropped link, so it goes with it
    const dropExpired = () => {
      if (telegram.expires > Date.now()) {
        return false;
      }
      setError(null);
      setTelegram(null);
      return true;
    };

    if (dropExpired()) {
      return;
    }
    let timer: ReturnType<typeof setTimeout>;
    const checkLater = () => {
      timer = setTimeout(
        () => {
          if (!dropExpired()) {
            checkLater();
          }
        },
        Math.min(EXPIRY_CHECK_INTERVAL, telegram.expires - Date.now())
      );
    };

    checkLater();
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [telegram, step]);

  // the link a check was made for can be replaced before the answer arrives
  const shownToken = useRef<string | undefined>(undefined);
  shownToken.current = telegram?.token;

  const fetchTelegramSubscription = async (token: string) => {
    try {
      setTimeout(() => {
        // to prevent dropdown closing
        setLoading(true);
      }, 0);
      const { updated } = await telegramCurrentSubscribtion({ token });
      setError(null);
      if (!updated) {
        return;
      }
      setStep('subscribed');
    } catch (err) {
      // a refusal of a link that has been replaced meanwhile says nothing about the one on screen
      if (shownToken.current === token) {
        setError(extractErrorMessageFromResponse(err, intl));
      }
    } finally {
      setLoading(false);
    }
  };

  const handleTelegramCheck = () => {
    if (!telegram) {
      return;
    }
    fetchTelegramSubscription(telegram.token);
  };

  const handleTelegramUnsubscribe = async () => {
    try {
      setTimeout(() => {
        // to prevent dropdown closing
        setLoading(true);
      }, 0);
      await telegramUnsubcribe();
      setError(null);
      setStep('unsubscribed');
    } catch (err) {
      setError(extractErrorMessageFromResponse(err, intl));
    } finally {
      setLoading(false);
    }
  };

  const handleTelegramResubscribe = async () => {
    setError(null);
    if (await fetchTelegram(false)) {
      setStep('initial');
    }
  };
  return (
    <div className={clsx(styles.root)}>
      {loading && <Preloader className={clsx(styles.preloader)} />}
      {!loading && error && (!telegram || step !== 'initial') && <div className="auth-error">{error}</div>}
      {!loading && telegram && step === 'initial' && (
        <TelegramLink errorMessage={error} onSubmit={handleTelegramCheck} bot={telegram.bot} token={telegram.token} />
      )}
      {!loading && step === 'subscribed' && (
        <>
          <p>{intl.formatMessage(messages.haveSubscribed)}</p>
          <Button kind="primary" size="large" onClick={handleTelegramUnsubscribe}>
            {intl.formatMessage(messages.unsubscribe)}
          </Button>
        </>
      )}
      {!loading && step === 'unsubscribed' && (
        <>
          <p>{intl.formatMessage(messages.haveUnsubscribed)}</p>
          <Button kind="primary" size="large" onClick={handleTelegramResubscribe}>
            {intl.formatMessage(messages.resubscribe)}
          </Button>
        </>
      )}
    </div>
  );
};

export const SubscribeByTelegram: FunctionComponent = () => {
  const theme = useTheme();
  const intl = useIntl();
  const user = useSelector<StoreState, User | null>(({ user }) => user);
  const isAnonymous = isUserAnonymous(user);
  const buttonTitle = intl.formatMessage(isAnonymous ? messages.onlyRegisteredUsers : messages.subscribeByTelegram);

  return (
    <Dropdown
      title={intl.formatMessage(messages.telegram)}
      theme={theme}
      disabled={isAnonymous}
      buttonTitle={buttonTitle}
    >
      <SubscribeByTelegramForm />
    </Dropdown>
  );
};
