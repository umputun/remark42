import { h, FunctionComponent, Fragment } from 'preact';
import { useState, useCallback, useRef } from 'preact/hooks';
import { useSelector, useDispatch } from 'react-redux';
import b from 'bem-react-helper';
import { useIntl, defineMessages, IntlShape, FormattedMessage } from 'react-intl';
import clsx from 'clsx';

import { User } from 'common/types';
import { StoreState } from 'store';
import { setUserSubscribed } from 'store/user/actions';
import { sleep } from 'utils/sleep';
import { extractErrorMessageFromResponse, RequestError } from 'utils/errorUtils';
import { getHandleClickProps } from 'common/accessibility';
import { emailVerificationForSubscribe, emailConfirmationForSubscribe, unsubscribeFromEmailUpdates } from 'common/api';
import { Input } from 'components/input';
import { Button } from 'components/button';
import { Dropdown } from 'components/dropdown';
import { Preloader } from 'components/preloader';
import { TextareaAutosize } from 'components/textarea-autosize';
import { getPersistedEmail } from 'components/auth/auth.utils';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { isJwtExpired } from 'utils/jwt';
import { useTheme } from 'hooks/useTheme';

import styles from './subscribe-by-email.module.css';

const emailRegexp = /[^@]+@[^.]+\..+/;

enum Step {
  Email,
  Token,
  Close,
  Subscribed,
  Unsubscribed,
}

