import clsx from 'clsx';
import type { JSX } from 'preact';
import { h, Fragment } from 'preact';
import { useState, useRef, useLayoutEffect } from 'preact/hooks';
import { useAppDispatch } from 'store';

import { useIntl } from 'common/intl';
import { signin } from 'store/user/actions';
import { getSignedInUser, getUser } from 'common/api';
import { getCookie } from 'common/cookies';
import { XSRF_COOKIE } from 'common/fetcher';
import type { User } from 'common/types';
import { Input } from 'components/input';
import { TelegramLink } from 'components/telegram/telegram-link';
import { CrossIcon } from 'components/icons/cross';
import { TextareaAutosize } from 'components/textarea-autosize';
import { Spinner } from 'components/spinner/spinner';
import { ArrowIcon } from 'components/icons/arrow';
import { RequestError } from 'utils/errorUtils';

import { Button } from './components/button';
import { OAuth } from './components/oauth';
import { messages } from './auth.messages';
import { useDropdown, useErrorMessage } from './auth.hooks';
import { getProviders, getTokenInvalidReason, isCompleteToken, persistEmail } from './auth.utils';
import {
  oauthSignin,
  emailSignin,
  verifyEmailSignin,
  anonymousSignin,
  verifyTelegramSignin,
  getTelegramSigninParams,
} from './auth.api';

// before the form's own styles: the token field takes the input's look and auth.module.css adjusts it
import inputStyles from 'components/input/input.module.css';
import styles from './auth.module.css';

const USERNAME_PATTERN = String.raw`\s*[\p{L}\d_.\-][\p{L}\d _.\-]{1,62}[\p{L}\d_.\-]\s*`;
const EMAIL_USERNAME_PATTERN = String.raw`\s*[\p{L}\d_\-][\p{L}\d _\-]{1,62}[\p{L}\d_\-]\s*`;

