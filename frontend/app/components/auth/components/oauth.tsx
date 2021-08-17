import { h, JSX } from 'preact';
import { useDispatch, useSelector } from 'react-redux';
import clsx from 'clsx';
import { useIntl } from 'react-intl';

import type { OAuthProvider } from 'common/types';
import { siteId } from 'common/settings';
import { useTheme } from 'hooks/useTheme';
import { setUser, setTelegramParams } from 'store/user/actions';
import { StoreState } from 'store';

import { messages } from 'components/auth/auth.messsages';

import { oauthSignin } from './oauth.api';
import { BASE_URL } from 'common/constants.config';
import { getButtonVariant, getProviderData } from './oauth.utils';
import styles from './oauth.module.css';
import { getTelegramSigninParams } from '../auth.api';

const location = encodeURIComponent(`${window.location.origin}${window.location.pathname}?selfClose`);

type Props = {
  providers: OAuthProvider[];
  toggleTelegram?: (showTelegram: boolean) => void;
};

export function OAuth({ providers, toggleTelegram }: Props) {
  const intl = useIntl();
  const dispatch = useDispatch();
  const telegramParams = useSelector((s: StoreState) => s.telegramParams);
  const theme = useTheme();
  const buttonVariant = getButtonVariant(providers.length);
  const handleOauthClick: JSX.GenericEventHandler<HTMLAnchorElement> = async (evt) => {
    const { href } = evt.currentTarget as HTMLAnchorElement;

    evt.preventDefault();
    const user = await oauthSignin(href);

    if (user === null) {
      return;
    }

    dispatch(setUser(user));
  };

  const handleTelegramClick: JSX.EventHandler<JSX.TargetedMouseEvent<HTMLButtonElement>> = async (evt) => {
    evt.preventDefault();
    if (!telegramParams) {
      const params = await getTelegramSigninParams();
      if (params === null) {
        return;
      }
      dispath(setTelegramParams(params));
    }
    toggleTelegram && toggleTelegram(true);
  };

  return (
    <ul className={clsx('oauth', styles.root)}>
      {providers.map((p) => {
        const { name, icon } = getProviderData(p, theme);

        return (
          <li key={name} className={clsx('oauth-item', styles.item)}>
            {name === 'Telegram' ? (
              <button
                onClick={handleTelegramClick}
                className={clsx('oauth-button telegram-auth', styles.button, styles[buttonVariant], styles[p])}
                data-provider-name={name}
                title={intl.formatMessage(messages.oauthTitle, { provider: name })}
              >
                <img className="oauth-icon telegram-auth" src={icon} width="20" height="20" alt="" aria-hidden={true} />
              </button>
            ) : (
              <a
                target="_blank"
                rel="noopener noreferrer"
                href={`${BASE_URL}/auth/${p}/login?from=${location}&site=${siteId}`}
                onClick={handleOauthClick}
                className={clsx('oauth-button', styles.button, styles[buttonVariant], styles[p])}
                data-provider-name={name}
                title={intl.formatMessage(messages.oauthTitle, { provider: name })}
              >
                <img className="oauth-icon" src={icon} width="20" height="20" alt="" aria-hidden={true} />
              </a>
            )}
          </li>
        );
      })}
    </ul>
  );
}
