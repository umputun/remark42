import clsx from 'clsx';
import { h, Fragment, FunctionComponent } from 'preact';
import { useIntl } from 'react-intl';
import { messages } from './telegram.messages';
import { BASE_URL, API_BASE } from 'common/constants.config';
import { Button } from 'components/button';

import styles from './telegram-link.module.css';

export type TelegramLinkProps = {
  bot: string;
  token: string;
  onSubmit: (evt: Event) => void;
  errorMessage?: string | null;
};

export const TelegramLink: FunctionComponent<TelegramLinkProps> = ({ bot, token, errorMessage, onSubmit }) => {
  const intl = useIntl();
  const telegramLink = `https://t.me/${bot}/?start=${token}`;
  return (
    <>
      <p className={clsx('telegram', styles.telegram)}>
        {intl.formatMessage(messages.telegramMessage1)}{' '}
        <a target="_blank" rel="noopener noreferrer" href={telegramLink}>
          {intl.formatMessage(messages.telegramLink)}
        </a>
        {window.screen.width >= 768 && ` ${intl.formatMessage(messages.telegramOptionalQR)}`}{' '}
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
