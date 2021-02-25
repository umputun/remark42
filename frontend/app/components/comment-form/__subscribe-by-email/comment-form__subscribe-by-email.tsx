import { h, FunctionComponent, Fragment } from 'preact';
import { useState, useCallback, useRef } from 'preact/hooks';
import { useSelector, useDispatch } from 'react-redux';
import b from 'bem-react-helper';
import { useIntl, defineMessages, IntlShape, FormattedMessage } from 'react-intl';

import { User } from 'common/types';
import { LS_EMAIL_KEY } from 'common/constants';
import { StoreState } from 'store';
import { setUserSubscribed } from 'store/user/actions';
import { sleep } from 'utils/sleep';
import { extractErrorMessageFromResponse } from 'utils/errorUtils';
import useTheme from 'hooks/useTheme';
import { getHandleClickProps } from 'common/accessibility';
import { emailVerificationForSubscribe, emailConfirmationForSubscribe, unsubscribeFromEmailUpdates } from 'common/api';
import { Input } from 'components/input';
import { Button } from 'components/button';
import { Dropdown } from 'components/dropdown';
import Preloader from 'components/preloader';
import TextareaAutosize from 'components/textarea-autosize';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { isJwtExpired } from 'utils/jwt';

const emailRegexp = /[^@]+@[^.]+\..+/;

enum Step {
  Email,
  Token,
  Final,
  Close,
  Subscribed,
  Unsubscribed,
}

const messages = defineMessages({
  token: {
    id: 'token',
    defaultMessage: 'Token',
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

const renderEmailPart = (
  loading: boolean,
  intl: IntlShape,
  emailAddress: string,
  handleChangeEmail: (e: Event) => void
) => (
  <>
    <div className="comment-form__subscribe-by-email__title">
      <FormattedMessage id="subscribeByEmail.subscribe-to-replies" defaultMessage="Subscribe to replies" />
    </div>
    <Input
      autofocus
      className="comment-form__subscribe-by-email__input"
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
    <Button kind="link" mix="auth-email-login-form__back-button" {...getHandleClickProps(setEmailStep)}>
      <FormattedMessage id="subscribeByEmail.back" defaultMessage="Back" />
    </Button>
    <TextareaAutosize
      className="comment-form__subscribe-by-email__token-input"
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
  const subscribed = useSelector<StoreState, boolean>(({ user }) =>
    user === null ? false : Boolean(user.email_subscription)
  );
  const previousStep = useRef<Step | null>(null);

  const [step, setStep] = useState(subscribed ? Step.Subscribed : Step.Email);

  const [token, setToken] = useState('');
  const [emailAddress, setEmailAddress] = useState(localStorage.getItem(LS_EMAIL_KEY) || '');

  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendForm = useCallback(
    async (currentToken: string = token) => {
      setLoading(true);
      setError(null);

      try {
        switch (step) {
          case Step.Email:
            await emailVerificationForSubscribe(emailAddress);
            setToken('');
            setStep(Step.Token);
            break;
          case Step.Token:
            await emailConfirmationForSubscribe(currentToken);
            dispatch(setUserSubscribed(true));
            previousStep.current = Step.Token;
            setStep(Step.Subscribed);
            break;
          default:
            break;
        }
      } catch (e) {
        setError(extractErrorMessageFromResponse(e, intl));
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
      previousStep.current = Step.Subscribed;
      setStep(Step.Unsubscribed);
    } catch (e) {
      setError(extractErrorMessageFromResponse(e, intl));
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
    const text =
      previousStep.current === Step.Token
        ? intl.formatMessage(messages.haveSubscribed)
        : intl.formatMessage(messages.subscribed);

    return (
      <div className={b('comment-form__subscribe-by-email', { mods: { subscribed: true } })}>
        {text}
        <Button
          kind="primary"
          size="middle"
          mix="comment-form__subscribe-by-email__button"
          theme={theme}
          onClick={handleUnsubscribe}
        >
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
      <div className={b('comment-form__subscribe-by-email', { mods: { unsubscribed: true } })}>
        <FormattedMessage
          id="subscribeByEmail.have-been-unsubscribed"
          defaultMessage="You have been unsubscribed by email to updates"
        />
        <Button
          kind="primary"
          size="middle"
          mix="comment-form__subscribe-by-email__button"
          theme={theme}
          onClick={() => setStep(Step.Close)}
        >
          <FormattedMessage id="subscribeByEmail.close" defaultMessage="Close" />
        </Button>
      </div>
    );
  }

  const buttonLabel =
    step === Step.Email ? intl.formatMessage(messages.submit) : intl.formatMessage(messages.subscribe);

  return (
    <form className={b('comment-form__subscribe-by-email', {}, { theme })} onSubmit={handleSubmit}>
      {step === Step.Email && renderEmailPart(loading, intl, emailAddress, handleChangeEmail)}
      {step === Step.Token && renderTokenPart(loading, intl, token, handleChangeToken, setEmailStep)}
      {error !== null && (
        <div className="comment-form__subscribe-by-email__error" role="alert">
          {error}
        </div>
      )}
      <Button
        mix="comment-form__subscribe-by-email__button"
        kind="primary"
        size="large"
        type="submit"
        disabled={!isValidEmailAddress || loading}
      >
        {loading ? <Preloader mix="comment-form__subscribe-by-email__preloader" /> : buttonLabel}
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
    <Dropdown
      mix="comment-form__email-dropdown"
      title={intl.formatMessage(messages.email)}
      theme={theme}
      disabled={isAnonymous}
      buttonTitle={buttonTitle}
    >
      <SubscribeByEmailForm />
    </Dropdown>
  );
};
