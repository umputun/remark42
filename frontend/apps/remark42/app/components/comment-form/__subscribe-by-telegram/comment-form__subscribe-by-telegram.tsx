import clsx from 'clsx';
import { h, FunctionComponent, Fragment } from 'preact';
import { useState, useEffect } from 'preact/hooks';
import { useSelector } from 'react-redux';
import { useIntl, defineMessages } from 'react-intl';

import { User } from 'common/types';
import { StoreState } from 'store';
import { FetcherError, RequestError, extractErrorMessageFromResponse } from 'utils/errorUtils';
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
    defaultMessage: 'You have been subscribed on updates by telegram',
  },
  haveUnsubscribed: {
    id: 'subscribeByTelegram.have-been-unsubscribed',
    defaultMessage: 'You have been unsubscribed by telegram to updates',
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
    defaultMessage: 'Available only for registered users',
  },
  telegram: {
    id: 'subscribeByTelegram.telegram',
    defaultMessage: 'Telegram',
  },
});

type STEP = 'initial' | 'subscribed' | 'unsubscribed';

export const SubscribeByTelegramForm: FunctionComponent = () => {
  const intl = useIntl();
  const [step, setStep] = useSessionStorage<STEP>('telegram-subscription-step', 'initial');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const [telegram, setTelegram] = useSessionStorage<{ token: string; bot: string }>('telegram-subscription-telegram');

  useEffect(() => {
    const fetchTelegram = async () => {
      setLoading(true);
      try {
        const { token, bot } = await telegramSubscribe();
        setTelegram({ token, bot });
      } catch (e) {
        if ((e as RequestError).error === 'already subscribed') {
          setStep('subscribed');
          return;
        }
        if ((e as RequestError).code === 409) {
          setStep('subscribed');
          return;
        }
        setError(extractErrorMessageFromResponse(e as FetcherError, intl));
      } finally {
        setLoading(false);
      }
    };
    if (telegram) {
      return;
    }
    fetchTelegram();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [telegram]);

  const fetchTelegramSubscription = async (token: string) => {
    try {
      setTimeout(() => {
        // to prevent dropdown closing
        setLoading(true);
      }, 0);
      const { updated } = await telegramCurrentSubscribtion({ token });
      if (!updated) {
        return;
      }
      setStep('subscribed');
    } catch (e) {
      setError(extractErrorMessageFromResponse(e as FetcherError, intl));
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
      setStep('unsubscribed');
    } catch (e) {
      setError(extractErrorMessageFromResponse(e as FetcherError, intl));
    } finally {
      setLoading(false);
    }
  };

  const handleTelegramResubscribe = async () => {
    setStep('initial');
  };
  return (
    <div className={clsx(styles.root)}>
      {loading && <Preloader className={clsx(styles.preloader)} />}
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
