import clsx from 'clsx';
import { h, Fragment, JSX } from 'preact';
import { useState, useRef } from 'preact/hooks';
import { useIntl } from 'react-intl';
import { useDispatch } from 'react-redux';

import { BASE_URL, API_BASE } from 'common/constants.config';
import { setUser } from 'store/user/actions';
import { Input } from 'components/input';
import { CrossIcon } from 'components/icons/cross';
import { TextareaAutosize } from 'components/textarea-autosize';
import { Spinner } from 'components/spinner/spinner';
import { ArrowIcon } from 'components/icons/arrow';

import { Button } from './components/button';
import { OAuth } from './components/oauth';
import { messages } from './auth.messsages';
import { useDropdown, useErrorMessage } from './auth.hooks';
import { getProviders, getTokenInvalidReason } from './auth.utils';
import {
  oauthSignin,
  emailSignin,
  verifyEmailSignin,
  anonymousSignin,
  verifyTelegramSignin,
  getTelegramSigninParams,
} from './auth.api';

import styles from './auth.module.css';

export function Auth() {
  const intl = useIntl();
  const telegramParamsRef = useRef<null | { bot: string; token: string }>(null);
  const dispatch = useDispatch();
  const [oauthProviders, formProviders] = getProviders();

  // UI State
  const [isLoading, setLoading] = useState(false);
  const [view, setView] = useState<typeof formProviders[number] | 'token' | 'telegram'>(formProviders[0]);
  const [ref, isDropdownShown, toggleDropdownState] = useDropdown(view === 'token' || view === 'telegram');

  // Errors
  const [errorMessage, setError] = useErrorMessage();

  function handleClickSingIn(evt: Event) {
    evt.preventDefault();
    toggleDropdownState();
  }

  function resetView() {
    setView(formProviders[0]);
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
      dispatch(setUser(user));
    } catch (e) {
      setError(e);
    }
  }

  function handleProviderChange(evt: Event) {
    const { value } = evt.currentTarget as HTMLInputElement;

    setView(value as typeof formProviders[number]);
    setError(null);
  }

  async function handleSubmit(evt: Event) {
    const data = new FormData(evt.target as HTMLFormElement);

    evt.preventDefault();
    setLoading(true);
    setError(null);

    try {
      switch (view) {
        case 'anonymous': {
          const username = data.get('username') as string;
          const user = await anonymousSignin(username);

          dispatch(setUser(user));
          break;
        }
        case 'email': {
          const email = data.get('email') as string;
          const username = data.get('username') as string;

          await emailSignin(email, username);
          setView('token');
          break;
        }
        case 'token': {
          const token = data.get('token') as string;
          const invalidReason = getTokenInvalidReason(token);

          if (invalidReason) {
            setError(invalidReason);
          } else {
            const user = await verifyEmailSignin(token);
            dispatch(setUser(user));
          }

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

      dispatch(setUser(user));
    } catch (e) {
      setError(e);
    }

    setLoading(false);
  }

  function handleShowEmailStep(evt: Event) {
    evt.preventDefault();
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
                <p className={clsx('telegram', styles.telegram)}>
                  {intl.formatMessage(messages.telegramMessage1)}{' '}
                  <a
                    target="_blank"
                    rel="noopener noreferrer"
                    href={`https://t.me/${telegramParamsRef.current.bot}/?start=${telegramParamsRef.current.token}`}
                  >
                    {intl.formatMessage(messages.telegramLink)}
                  </a>
                  {window.screen.width >= 768 && ` ${intl.formatMessage(messages.telegramOptionalQR)}`}{' '}
                  {intl.formatMessage(messages.telegramMessage2)}
                  <br />
                  {intl.formatMessage(messages.telegramMessage3)}
                </p>
                {window.screen.width >= 768 && (
                  <img
                    src={`${BASE_URL}${API_BASE}/qr/telegram?url=https://t.me/${telegramParamsRef.current.bot}/?start=${telegramParamsRef.current.token}`}
                    height="200"
                    width="200"
                    className={clsx('telegram-qr', styles.telegramQR)}
                    alt={intl.formatMessage(messages.telegramQR)}
                  />
                )}
                {errorMessage && <div className={clsx('auth-error', styles.error)}>{errorMessage}</div>}
                <Button key="submit" className="auth-submit" type="submit" onClick={handleTelegramSubmit}>
                  {intl.formatMessage(messages.telegramCheck)}
                </Button>
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
                    className={clsx('auth-token-textarea', styles.textarea)}
                    placeholder={intl.formatMessage(messages.token)}
                    disabled={isLoading}
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
                        minLength={3}
                        pattern="[\p{L}\d\s_]+"
                        title={intl.formatMessage(messages.usernameRestriction)}
                        placeholder={intl.formatMessage(messages.username)}
                        disabled={isLoading}
                        onBlur={(evt) => {
                          const element = evt.target as HTMLInputElement;
                          element.value = element.value.trim();
                        }}
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
                        />
                      </div>
                    )}
                    <input className={styles.honeypot} type="checkbox" tabIndex={-1} autoComplete="off" />
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
