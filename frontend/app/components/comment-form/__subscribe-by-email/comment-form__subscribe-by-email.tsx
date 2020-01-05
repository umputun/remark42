/** @jsx createElement */
import { createElement, FunctionComponent } from 'preact';
import { useState, useCallback, useEffect, useRef } from 'preact/hooks';

import { sleep } from '@app/utils/sleep';
import { getHandleClickProps } from '@app/common/accessibility';
import { sendEmailVerificationForSubscribe, sendEmailConformationForSubscribe } from '@app/common/api';
import { Input } from '@app/components/input';
import { Button } from '@app/components/button';
import TextareaAutosize from '@app/components/comment-form/textarea-autosize';
import { Dropdown } from '@app/components/dropdown';
import Preloader from '@app/components/preloader';
import useTheme from '@app/hooks/useTheme';

const emailRegex = /[^@]+@[^.]+\..+/;

const EmailStep: FunctionComponent<{ onEmailSended(): void }> = ({ onEmailSended }) => {
  const emailAddressRef = useRef<HTMLInputElement>();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [emailAddress, setEmailAddress] = useState('sd@x101.pw');
  const handleChange = useCallback((e: Event) => {
    const value = (e.target as HTMLInputElement).value;

    e.preventDefault();
    setEmailAddress(value);
  }, []);

  const handleSubmit = useCallback(
    async (e: Event) => {
      e.preventDefault();
      setLoading(true);

      try {
        await sendEmailVerificationForSubscribe(emailAddress);
        onEmailSended();
      } catch (e) {
        setError('Something went wrong.');
      } finally {
        setLoading(false);
      }
    },
    [onEmailSended]
  );

  const isValidEmailAddress = emailRegex.test(emailAddress);

  useEffect(() => {
    if (emailAddressRef.current) {
      emailAddressRef.current.focus();
    }
  }, []);

  return (
    <form className="comment-form__subscribe-by-email" onSubmit={handleSubmit}>
      <div className="comment-form__subscribe-by-email__title">Subscribe to replies</div>
      <Input
        ref={emailAddressRef}
        mix="comment-form__subscribe-by-email__input"
        placeholder="Email"
        value={emailAddress}
        onInput={handleChange}
        disabled={loading}
      />
      {error !== null && <div className="comment-form__subscribe-by-email__error">{error}</div>}
      <Button
        mix="comment-form__subscribe-by-email__button"
        kind="primary"
        size="large"
        type="submit"
        disabled={!isValidEmailAddress || loading}
      >
        {loading ? <Preloader mix="comment-form__subscribe-by-email__preloader" /> : 'Submit'}
      </Button>
    </form>
  );
};

interface TokenStepProps {
  onSubmit(e: Event): void;
  onGoBack(e: Event): void;
}

const TokenStep: FunctionComponent<TokenStepProps> = ({ onGoBack }) => {
  const [token, setToken] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleChange = useCallback(
    (e: Event) => {
      setToken((e.target as HTMLTextAreaElement).value);
    },
    [setToken]
  );

  const handleSubmit = useCallback(
    async (e: Event) => {
      e.preventDefault();

      try {
        setLoading(true);
        await sendEmailConformationForSubscribe(token);
      } catch (e) {
        setError('Somthing went wrong');
      } finally {
        setLoading(false);
      }
    },
    [setLoading, setError, token]
  );

  return (
    <form className="comment-form__subscribe-by-email" onSubmit={handleSubmit}>
      <Button kind="link" mix="auth-panel-email-login-form__back-button" {...getHandleClickProps(onGoBack)}>
        Back
      </Button>
      <TextareaAutosize
        className="comment-form__subscribe-by-email__token-input"
        placeholder="Token"
        autofocus
        onInput={handleChange}
      />
      {error !== null && <div className="comment-form__subscribe-by-email__error">{error}</div>}
      <Button
        mix="comment-form__subscribe-by-email__button"
        kind="primary"
        size="large"
        type="submit"
        disabled={loading}
      >
        {loading ? <Preloader mix="comment-form__subscribe-by-email__preloader" /> : 'Subscribe'}
      </Button>
    </form>
  );
};

export const SubscribeByEmail: FunctionComponent = () => {
  const theme = useTheme();
  const [emailStep, setEmailStep] = useState(true);

  const setNextStep = useCallback(() => {
    setEmailStep(false);
  }, [setEmailStep]);

  const handleSubmitToken = useCallback((e: Event) => {
    e.preventDefault();
  }, []);

  const handleGoBack = useCallback(
    async (e: Event) => {
      e.preventDefault();
      await sleep(0);
      setEmailStep(true);
    },
    [setEmailStep]
  );

  return (
    <Dropdown title="Email" theme={theme}>
      {emailStep ? (
        <EmailStep onEmailSended={setNextStep} />
      ) : (
        <TokenStep onSubmit={handleSubmitToken} onGoBack={handleGoBack} />
      )}
    </Dropdown>
  );
};
