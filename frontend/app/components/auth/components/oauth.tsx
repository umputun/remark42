import { h, JSX } from 'preact';
import { useDispatch } from 'react-redux';
import clsx from 'clsx';
import { useIntl } from 'react-intl';

import type { OAuthProvider } from 'common/types';
import { siteId } from 'common/settings';
import { useTheme } from 'hooks/useTheme';
import { setUser } from 'store/user/actions';

import { messages } from 'components/auth/auth.messsages';

import { oauthSignin } from './oauth.api';
import { BASE_URL } from 'common/constants.config';
import { getButtonVariant, getProviderData } from './oauth.utils';
import styles from './oauth.module.css';

const location = encodeURIComponent(`${window.location.origin}${window.location.pathname}?selfClose`);

type Props = {
  providers: OAuthProvider[];
};

export function OAuth({ providers }: Props) {
  const intl = useIntl();
  const dispatch = useDispatch();
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
              onClick={handleOauthClick}
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
