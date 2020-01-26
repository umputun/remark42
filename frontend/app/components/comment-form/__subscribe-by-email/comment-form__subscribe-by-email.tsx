/** @jsx createElement */
import { createElement, FunctionComponent, Fragment } from 'preact';
import { useState, useCallback, useEffect, useRef } from 'preact/hooks';
import { useSelector, useDispatch } from 'react-redux';
import b from 'bem-react-helper';

import { User } from '@app/common/types';
import { StoreState } from '@app/store';
import { setUserSubscribed } from '@app/store/user/actions';
import { sleep } from '@app/utils/sleep';
import { extractErrorMessageFromResponse } from '@app/utils/errorUtils';
import useTheme from '@app/hooks/useTheme';
import { getHandleClickProps } from '@app/common/accessibility';
import {
  emailVerificationForSubscribe,
  emailConfirmationForSubscribe,
  unsubscribeFromEmailUpdates,
} from '@app/common/api';
import { Input } from '@app/components/input';
import { Button } from '@app/components/button';
import { Dropdown } from '@app/components/dropdown';
import { Preloader } from '@app/components/preloader';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';
import { isUserAnonymous } from '@app/utils/isUserAnonymous';
import { isJwtExpired } from '@app/utils/jwt';

const emailRegex = /[^@]+@[^.]+\..+/;

enum Step {
  Email,
  Token,
  Final,
  Close,
  Subscribed,
  Unsubscribed,
}

const renderEmailPart = (
  loading: boolean,
  emailAddress: string,
  handleChangeEmail: (e: Event) => void,
  emailAddressRef: ReturnType<typeof useRef>
) => (
  <Fragment>
    <div className="comment-form__subscribe-by-email__title">Subscribe to replies</div>
    <Input
      ref={emailAddressRef}
      mix="comment-form__subscribe-by-email__input"
      placeholder="Email"
      value={emailAddress}
      onInput={handleChangeEmail}
      disabled={loading}
    />
  </Fragment>
);

const renderTokenPart = (
  loading: boolean,
  token: string,
  handleChangeToken: (e: Event) => void,
  setEmailStep: () => void
) => (
  <Fragment>
    <Button kind="link" mix="auth-panel-email-login-form__back-button" {...getHandleClickProps(setEmailStep)}>
      Back
    </Button>
    <TextareaAutosize
      className="comment-form__subscribe-by-email__token-input"
      placeholder="Token"
      autofocus
      onInput={handleChangeToken}
      disabled={loading}
      value={token}
    />
  </Fragment>
);

export const SubscribeByEmailForm: FunctionComponent = () => {
  const theme = useTheme();
  const dispatch = useDispatch();
  const subscribed = useSelector<StoreState, boolean>(({ user }) =>
    user === null ? false : Boolean(user.email_subscription)
  );
  const emailAddressRef = useRef<HTMLInputElement>();
  const previousStep = useRef<Step | null>(null);

  const [step, setStep] = useState(subscribed ? Step.Subscribed : Step.Email);

  const [token, setToken] = useState('');
  const [emailAddress, setEmailAddress] = useState('');

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
        setError(extractErrorMessageFromResponse(e));
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, setStep, step, emailAddress, token]
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
          setError('Token is expired');
        } else {
          sendForm(value);
        }
      } catch (e) {}

      setToken(value);
    },
    [sendForm, setError, setToken]
  );

  const handleSubmit = useCallback(
    async (e: Event) => {
      e.preventDefault();
      sendForm();
    },
    [sendForm]
  );

  const isValidEmailAddress = emailRegex.test(emailAddress);

  const setEmailStep = useCallback(async () => {
    await sleep(0);
    setError(null);
    setStep(Step.Email);
  }, [setStep]);

  useEffect(() => {
    if (emailAddressRef.current) {
      emailAddressRef.current.focus();
    }
  }, []);

  /**
   * It needs for dropdown closing by click on button
   * More info below
   */
  if (step === Step.Close) {
    return null;
  }

  if (step === Step.Subscribed) {
    const handleUnsubscribe = useCallback(async () => {
      setLoading(true);
      try {
        await unsubscribeFromEmailUpdates();
        dispatch(setUserSubscribed(false));
        previousStep.current = Step.Subscribed;
        setStep(Step.Unsubscribed);
      } catch (e) {
        setError(extractErrorMessageFromResponse(e));
      } finally {
        setLoading(false);
      }
    }, [setLoading, setStep, setError]);

    const text =
      previousStep.current === Step.Token
        ? 'You have been subscribed on updates by email'
        : 'You are subscribed on updates by email';

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
          Unsubscribe
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
        You have been unsubscribed by email to updates
        <Button
          kind="primary"
          size="middle"
          mix="comment-form__subscribe-by-email__button"
          theme={theme}
          onClick={() => setStep(Step.Close)}
        >
          Close
        </Button>
      </div>
    );
  }

  const buttonLabel = step === Step.Email ? 'Submit' : 'Subscribe';

  return (
    <form className={b('comment-form__subscribe-by-email', {}, { theme })} onSubmit={handleSubmit}>
      {step === Step.Email && renderEmailPart(loading, emailAddress, handleChangeEmail, emailAddressRef)}
      {step === Step.Token && renderTokenPart(loading, token, handleChangeToken, setEmailStep)}
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
  const user = useSelector<StoreState, User | null>(({ user }) => user);
  const isAnonymous = isUserAnonymous(user);
  const buttonTitle = isAnonymous ? 'Available only for registered users' : 'Subscribe by Email';

  return (
    <Dropdown
      mix="comment-form__email-dropdown"
      title="Email"
      theme={theme}
      disabled={isAnonymous}
      buttonTitle={buttonTitle}
    >
      <SubscribeByEmailForm />
    </Dropdown>
  );
};
