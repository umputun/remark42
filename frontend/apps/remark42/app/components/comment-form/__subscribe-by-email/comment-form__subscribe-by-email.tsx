import type { FunctionComponent } from 'preact';
import { h, Fragment } from 'preact';
import { useState, useCallback, useEffect, useLayoutEffect, useRef } from 'preact/hooks';
import { useSelector, useDispatch } from 'store/context';
import clsx from 'clsx';

import type { IntlShape } from 'common/intl';
import { useIntl, defineMessages, FormattedMessage } from 'common/intl';
import type { User } from 'common/types';
import type { StoreState } from 'store';
import { setUserSubscribed } from 'store/user/actions';
import type { RequestError } from 'utils/errorUtils';
import { extractErrorMessageFromResponse } from 'utils/errorUtils';
import { useTheme } from 'hooks/useTheme';
import { getHandleClickProps } from 'common/accessibility';
import {
  emailVerificationForSubscribe,
  emailConfirmationForSubscribe,
  unsubscribeFromEmailUpdates,
  getEmailSubscription,
} from 'common/api';
import { Input } from 'components/input';
import { Button } from 'components/button';
import { Dropdown, useDropdown } from 'components/dropdown';
import { Preloader } from 'components/preloader';
import { TextareaAutosize } from 'components/textarea-autosize';
import { getPersistedEmail, isCompleteToken } from 'components/auth/auth.utils';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { isJwtExpired } from 'utils/jwt';

import styles from './subscribe-by-email.module.css';

const emailRegexp = /[^@]+@[^.]+\..+/;

