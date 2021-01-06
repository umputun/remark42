import { h, Component, createRef } from 'preact';
import { useCallback } from 'preact/hooks';
import { IntlShape, FormattedMessage, defineMessages, useIntl } from 'react-intl';

import { AuthProvider, Theme, User } from 'common/types';
import { PROVIDER_NAMES, IS_STORAGE_AVAILABLE } from 'common/constants';
import { getHandleClickProps } from 'common/accessibility';
import { Button } from 'components/button';
import { Dropdown, DropdownItem } from 'components/dropdown';

import debounce from 'utils/debounce';
import { ProviderState } from 'store/provider/reducers';
import { StaticStore } from 'common/static-store';
import { useSelector, useDispatch } from 'react-redux';
import { StoreState } from 'store';
import useTheme from 'hooks/useTheme';
import { logIn } from 'store/user/actions';

import { AnonymousLoginForm } from './__anonymous-login-form';
import { EmailLoginFormConnected, EmailLoginFormRef } from './__email-login-form';

import styles from './auth.module.css';

interface Props {
  intl: IntlShape;
  theme: Theme;
  onSignIn(provider: AuthProvider): any; // eslint-disable-line
  user: User | null;
  provider: ProviderState;
}

interface State {
  threshold: number;
}

class Auth extends Component<Props, State> {
  emailLoginRef = createRef<EmailLoginFormRef>();
  singInMessageAndSortWidth = 255;

  state = {
    threshold: 3,
  };

  componentWillMount() {
    this.resizeHandler();
    window.addEventListener('resize', this.resizeHandler);
  }

  componentWillUnmount() {
    window.removeEventListener('resize', this.resizeHandler);
  }
  resizeHandler = debounce(() => {
    this.setState({
      threshold: Math.max(3, Math.round((window.innerWidth - this.singInMessageAndSortWidth) / 80)),
    });
  }, 100);

  onEmailTitleClick = () => {
    this.emailLoginRef.current && this.emailLoginRef.current.focus();
  };

  onEmailSignIn = (token: string) => {
    return this.props.onSignIn({ name: 'email', token });
  };

  handleOAuthLogin = async (e: MouseEvent | KeyboardEvent) => {
    const name = (e.target as HTMLButtonElement).dataset.provider! as AuthProvider['name'];

    this.props.onSignIn({ name } as AuthProvider);
  };

  handleAnonymousLoginFormSubmut = async (username: string) => {
    this.props.onSignIn({ name: 'anonymous', username });
  };

  renderOther = (providers: AuthProvider['name'][]) => {
    const other = this.props.intl.formatMessage(authPanelMessages.otherProvider);

    return (
      <Dropdown title={other} theme={this.props.theme} onTitleClick={this.onEmailTitleClick}>
        {providers.map(provider => (
          <DropdownItem>{this.renderProvider(provider)}</DropdownItem>
        ))}
      </Dropdown>
    );
  };

  renderProvider = (provider: AuthProvider['name']) => {
    if (provider === 'anonymous') {
      const anonymous = this.props.intl.formatMessage(authPanelMessages.anonymousProvider);
      return (
        <Dropdown title={anonymous} theme={this.props.theme}>
          <DropdownItem>
            <AnonymousLoginForm
              onSubmit={this.handleAnonymousLoginFormSubmut}
              theme={this.props.theme}
              intl={this.props.intl}
            />
          </DropdownItem>
        </Dropdown>
      );
    }
    if (provider === 'email') {
      return (
        <Dropdown title={PROVIDER_NAMES['email']} theme={this.props.theme} onTitleClick={this.onEmailTitleClick}>
          <DropdownItem>
            <EmailLoginFormConnected ref={this.emailLoginRef} onSignIn={this.onEmailSignIn} theme={this.props.theme} />
          </DropdownItem>
        </Dropdown>
      );
    }

    return (
      <Button kind="link" data-provider={provider} {...getHandleClickProps(this.handleOAuthLogin)} role="link">
        {PROVIDER_NAMES[provider]}
      </Button>
    );
  };

  render({ intl }: Props, { threshold }: State) {
    if (!IS_STORAGE_AVAILABLE) return null;

    const sortedProviders = ((providers): typeof providers => {
      if (!this.props.provider.name) return providers;
      const lastProviderIndex = providers.indexOf(this.props.provider.name as typeof providers[0]);
      if (lastProviderIndex < 1) return providers;
      return [
        this.props.provider.name as typeof providers[0],
        ...providers.slice(0, lastProviderIndex),
        ...providers.slice(lastProviderIndex + 1),
      ];
    })(StaticStore.config.auth_providers);

    const isAboveThreshold = sortedProviders.length > threshold;
    const or = intl.formatMessage(authPanelMessages.orProvider);

    return (
      <div className={styles.auth}>
        <FormattedMessage id="authPanel.login" defaultMessage="Login:" />{' '}
        {!isAboveThreshold &&
          sortedProviders.map((provider, i) => {
            const comma = i === 0 ? '' : i === sortedProviders.length - 1 ? ` ${or} ` : ', ';

            return (
              <span>
                {comma}
                {this.renderProvider(provider)}
              </span>
            );
          })}
        {isAboveThreshold &&
          sortedProviders.slice(0, threshold - 1).map((provider, i) => {
            const comma = i === 0 ? '' : ', ';

            return (
              <span>
                {comma}
                {this.renderProvider(provider)}
              </span>
            );
          })}
        {isAboveThreshold && (
          <span>
            {` ${or} `}
            {this.renderOther(sortedProviders.slice(threshold - 1))}
          </span>
        )}
      </div>
    );
  }
}

const authPanelMessages = defineMessages({
  otherProvider: {
    id: 'authPanel.other-provider',
    defaultMessage: 'Other',
  },
  anonymousProvider: {
    id: 'authPanel.anonymous-provider',
    defaultMessage: 'Anonymous',
  },
  orProvider: {
    id: 'authPanel.or-provider',
    defaultMessage: 'or',
  },
});

export default function AuthWrapper() {
  const dispatch = useDispatch();
  const provider = useSelector<StoreState, ProviderState>(store => store.provider);
  const user = useSelector<StoreState, User | null>(store => store.user);
  const theme = useTheme();
  const intl = useIntl();
  const handleSignin = useCallback((provider: AuthProvider) => dispatch(logIn(provider)), [dispatch]);

  return <Auth provider={provider} theme={theme} onSignIn={handleSignin} intl={intl} user={user} />;
}
