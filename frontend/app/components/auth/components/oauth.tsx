import { h, FunctionComponent, JSX } from 'preact';
import { useDispatch } from 'react-redux';
import cn from 'classnames';
import { useIntl } from 'react-intl';

import { siteId } from 'common/settings';
import type { OAuthProvider } from 'common/types';
import useTheme from 'hooks/useTheme';
import { setUser } from 'store/user/actions';

import messages from 'components/auth/auth.messsages';

import { getButtonVariant, getProviderData } from './oauth.utils';
import { oauthSignin } from './oauth.api';
import styles from './oauth.module.css';

export type OAuthProvidersProps = {
  providers: OAuthProvider[];
};

const location = encodeURIComponent(`${window.location.origin}${window.location.pathname}?selfClose`);

const OAuthProviders: FunctionComponent<OAuthProvidersProps> = ({ providers }) => {
  const intl = useIntl();
  const dispath = useDispatch();
  const theme = useTheme();
  const buttonVariant = getButtonVariant(providers.length);
  const handleOathClick: JSX.GenericEventHandler<HTMLAnchorElement> = async (evt) => {
    const { href } = evt.currentTarget as HTMLAnchorElement;

    evt.preventDefault();
    const user = await oauthSignin(href);

    if (user === null) {
      return;
    }

    dispath(setUser(user));
  };

  return (
    <ul className={cn('oauth', styles.root)}>
      {providers.map((p) => {
        const { name, icon } = getProviderData(p, theme);

        return (
          <li className={cn('oauth-item', styles.item)}>
            <a
              target="_blank"
              rel="noopener noreferrer"
              href={`/auth/${p}/login?from=${location}&site=${siteId}`}
              onClick={handleOathClick}
              className={cn('oauth-button', styles.button, styles[buttonVariant], styles[p])}
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
};

export default OAuthProviders;