function sameAddress(a: string, b: string): boolean {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

// how often the token step asks whether the link in the email has confirmed the subscription
export const CONFIRMATION_POLL_INTERVAL = 10 * 1000;
// how long it keeps asking: the confirmation token the email carries expires after 30 minutes
export const CONFIRMATION_POLL_DURATION = 30 * 60 * 1000;

enum Step {
  Email,
  Token,
  Subscribed,
  Unsubscribed,
}

const messages = defineMessages({
  token: {
    id: 'subscribeByEmail.token',
    defaultMessage: 'Paste the token from the email, or click the link in it',
  },
  expiredToken: {
    id: 'token.expired',
    defaultMessage: 'Token is expired',
  },
  haveSubscribed: {
    id: 'subscribeByEmail.have-been-subscribed',
    defaultMessage: 'You have subscribed to email updates',
  },
  subscribed: {
    id: 'subscribeByEmail.subscribed',
    defaultMessage: 'You are subscribed to email updates',
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
    defaultMessage: 'Subscribe by email',
  },
  onlyRegisteredUsers: {
    id: 'subscribeByEmail.only-registered-users',
    defaultMessage: 'Only available to registered users',
  },
  email: {
    id: 'subscribeByEmail.email',
    defaultMessage: 'Email',
  },
});

const renderEmailPart = (
  loading: boolean,
  intl: IntlShape,
  emailAddress: string,
  handleChangeEmail: (e: Event) => void
) => (
  <>
    <div className={styles.title}>
      <FormattedMessage id="subscribeByEmail.subscribe-to-replies" defaultMessage="Email subscription to replies" />
    </div>
    <div className={styles.description}>
      <FormattedMessage
        id="subscribeByEmail.subscribe-to-replies-description"
        defaultMessage="Get an email about every reply to your comments and to the discussions under them."
      />
    </div>
    <Input
      autofocus
      placeholder={intl.formatMessage(messages.email)}
      value={emailAddress}
      onInput={handleChangeEmail}
      disabled={loading}
    />
  </>
);

const renderTokenPart = (
  loading: boolean,
  intl: IntlShape,
  token: string,
  handleChangeToken: (e: Event) => void,
  setEmailStep: () => void
) => (
  <>
    <Button kind="link" {...getHandleClickProps(setEmailStep)}>
      <FormattedMessage id="subscribeByEmail.back" defaultMessage="Back" />
    </Button>
    <TextareaAutosize
      className={styles.tokenInput}
      placeholder={intl.formatMessage(messages.token)}
      autofocus
      onInput={handleChangeToken}
      disabled={loading}
      value={token}
    />
  </>
);

export const SubscribeByEmailForm: FunctionComponent = () => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const intl = useIntl();
  const { close, keepOpen } = useDropdown();
  const subscribed = useSelector<StoreState, boolean>(({ user }) =>
    user === null ? false : Boolean(user.email_subscription)
  );
  const userId = useSelector<StoreState, string | undefined>(({ user }) => user?.id);
  const justSubscribed = useRef<Boolean>(false);

  const [step, setStep] = useState(subscribed ? Step.Subscribed : Step.Email);

  const [token, setToken] = useState('');
  const [emailAddress, setEmailAddress] = useState(getPersistedEmail);

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  // a click elsewhere on the page while the reader is in the mailbox would otherwise drop the token step.
  // A layout effect, so it holds before the token field is painted: an effect runs a frame later, and a
  // click in that frame still closes the panel
  useLayoutEffect(() => {
    keepOpen(step === Step.Token);
    return () => keepOpen(false);
  }, [step, keepOpen]);

  // read by the checks below without restarting them on every keystroke
  const pendingAddress = useRef(emailAddress);
  pendingAddress.current = emailAddress;

  // the link in the email confirms the subscription from another tab or another device, so the form
  // asks the API whether it has happened: once when it opens, and while it waits for the token or has
  // just unsubscribed, each time the page becomes visible again, and every few seconds until the token
  // in the email expires
  useEffect(() => {
    if (step === Step.Subscribed) {
      return undefined;
    }
    let active = true;

    async function check() {
      if (document.hidden) {
        return;
      }
      const subscription = await getEmailSubscription().catch(() => null);
      // another tab may have signed in as someone else since this form was rendered
      if (!active || !subscription?.address || subscription.user.id !== userId) {
        return;
      }
      // the token step waits for one address; a link to an earlier, perhaps mistyped one does not end
      // it, and pasting this step's token still replaces that address
      if (step === Step.Token && !sameAddress(subscription.address, pendingAddress.current)) {
        return;
      }
      justSubscribed.current = step !== Step.Email;
      dispatch(setUserSubscribed(true));
      setStep(Step.Subscribed);
    }

    if (step === Step.Email) {
      void check();
      return () => {
        active = false;
      };
    }

    // the reader may confirm in webmail in this browser and come back later than that, so the check on
    // returning to the page has no deadline; only the timer stops
    const deadline = Date.now() + CONFIRMATION_POLL_DURATION;
    const timer = setInterval(() => {
      if (Date.now() > deadline) {
        clearInterval(timer);
        return;
      }
      void check();
    }, CONFIRMATION_POLL_INTERVAL);
    document.addEventListener('visibilitychange', check);
    return () => {
      active = false;
      clearInterval(timer);
      document.removeEventListener('visibilitychange', check);
    };
  }, [step, dispatch, userId]);

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
              if ((e as RequestError).status !== 409) {
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

      // a token still being pasted or typed is not sent, the server would refuse it
      const pasted = value.trim();

      try {
        if (pasted.length > 0 && isJwtExpired(pasted)) {
          setError(intl.formatMessage(messages.expiredToken));
        } else if (isCompleteToken(pasted)) {
          sendForm(pasted);
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

  const setEmailStep = useCallback(() => {
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

  if (step === Step.Subscribed) {
    const text = justSubscribed.current
      ? intl.formatMessage(messages.haveSubscribed)
      : intl.formatMessage(messages.subscribed);

    return (
      <div className={styles.subscribed}>
        {text}
        <Button kind="primary" size="middle" className={styles.button} theme={theme} onClick={handleUnsubscribe}>
          <FormattedMessage id="subscribeByEmail.unsubscribe" defaultMessage="Unsubscribe" />
        </Button>
      </div>
    );
  }

  if (step === Step.Unsubscribed) {
    return (
      <div className={styles.unsubscribed}>
        <FormattedMessage
          id="subscribeByEmail.have-been-unsubscribed"
          defaultMessage="You have unsubscribed from email updates"
        />
        <Button kind="primary" size="middle" className={styles.button} theme={theme} onClick={close}>
          <FormattedMessage id="subscribeByEmail.close" defaultMessage="Close" />
        </Button>
      </div>
    );
  }

  const buttonLabel =
    step === Step.Email ? intl.formatMessage(messages.submit) : intl.formatMessage(messages.subscribe);

  return (
    <form
      className={clsx(styles.root, theme === 'dark' ? styles.themeDark : styles.themeLight)}
      onSubmit={handleSubmit}
    >
      {step === Step.Email && renderEmailPart(loading, intl, emailAddress, handleChangeEmail)}
      {step === Step.Token && renderTokenPart(loading, intl, token, handleChangeToken, setEmailStep)}
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
  const theme = useTheme();
  const intl = useIntl();
  const user = useSelector<StoreState, User | null>(({ user }) => user);
  const isAnonymous = isUserAnonymous(user);
  const buttonTitle = intl.formatMessage(isAnonymous ? messages.onlyRegisteredUsers : messages.subscribeByEmail);

  return (
    <Dropdown title={intl.formatMessage(messages.email)} theme={theme} disabled={isAnonymous} buttonTitle={buttonTitle}>
      <SubscribeByEmailForm />
    </Dropdown>
  );
};