export const SubscribeByEmailForm: FunctionComponent = () => {
  const dispatch = useDispatch();
  const intl = useIntl();
  const subscribed = useSelector<StoreState, boolean>(({ user }) =>
    user === null ? false : Boolean(user.email_subscription)
  );
  const justSubscribed = useRef<Boolean>(false);

  const [step, setStep] = useState(subscribed ? Step.Subscribed : Step.Email);

  const [token, setToken] = useState('');
  const [emailAddress, setEmailAddress] = useState(getPersistedEmail);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendForm = useCallback(
    async (currentToken: string = token) => {
      setLoading(true);
      setError(null);

      try {
        let emailVerificationResponse;

        switch (step) {
          case Step.Email:
            try {
              emailVerificationResponse = await emailVerificationForSubscribe(emailAddress);
              justSubscribed.current = true;
            } catch (e) {
              if ((e as RequestError).code !== 409) {
                throw e;
              }
              emailVerificationResponse = { address: emailAddress, updated: true };
            }
            if (emailVerificationResponse.updated) {
              dispatch(setUserSubscribed(true));
              setStep(Step.Subscribed);
              break;
            }
            setToken('');
            setStep(Step.Token);
            break;
          case Step.Token:
            await emailConfirmationForSubscribe(currentToken);
            dispatch(setUserSubscribed(true));
            justSubscribed.current = true;
            setStep(Step.Subscribed);
            break;
          default:
            break;
        }
      } catch (err) {
        setError(extractErrorMessageFromResponse(err, intl));
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, setStep, step, emailAddress, token, dispatch, intl]
  );

  const handleChangeEmail = useCallback((e: Event) => {
    const { value } = e.target as HTMLInputElement;

    e.preventDefault();
    setError(null);
    setEmailAddress(value);
  }, []);

  const handleChangeToken = useCallback(
    (e: Event) => {
      const { value } = e.target as HTMLInputElement;

      e.preventDefault();
      setError(null);

      try {
        if (value.length > 0 && isJwtExpired(value)) {
          setError(intl.formatMessage(messages.expiredToken));
        } else {
          sendForm(value);
        }
      } catch (e) {}

      setToken(value);
    },
    [sendForm, setError, setToken, intl]
  );

  const handleSubmit = useCallback(
    async (e: Event) => {
      e.preventDefault();
      sendForm();
    },
    [sendForm]
  );

  const isValidEmailAddress = emailRegexp.test(emailAddress);

  const setEmailStep = useCallback(async () => {
    await sleep(0);
    setError(null);
    setStep(Step.Email);
  }, [setStep]);

  const handleUnsubscribe = useCallback(async () => {
    setLoading(true);
    try {
      await unsubscribeFromEmailUpdates();
      dispatch(setUserSubscribed(false));
      setStep(Step.Unsubscribed);
    } catch (err) {
      setError(extractErrorMessageFromResponse(err, intl));
    } finally {
      setLoading(false);
    }
  }, [setLoading, setStep, setError, dispatch, intl]);

  /**
   * It needs for dropdown closing by click on button
   * More info below
   */
  if (step === Step.Close) {
    return null;
  }

  if (step === Step.Subscribed) {
    const text = justSubscribed.current
      ? intl.formatMessage(messages.haveSubscribed)
      : intl.formatMessage(messages.subscribed);

    return (
      <div className={clsx(styles.root, styles.rootSubscribed)}>
        {text}
        <Button kind="primary" size="middle" className={styles.button} onClick={handleUnsubscribe}>
          <FormattedMessage id="subscribeByEmail.unsubscribe" defaultMessage="Unsubscribe" />
        </Button>
      </div>
    );
  }

  if (step === Step.Unsubscribed) {
    /**
     * It works because click on button changes step
     * And dropdown doesn't find event target in rerendered view
     * NOTE: If you can suggest more elegant solve you can open issue or PR
     */

    return (
      <div className={clsx(styles.root, styles.rootUnsubscribed)}>
        <FormattedMessage
          id="subscribeByEmail.have-been-unsubscribed"
          defaultMessage="You have been unsubscribed by email to updates"
        />
        <Button kind="primary" size="middle" className={styles.button} onClick={() => setStep(Step.Close)}>
          <FormattedMessage id="subscribeByEmail.close" defaultMessage="Close" />
        </Button>
      </div>
    );
  }

  const buttonLabel =
    step === Step.Email ? intl.formatMessage(messages.submit) : intl.formatMessage(messages.subscribe);

  return (
    <form className={styles.root} onSubmit={handleSubmit}>
      {step === Step.Email && (
        <>
          <div className={styles.title}>
            <FormattedMessage id="subscribeByEmail.subscribe-to-replies" defaultMessage="Subscribe to replies" />
          </div>
          <Input
            autofocus
            className={styles.input}
            placeholder={intl.formatMessage(messages.email)}
            value={emailAddress}
            onInput={handleChangeEmail}
            disabled={loading}
          />
        </>
      )}
      {step === Step.Token && (
        <>
          <Button kind="link" {...getHandleClickProps(setEmailStep)}>
            <FormattedMessage id="subscribeByEmail.back" defaultMessage="Back" />
          </Button>
          <TextareaAutosize
            placeholder={intl.formatMessage(messages.token)}
            autofocus
            onInput={handleChangeToken}
            disabled={loading}
            value={token}
          />
        </>
      )}
      {error !== null && (
        <div className={styles.error} role="alert">
          {error}
        </div>
      )}
      <Button
        className={styles.button}
        kind="primary"
        size="large"
        type="submit"
        disabled={!isValidEmailAddress || loading}
      >
        {loading ? <Preloader className={styles.preloader} /> : buttonLabel}
      </Button>
    </form>
  );
};

export const SubscribeByEmail: FunctionComponent = () => {
  const intl = useIntl();
  const theme = useTheme();
  const user = useSelector<StoreState, User | null>(({ user }) => user);
  const isAnonymous = isUserAnonymous(user);
  const buttonTitle = intl.formatMessage(isAnonymous ? messages.onlyRegisteredUsers : messages.subscribeByEmail);

  return (
    <Dropdown theme={theme} title={intl.formatMessage(messages.email)} disabled={isAnonymous} buttonTitle={buttonTitle}>
      <SubscribeByEmailForm />
    </Dropdown>
  );
};

const messages = defineMessages({
  token: {
    id: 'token',
    defaultMessage: 'Copy and paste the token from the email',
  },
  expiredToken: {
    id: 'token.expired',
    defaultMessage: 'Token is expired',
  },
  haveSubscribed: {
    id: 'subscribeByEmail.have-been-subscribed',
    defaultMessage: 'You have been subscribed on updates by email',
  },
  subscribed: {
    id: 'subscribeByEmail.subscribed',
    defaultMessage: 'You are subscribed on updates by email',
  },
  submit: {
    id: 'subscribeByEmail.submit',
    defaultMessage: 'Submit',
  },
  subscribe: {
    id: 'subscribeByEmail.subscribe',
    defaultMessage: 'Subscribe',
  },
  subscribeByEmail: {
    id: 'subscribeByEmail.subscribe-by-email',
    defaultMessage: 'Subscribe by Email',
  },
  onlyRegisteredUsers: {
    id: 'subscribeByEmail.only-registered-users',
    defaultMessage: 'Available only for registered users',
  },
  email: {
    id: 'subscribeByEmail.email',
    defaultMessage: 'Email',
  },
});
