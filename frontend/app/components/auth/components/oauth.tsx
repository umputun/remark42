import { h, JSX } from 'preact';
import clsx from 'clsx';
import { useIntl } from 'react-intl';

import type { OAuthProvider } from 'common/types';
import { siteId } from 'common/settings';
import { useTheme } from 'hooks/useTheme';

import { messages } from 'components/auth/auth.messsages';

import { BASE_URL } from 'common/constants.config';
import { getButtonVariant, getProviderData } from './oauth.utils';
import styles from './oauth.module.css';

const location = encodeURIComponent(`${window.location.origin}${window.location.pathname}?selfClose`);

type Props = {
  providers: OAuthProvider[];
  onOauthClick?(evt: JSX.TargetedMouseEvent<HTMLAnchorElement>): void;
};

export function OAuth({ providers, onOauthClick }: Props) {
  const intl = useIntl();
  const theme = useTheme();
  const buttonVariant = getButtonVariant(providers.length);

  return (
    <ul className={clsx('oauth', styles.root)}>
      {providers.map((p) => {
        const { name, icon } = getProviderData(p, theme);

        return (
          <li key={name} className={clsx('oauth-item', styles.item)}>
            <a
              target="_blank"
              rel="noopener noreferrer"
              href={`${BASE_URL}/auth/${p}/login?from=${location}&site=${siteId}`}
              onClick={onOauthClick}
              className={clsx('oauth-button', styles.button, styles[buttonVariant], styles[p])}
              data-provider-name={name}
              title={intl.formatMessage(messages.oauthTitle, { provider: name })}
            >
              <img className="oauth-icon" src={icon} width="20" height="20" alt="" aria-hidden={true} />
            </a>
          </li>
        );
      })}
    </ul>
  );
}