export function Auth() {
  const intl = useIntl();
  const telegramParamsRef = useRef<null | { bot: string; token: string }>(null);
  const emailRef = useRef('');
  const isVerifyingRef = useRef(false);
  const isSignedInRef = useRef(false);
  const dispatch = useAppDispatch();
  const [oauthProviders, formProviders] = getProviders();

  // UI State
  const [isLoading, setLoading] = useState(false);
  const [view, setView] = useState<(typeof formProviders)[number] | 'token' | 'telegram'>(formProviders[0]);
  // what the reader has typed and the tab it is on stay while the dropdown is closed and opened
  // again, and the address while the email field is away on another tab or step
  const [formView, setFormView] = useState(formProviders[0]);
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [ref, isDropdownShown, toggleDropdownState] = useDropdown(view === 'token' || view === 'telegram');

  // Errors
  const [errorMessage, setError] = useErrorMessage();

  // the link in the email signs the browser in on a page of its own, and this frame shares that page's
  // cookies wherever the browser lets it. A new XSRF cookie is the sign it happened: reading the cookie
  // costs no request, so the user is loaded only once it has changed. A browser keeps a hidden tab's
  // timers slow, so the check also runs when the reader comes back to it. A layout effect starts
  // watching as the step renders, so a cookie set at once is still seen as new
  useLayoutEffect(() => {
    if (view !== 'token') {
      return undefined;
    }

    let seen = getCookie(XSRF_COOKIE);

    async function check() {
      const current = getCookie(XSRF_COOKIE);

      if (!current || current === seen || isVerifyingRef.current || isSignedInRef.current) {
        return;
      }

      const previous = seen;

      // a cookie that does not lead to a user is not looked at again
      seen = current;
      isVerifyingRef.current = true;

      try {
        const user = await getSignedInUser();

        isSignedInRef.current = true;
        dispatch(signin(user));
      } catch (e) {
        // a refusal is the answer for this cookie, and the token from the email still works. A request
        // that did not get through, on the network or at the rate limit, is made again on the next check
        if (!(e instanceof RequestError && (e.code === 401 || e.code === 403))) {
          seen = previous;
        }
      } finally {
        isVerifyingRef.current = false;
      }
    }

    const timer = window.setInterval(check, 1000);

    document.addEventListener('visibilitychange', check);
    window.addEventListener('focus', check);

    return () => {
      window.clearInterval(timer);
      document.removeEventListener('visibilitychange', check);
      window.removeEventListener('focus', check);
    };
  }, [view, dispatch]);

  function handleClickSingIn(evt: Event) {
    evt.preventDefault();
    toggleDropdownState();
  }

  function resetView() {
    setView(formView);
    setError(null);
  }

  function handleDropdownClose(evt: Event) {
    evt.preventDefault();
    resetView();
    toggleDropdownState();
  }

  function handleClickBack(evt: JSX.TargetedMouseEvent<HTMLButtonElement>) {
    evt.preventDefault();
    resetView();
  }

  async function handleOauthClick(evt: JSX.TargetedMouseEvent<HTMLAnchorElement>) {
    evt.preventDefault();

    const { href, dataset } = evt.currentTarget;

    if (dataset.providerName?.toLowerCase() === 'telegram') {
      try {
        telegramParamsRef.current = await getTelegramSigninParams();
      } catch (e) {
        setError(e);
        return;
      }

      setView('telegram');
      setError(null);
      return;
    }

    try {
      const user = await oauthSignin(href);

      if (user === null) {
        // TODO: add error message when user is null
        return;
      }
      dispatch(signin(user));
    } catch (e) {
      setError(e);
    }
  }

  function handleProviderChange(evt: Event) {
    const value = (evt.currentTarget as HTMLInputElement).value as (typeof formProviders)[number];

    setFormView(value);
    setView(value);
    setError(null);
  }

  /**
   * A confirmation token signs in once, so a second request with the same one is refused. The
   * state that disables the form is set a render later than the events that can ask twice
   */
  async function verifyToken(token: string) {
    if (isVerifyingRef.current || isSignedInRef.current) {
      return;
    }

    isVerifyingRef.current = true;

    try {
      let user: User | null;

      try {
        user = await verifyEmailSignin(token);
        persistEmail(emailRef.current);
      } catch (e) {
        // a token refused as used may be the one that signed this browser in: its first request
        // went through and the user was not loaded after it. The session is what tells. It can
        // also belong to another account signed in from another tab, so the address typed here
        // is not kept for it
        user = await getUser();

        if (user === null) {
          throw e;
        }
      }

      isSignedInRef.current = true;
      dispatch(signin(user));
    } finally {
      isVerifyingRef.current = false;
    }
  }

  /**
   * Signs in as soon as the field holds a whole token, the way the subscription form does. Text
   * that is not one yet is left alone, since it may still be typed, and the submit button reports it
   */
  async function handleTokenInput(evt: JSX.TargetedEvent<HTMLTextAreaElement>) {
    const token = evt.currentTarget.value.trim();

    if (isVerifyingRef.current) {
      return;
    }

    setError(null);

    if (!isCompleteToken(token)) {
      return;
    }

    const invalidReason = getTokenInvalidReason(token);

    if (invalidReason) {
      setError(invalidReason);
      return;
    }

    setLoading(true);

    try {
      await verifyToken(token);
    } catch (e) {
      setError(e);
    }

    setLoading(false);
  }

  async function handleSubmit(evt: Event) {
    const data = new FormData(evt.target as HTMLFormElement);

    evt.preventDefault();
    setLoading(true);
    setError(null);

    try {
      switch (view) {
        case 'anonymous': {
          // the field is trimmed on blur, which pressing Enter skips, and the server refuses a space at either end
          const username = (data.get('username') as string).trim();
          const user = await anonymousSignin(username);

          dispatch(signin(user));
          break;
        }
        case 'email': {
          const email = data.get('email') as string;
          const username = (data.get('username') as string).trim();

          emailRef.current = email;

          await emailSignin(email, username);
          setView('token');
          break;
        }
        case 'token': {
          const token = (data.get('token') as string).trim();
          const invalidReason = getTokenInvalidReason(token);

          if (invalidReason) {
            setError(invalidReason);
            break;
          }

          await verifyToken(token);

          break;
        }
      }
    } catch (e) {
      setError(e);
    }

    setLoading(false);
  }

  async function handleTelegramSubmit(evt: Event) {
    evt.preventDefault();
    setLoading(true);
    setError(null);

    if (telegramParamsRef.current === null) {
      try {
        telegramParamsRef.current = await getTelegramSigninParams();
      } catch (e) {
        setError(e);
        return;
      }
    }

    try {
      const user = await verifyTelegramSignin(telegramParamsRef.current.token);

      dispatch(signin(user));
    } catch (e) {
      setError(e);
    }

    setLoading(false);
  }

  function handleShowEmailStep(evt: Event) {
    evt.preventDefault();
    setFormView('email');
    setView('email');
    setError(null);
  }

  const hasOAuthProviders = oauthProviders.length > 0;
  const hasFormProviders = formProviders.length > 0;

  const formFooterJSX = (
    <>
      {errorMessage && <div className={clsx('auth-error', styles.error)}>{errorMessage}</div>}
      <Button className="auth-submit" type="submit" disabled={isLoading}>
        {isLoading ? <Spinner /> : intl.formatMessage(messages.submit)}
      </Button>
    </>
  );

  return (
    <div className={clsx('auth', styles.root)}>
      <Button className="auth-button" selected={isDropdownShown} onClick={handleClickSingIn} suffix={<ArrowIcon />}>
        {intl.formatMessage(messages.signin)}
      </Button>
      {isDropdownShown && (
        <div className={clsx('auth-dropdown', styles.dropdown)} ref={ref}>
          {!hasOAuthProviders && !hasFormProviders && (
            <div className={clsx('auth-error', styles.error)}>{intl.formatMessage(messages.noProviders)}</div>
          )}
          <form className={clsx('auth-form', styles.form)} onSubmit={handleSubmit}>
            {view === 'telegram' && telegramParamsRef.current !== null ? (
              <>
                <div className={clsx('auth-row', styles.row)}>
                  <div className={styles.backButton}>
                    <Button className="auth-back-button" size="xs" kind="transparent" onClick={handleClickBack}>
                      <svg
                        className={styles.backButtonArrow}
                        width="14"
                        height="14"
                        viewBox="0 0 14 14"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M8.75 3L5 7.25L9 11"
                          stroke="currentColor"
                          stroke-width="2"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                        />
                      </svg>
                      {intl.formatMessage(messages.back)}
                    </Button>
                  </div>
                  <button
                    className={clsx('auth-close-button', styles.closeButton)}
                    title="Close sign-in dropdown"
                    onClick={handleDropdownClose}
                  >
                    <CrossIcon />
                  </button>
                </div>
                <TelegramLink
                  onSubmit={handleTelegramSubmit}
                  errorMessage={errorMessage}
                  bot={telegramParamsRef.current.bot}
                  token={telegramParamsRef.current.token}
                />
              </>
            ) : view === 'token' ? (
              <>
                <div className={clsx('auth-row', styles.row)}>
                  <div className={styles.backButton}>
                    <Button className="auth-back-button" size="xs" kind="transparent" onClick={handleShowEmailStep}>
                      <svg
                        className={styles.backButtonArrow}
                        width="14"
                        height="14"
                        viewBox="0 0 14 14"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M8.75 3L5 7.25L9 11"
                          stroke="currentColor"
                          stroke-width="2"
                          stroke-linecap="round"
                          stroke-linejoin="round"
                        />
                      </svg>
                      {intl.formatMessage(messages.back)}
                    </Button>
                  </div>
                  <button
                    className={clsx('auth-close-button', styles.closeButton)}
                    title="Close sign-in dropdown"
                    onClick={handleDropdownClose}
                  >
                    <CrossIcon />
                  </button>
                </div>
                <div className={clsx('auth-row', styles.row)}>
                  <TextareaAutosize
                    name="token"
                    className={clsx('auth-token-textarea', inputStyles.input, styles.textarea)}
                    placeholder={intl.formatMessage(messages.token)}
                    disabled={isLoading}
                    onInput={handleTokenInput}
                  />
                </div>
                {formFooterJSX}
              </>
            ) : (
              <>
                {hasOAuthProviders && (
                  <>
                    <h5 className={clsx('auth-form-title', styles.title)}>
                      {intl.formatMessage(messages.oauthSource)}
                    </h5>
                    <OAuth providers={oauthProviders} onOauthClick={handleOauthClick} />
                  </>
                )}
                {hasOAuthProviders && hasFormProviders && (
                  <div className={clsx('auth-divider', styles.divider)} title={intl.formatMessage(messages.or)} />
                )}
                {hasFormProviders && (
                  <>
                    {formProviders.length === 1 ? (
                      <h5 className={clsx('auth-form-title', styles.title)}>{formProviders[0]}</h5>
                    ) : (
                      <div className={clsx('auth-tabs', styles.tabs)}>
                        {formProviders.map((p) => (
                          <Fragment key={p}>
                            <input
                              className={styles.radio}
                              type="radio"
                              id={`form-provider-${p}`}
                              name="form-provider"
                              value={p}
                              onChange={handleProviderChange}
                              checked={p === view}
                            />
                            <label className={clsx('auth-tabs-item', styles.provider)} htmlFor={`form-provider-${p}`}>
                              {p.slice(0, 6)}
                            </label>
                          </Fragment>
                        ))}
                      </div>
                    )}
                    <div className={clsx('auth-row', styles.row)}>
                      <Input
                        className="auth-input-username"
                        required
                        name="username"
                        // 3 to 64 characters once trimmed, counted in code points as the server counts
                        // them; minLength would count UTF-16 code units and miss a space at either end.
                        // \s is the whitespace trim() removes on blur and on submit. Email sign-in takes
                        // no dot, as the name goes into a mail to an address nobody has confirmed yet
                        pattern={view === 'email' ? EMAIL_USERNAME_PATTERN : USERNAME_PATTERN}
                        title={intl.formatMessage(
                          view === 'email' ? messages.emailUsernameRestriction : messages.usernameRestriction
                        )}
                        placeholder={intl.formatMessage(messages.username)}
                        disabled={isLoading}
                        value={username}
                        onInput={(evt) => setUsername(evt.currentTarget.value)}
                        onBlur={(evt) => {
                          const element = evt.currentTarget;
                          element.value = element.value.trim();
                          setUsername(element.value);
                        }}
                        dir="auto"
                      />
                    </div>
                    {view === 'email' && (
                      <div className={clsx('auth-row', styles.row)}>
                        <Input
                          className="auth-input-email"
                          required
                          name="email"
                          type="email"
                          placeholder={intl.formatMessage(messages.emailAddress)}
                          disabled={isLoading}
                          value={email}
                          onInput={(evt) => setEmail(evt.currentTarget.value)}
                          dir="auto"
                        />
                      </div>
                    )}
                    {formFooterJSX}
                  </>
                )}
              </>
            )}
          </form>
        </div>
      )}
    </div>
  );
}
