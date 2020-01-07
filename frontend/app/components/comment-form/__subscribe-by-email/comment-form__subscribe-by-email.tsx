/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';
import { useState, useCallback, useEffect, useRef } from 'preact/hooks';
import { useSelector, useDispatch } from 'react-redux';
import b from 'bem-react-helper';

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
import { Preloader } from '@app/components/preloader';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';

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
) => [
  <div className="comment-form__subscribe-by-email__title">Subscribe to replies</div>,
  <Input
    ref={emailAddressRef}
    mix="comment-form__subscribe-by-email__input"
    placeholder="Email"
    value={emailAddress}
    onInput={handleChangeEmail}
    disabled={loading}
  />,
];

const renderTokenPart = (
  loading: boolean,
  token: string,
  handleChangeToken: (e: Event) => void,
  setEmailStep: () => void
) => [
  <Button kind="link" mix="auth-panel-email-login-form__back-button" {...getHandleClickProps(setEmailStep)}>
    Back
  </Button>,
  <TextareaAutosize
    className="comment-form__subscribe-by-email__token-input"
    placeholder="Token"
    autofocus
    onInput={handleChangeToken}
    disabled={loading}
    value={token}
  />,
];

export const SubscribeByEmail: FunctionComponent = () => {
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

  const handleChangeEmail = useCallback((e: Event) => {
    const value = (e.target as HTMLInputElement).value;

    e.preventDefault();
    setError(null);
    setEmailAddress(value);
  }, []);

  const handleChangeToken = useCallback((e: Event) => {
    const value = (e.target as HTMLInputElement).value;

    e.preventDefault();
    setError(null);
    setToken(value);
  }, []);

  const handleSubmit = async (e: Event) => {
    e.preventDefault();
    setLoading(true);
    setError(null);

    try {
      switch (step) {
        case Step.Email:
          await emailVerificationForSubscribe(emailAddress);
          setStep(Step.Token);
          break;
        case Step.Token:
          await emailConfirmationForSubscribe(token);
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
  };

  const isValidEmailAddress = emailRegex.test(emailAddress);

  useEffect(() => {
    if (emailAddressRef.current) {
      emailAddressRef.current.focus();
    }
  }, []);

  const setEmailStep = useCallback(async () => {
    await sleep(0);
    setStep(Step.Email);
  }, [setStep]);

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
