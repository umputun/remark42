import clsx from 'clsx';
import type { FunctionComponent } from 'preact';
import { h, Fragment } from 'preact';
import { messages } from './telegram.messages';
import { BASE_URL, API_BASE } from 'common/constants.config';
import { Button } from 'components/button';

import { useIntl } from 'common/intl';
import styles from './telegram-link.module.css';

export type TelegramLinkProps = {
  bot: string;
  token: string;
  onSubmit: (evt: Event) => void;
  errorMessage?: string | null;
};

const SPACELESS_LOCALES = new Set(['ja', 'zh', 'zh-tw']);

export const TelegramLink: FunctionComponent<TelegramLinkProps> = ({ bot, token, errorMessage, onSubmit }) => {
  const intl = useIntl();
  const telegramLink = `https://t.me/${bot}/?start=${token}`;
  // this paragraph is assembled from separate messages, and Japanese and Chinese do not put
  // spaces between words, so joining their fragments with one produces spaces mid-sentence.
  // matched exactly as `loadLocale` matches, so the separator can never disagree with the
  // catalogue that actually loaded: an unrecognised casing falls back to English and keeps spaces
  const gap = SPACELESS_LOCALES.has(intl.locale) ? '' : ' ';
  return (
    <>
      <p className={clsx('telegram', styles.telegram)}>
        {intl.formatMessage(messages.telegramMessage1)}
        {gap}
        <a target="_blank" rel="noopener noreferrer" href={telegramLink}>
          {intl.formatMessage(messages.telegramLink)}
        </a>
        {window.screen.width >= 768 && `${gap}${intl.formatMessage(messages.telegramOptionalQR)}`}
        {gap}
        {intl.formatMessage(messages.telegramMessage2)}
        <br />
        {intl.formatMessage(messages.telegramMessage3)}
      </p>
      {window.screen.width >= 768 && (
        <img
          src={`${BASE_URL}${API_BASE}/qr/telegram?url=${telegramLink}`}
          height="200"
          width="200"
          className={clsx('telegram-qr', styles.telegramQR)}
          alt={intl.formatMessage(messages.telegramQR)}
        />
      )}
      {errorMessage && <div className={clsx('auth-error', styles.error)}>{errorMessage}</div>}
      <Button
        key="submit"
        kind="primary"
        size="large"
        className={clsx('auth-submit', styles.button)}
        type="submit"
        onClick={onSubmit}
      >
        {intl.formatMessage(messages.telegramCheck)}
      </Button>
    </>
  );
};
