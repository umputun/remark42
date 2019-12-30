/** @jsx createElement */
import { createElement, Component, createRef } from 'preact';
import b from 'bem-react-helper';

import { PROVIDER_NAMES, IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from '@app/common/constants';
import { requestDeletion } from '@app/utils/email';
import { getHandleClickProps } from '@app/common/accessibility';
import { User, AuthProvider, Sorting, Theme, PostInfo } from '@app/common/types';

import debounce from '@app/utils/debounce';
import postMessage from '@app/utils/postMessage';
import { StoreState } from '@app/store';
import { ProviderState } from '@app/store/provider/reducers';
import Dropdown, { DropdownItem } from '@app/components/dropdown';
import { UIButton } from '@app/components/ui-button';

import { AnonymousLoginForm } from './__anonymous-login-form';
import { EmailLoginFormConnected } from './__email-login-form';
import { EmailLoginFormRef } from './__email-login-form/auth-panel__email-login-form';

export interface Props {
  user: User | null;
  hiddenUsers: StoreState['hiddenUsers'];
  sort: Sorting;
  isCommentsDisabled: boolean;
  theme: Theme;
  postInfo: PostInfo;
  providers: (AuthProvider['name'])[];
  provider: ProviderState;

  onSortChange(s: Sorting): Promise<void>;
  onSignIn(p: AuthProvider): Promise<User | null>;
  onSignOut(): Promise<void>;
  onCommentsEnable(): Promise<boolean>;
  onCommentsDisable(): Promise<boolean>;
  onBlockedUsersShow(): void;
  onBlockedUsersHide(): void;
}

interface State {
  isBlockedVisible: boolean;
  anonymousUsernameInputValue: string;
  threshold: number;
  sortSelectFocused: boolean;
}

export class AuthPanel extends Component<Props, State> {
  emailLoginRef = createRef<EmailLoginFormRef>();

  constructor(props: Props) {
    super(props);

    this.state = {
      isBlockedVisible: false,
      anonymousUsernameInputValue: 'anon',
      threshold: 3,
      sortSelectFocused: false,
    };

    this.toggleBlockedVisibility = this.toggleBlockedVisibility.bind(this);
    this.toggleCommentsAvailability = this.toggleCommentsAvailability.bind(this);
    this.onSortChange = this.onSortChange.bind(this);
    this.onSignIn = this.onSignIn.bind(this);
    this.onEmailSignIn = this.onEmailSignIn.bind(this);
    this.handleAnonymousLoginFormSubmut = this.handleAnonymousLoginFormSubmut.bind(this);
    this.handleOAuthLogin = this.handleOAuthLogin.bind(this);
    this.toggleUserInfoVisibility = this.toggleUserInfoVisibility.bind(this);
    this.onEmailTitleClick = this.onEmailTitleClick.bind(this);
  }

  componentWillMount() {
    this.resizeHandler();
    window.addEventListener('resize', this.resizeHandler);
  }

  componentWillUnmount() {
    window.removeEventListener('resize', this.resizeHandler);
  }

  singInMessageAndSortWidth = 255;

  resizeHandler = debounce(() => {
    this.setState({
      threshold: Math.max(3, Math.round((window.innerWidth - this.singInMessageAndSortWidth) / 80)),
    });
  }, 100);

  onEmailTitleClick() {
    this.emailLoginRef.current && this.emailLoginRef.current.focus();
  }

  onSortChange(e: Event) {
    if (this.props.onSortChange) {
      this.props.onSortChange((e.target! as HTMLOptionElement).value as Sorting);
    }
  }

  onSortFocus = () => {
    this.setState({ sortSelectFocused: true });
  };

  onSortBlur = (e: Event) => {
    this.setState({ sortSelectFocused: false });

    this.onSortChange(e);
  };

  toggleBlockedVisibility() {
    if (!this.state.isBlockedVisible) {
      if (this.props.onBlockedUsersShow) this.props.onBlockedUsersShow();
    } else if (this.props.onBlockedUsersHide) this.props.onBlockedUsersHide();

    this.setState({ isBlockedVisible: !this.state.isBlockedVisible });
  }

  toggleCommentsAvailability() {
    if (this.props.isCommentsDisabled) {
      this.props.onCommentsEnable && this.props.onCommentsEnable();
    } else {
      this.props.onCommentsDisable && this.props.onCommentsDisable();
    }
  }

  toggleUserInfoVisibility() {
    const user = this.props.user;
    if (window.parent && user) {
      const data = { isUserInfoShown: true, user };
      postMessage(data);
    }
  }

  /** wrapper function to handle both oauth and anonymous providers*/
  onSignIn(provider: AuthProvider) {
    this.props.onSignIn(provider);
  }

  onEmailSignIn(token: string) {
    return this.props.onSignIn({ name: 'email', token });
  }

  async handleAnonymousLoginFormSubmut(username: string) {
    this.onSignIn({ name: 'anonymous', username });
  }

  async handleOAuthLogin(e: MouseEvent | KeyboardEvent) {
    const p = (e.target as HTMLButtonElement).dataset.provider! as AuthProvider['name'];
    this.onSignIn({ name: p } as AuthProvider);
  }

  renderAuthorized = () => {
    const { user, onSignOut, theme } = this.props;
    if (!user) return null;

    const isUserAnonymous = user && user.id.substr(0, 10) === 'anonymous_';

    return (
      <div className="auth-panel__column">
        You signed in as{' '}
        <Dropdown title={user.name} titleClass="auth-panel__user-dropdown-title" theme={theme}>
          <DropdownItem separator={!isUserAnonymous}>
            <div
              id={user.id}
              className={b('auth-panel__user-id', {}, { theme })}
              {...getHandleClickProps(this.toggleUserInfoVisibility)}
            >
              {user.id}
            </div>
          </DropdownItem>

          {!isUserAnonymous && (
            <DropdownItem>
              <UIButton theme={theme} onClick={() => requestDeletion().then(onSignOut)}>
                Request my data removal
              </UIButton>
            </DropdownItem>
          )}
        </Dropdown>{' '}
        <UIButton kind="link" theme={theme} onClick={onSignOut}>
          Sign out?
        </UIButton>
      </div>
    );
  };

  renderProvider = (provider: AuthProvider['name'], dropdown = false) => {
    if (provider === 'anonymous') {
      return (
        <Dropdown
          title={PROVIDER_NAMES['anonymous']}
          titleClass={dropdown ? 'auth-panel__dropdown-provider' : ''}
          theme={this.props.theme}
        >
          <DropdownItem>
            <AnonymousLoginForm
              onSubmit={this.handleAnonymousLoginFormSubmut}
              theme={this.props.theme}
              className="auth-panel__anonymous-login-form"
            />
          </DropdownItem>
        </Dropdown>
      );
    }
    if (provider === 'email') {
      return (
        <Dropdown
          title={PROVIDER_NAMES['email']}
          titleClass={dropdown ? 'auth-panel__dropdown-provider' : ''}
          theme={this.props.theme}
          onTitleClick={this.onEmailTitleClick}
        >
          <DropdownItem>
            <EmailLoginFormConnected
              ref={this.emailLoginRef}
              onSignIn={this.onEmailSignIn}
              theme={this.props.theme}
              className="auth-panel__email-login-form"
            />
          </DropdownItem>
        </Dropdown>
      );
    }

    return (
      <UIButton
        mix={dropdown ? 'auth-panel__dropdown-provider' : ''}
        kind="link"
        data-provider={provider}
        {...getHandleClickProps(this.handleOAuthLogin)}
        role="link"
      >
        {PROVIDER_NAMES[provider]}
      </UIButton>
    );
  };

  renderOther = (providers: (AuthProvider['name'])[]) => {
    return (
      <Dropdown title="Other" theme={this.props.theme} onTitleClick={this.onEmailTitleClick}>
        {providers.map(provider => (
          <DropdownItem>{this.renderProvider(provider, true)}</DropdownItem>
        ))}
      </Dropdown>
    );
  };

  renderUnauthorized = () => {
    const { user, providers = [], postInfo } = this.props;
    const { threshold } = this.state;
    if (user || !IS_STORAGE_AVAILABLE) return null;

    const signInMessage = postInfo.read_only ? 'Sign in using ' : 'Sign in to comment using ';
    const sortedProviders = ((): typeof providers => {
      if (!this.props.provider.name) return providers;
      const lastProviderIndex = providers.indexOf(this.props.provider.name as typeof providers[0]);
      if (lastProviderIndex < 1) return providers;
      return [
        this.props.provider.name as typeof providers[0],
        ...providers.slice(0, lastProviderIndex),
        ...providers.slice(lastProviderIndex + 1),
      ];
    })();

    const isAboveThreshold = sortedProviders.length > threshold;

    return (
      <div className="auth-panel__column">
        {signInMessage}
        {!isAboveThreshold &&
          sortedProviders.map((provider, i) => {
            const comma = i === 0 ? '' : i === sortedProviders.length - 1 ? ' or ' : ', ';

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
            {' or '}
            {this.renderOther(sortedProviders.slice(threshold - 1))}
          </span>
        )}
      </div>
    );
  };

  renderThirdPartyWarning = () => {
    if (IS_STORAGE_AVAILABLE || !IS_THIRD_PARTY) return null;
    return (
      <div className="auth-panel__column">
        Disable third-party cookies blocking to sign in or open comments in{' '}
        <a
          className="auth-panel__pseudo-link"
          href={`${window.location.origin}/web/comments.html${window.location.search}`}
          target="_blank"
        >
          new page
        </a>
      </div>
    );
  };

  renderCookiesWarning = () => {
    if (IS_STORAGE_AVAILABLE || IS_THIRD_PARTY) return null;
    return <div className="auth-panel__column">Allow cookies to sign in and comment</div>;
  };

  renderSettingsLabel = () => {
    return (
      <UIButton
        kind="link"
        mix="auth-panel__admin-action"
        {...getHandleClickProps(() => this.toggleBlockedVisibility())}
        role="link"
      >
        {this.state.isBlockedVisible ? 'Hide' : 'Show'} settings
      </UIButton>
    );
  };

  renderReadOnlySwitch = () => {
    const { isCommentsDisabled } = this.props;
    return (
      <UIButton
        kind="link"
        mix="auth-panel__admin-action"
        {...getHandleClickProps(() => this.toggleCommentsAvailability())}
        role="link"
      >
        {isCommentsDisabled ? 'Enable' : 'Disable'} comments
      </UIButton>
    );
  };

  renderSort = () => {
    const { sort } = this.props;
    const { sortSelectFocused } = this.state;
    const sortArray = getSortArray(sort);
    return (
      <span className="auth-panel__sort">
        Sort by{' '}
        <span className="auth-panel__select-label">
          <span className={b('auth-panel__select-label-value', {}, { focused: sortSelectFocused })}>
            {sortArray.find(x => 'selected' in x && x.selected!)!.label}
          </span>
          <select
            className="auth-panel__select"
            onChange={this.onSortChange}
            onFocus={this.onSortFocus}
            onBlur={this.onSortBlur}
          >
            {sortArray.map(sort => (
              <option value={sort.value} selected={sort.selected}>
                {sort.label}
              </option>
            ))}
          </select>
        </span>
      </span>
    );
  };

  render(props: Props, { isBlockedVisible }: State) {
    const {
      user,
      postInfo: { read_only },
      theme,
    } = props;
    const isAdmin = user && user.admin;
    const isSettingsLabelVisible = Object.keys(this.props.hiddenUsers).length > 0 || isAdmin || isBlockedVisible;

    return (
      <div className={b('auth-panel', {}, { theme, loggedIn: !!user })}>
        {this.renderAuthorized()}
        {this.renderUnauthorized()}
        {this.renderThirdPartyWarning()}
        {this.renderCookiesWarning()}
        <div className="auth-panel__column">
          {isSettingsLabelVisible && this.renderSettingsLabel()}

          {isSettingsLabelVisible && ' • '}

          {isAdmin && this.renderReadOnlySwitch()}

          {isAdmin && ' • '}

          {!isAdmin && read_only && <span className="auth-panel__readonly-label">Read-only</span>}

          {this.renderSort()}
        </div>
      </div>
    );
  }
}

function getSortArray(currentSort: Sorting) {
  const sortArray: {
    value: Sorting;
    label: string;
    selected?: boolean;
  }[] = [
    {
      value: '-score',
      label: 'Best',
    },
    {
      value: '+score',
      label: 'Worst',
    },
    {
      value: '-time',
      label: 'Newest',
    },
    {
      value: '+time',
      label: 'Oldest',
    },
    {
      value: '-active',
      label: 'Recently updated',
    },
    {
      value: '+active',
      label: 'Least recently updated',
    },
    {
      value: '-controversy',
      label: 'Most controversial',
    },
    {
      value: '+controversy',
      label: 'Least controversial',
    },
  ];

  return sortArray.map(sort => {
    if (sort.value === currentSort) {
      sort.selected = true;
    }

    return sort;
  });
}
