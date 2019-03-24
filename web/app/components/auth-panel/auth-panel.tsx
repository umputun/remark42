/** @jsx h */
import { h, Component, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { PROVIDER_NAMES, IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from '@app/common/constants';
import { requestDeletion } from '@app/utils/email';
import { getHandleClickProps } from '@app/common/accessibility';
import { User, AuthProvider, Sorting, Theme, PostInfo } from '@app/common/types';

import Dropdown, { DropdownItem } from '@app/components/dropdown';
import { Button } from '@app/components/button';
import { UserID } from './__user-id';
import { AnonymousLoginForm } from './__anonymous-login-form';

export interface Props {
  user: User | null;
  providers: (AuthProvider['name'])[];
  sort: Sorting;
  isCommentsDisabled: boolean;
  theme: Theme;
  postInfo: PostInfo;

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
}

export class AuthPanel extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      isBlockedVisible: false,
      anonymousUsernameInputValue: 'anon',
    };

    this.toggleBlockedVisibility = this.toggleBlockedVisibility.bind(this);
    this.toggleCommentsAvailability = this.toggleCommentsAvailability.bind(this);
    this.onSortChange = this.onSortChange.bind(this);
    this.onSignIn = this.onSignIn.bind(this);
    this.handleAnonymousLoginFormSubmut = this.handleAnonymousLoginFormSubmut.bind(this);
    this.handleOAuthLogin = this.handleOAuthLogin.bind(this);
    this.toggleUserInfoVisibility = this.toggleUserInfoVisibility.bind(this);
  }

  onSortChange(e: Event) {
    if (this.props.onSortChange) {
      this.props.onSortChange((e.target! as HTMLOptionElement).value as Sorting);
    }
  }

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
      const data = JSON.stringify({ isUserInfoShown: true, user });
      window.parent.postMessage(data, '*');
    }
  }

  getUserTitle() {
    const { user } = this.props;
    return <span className="auth-panel__username">{user!.name}</span>;
  }

  /** wrapper function to handle both oauth and anonymous providers*/
  onSignIn(provider: AuthProvider) {
    this.props.onSignIn(provider);
  }

  async handleAnonymousLoginFormSubmut(username: string) {
    this.onSignIn({ name: 'anonymous', username });
  }

  async handleOAuthLogin(e: MouseEvent | KeyboardEvent) {
    const p = (e.target as HTMLButtonElement).dataset.provider! as AuthProvider['name'];
    // eslint-disable-next-line @typescript-eslint/no-object-literal-type-assertion
    this.onSignIn({ name: p } as AuthProvider);
  }

  render(props: RenderableProps<Props>, { isBlockedVisible }: State) {
    const { user, providers = [], sort, isCommentsDisabled } = props;
    const sortArray = getSortArray(sort);
    const loggedIn = !!user;
    const signInMessage = props.postInfo.read_only ? 'Sign in using ' : 'Sign in to comment using ';
    const isUserAnonymous = user && user.id.substr(0, 10) === 'anonymous_';

    return (
      <div className={b('auth-panel', {}, { theme: props.theme, loggedIn })}>
        {user && (
          <div className="auth-panel__column">
            You signed in as{' '}
            <Dropdown title={user.name} theme={this.props.theme}>
              <DropdownItem separator={!isUserAnonymous}>
                <UserID id={user.id} theme={this.props.theme} {...getHandleClickProps(this.toggleUserInfoVisibility)} />
              </DropdownItem>

              {!isUserAnonymous && (
                <DropdownItem>
                  <Button
                    kind="link"
                    theme={this.props.theme}
                    onClick={() => requestDeletion().then(() => props.onSignOut())}
                  >
                    Request my data removal
                  </Button>
                </DropdownItem>
              )}
            </Dropdown>{' '}
            <Button
              className="auth-panel__sign-out"
              kind="link"
              theme={this.props.theme}
              onClick={() => props.onSignOut()}
            >
              Sign out?
            </Button>
          </div>
        )}

        {IS_STORAGE_AVAILABLE && !loggedIn && (
          <div className="auth-panel__column">
            {signInMessage}
            {providers.map((provider, i) => {
              const comma = i === 0 ? '' : i === providers.length - 1 ? ' or ' : ', ';

              if (provider === 'anonymous') {
                return (
                  <span>
                    {comma}{' '}
                    <Dropdown
                      title={PROVIDER_NAMES[provider]}
                      titleClass="auth-panel__pseudo-link"
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
                  </span>
                );
              }

              return (
                <span>
                  {comma}
                  <span
                    className="auth-panel__pseudo-link"
                    data-provider={provider}
                    // eslint-disable-next-line @typescript-eslint/no-object-literal-type-assertion
                    {...getHandleClickProps(this.handleOAuthLogin)}
                    role="link"
                  >
                    {PROVIDER_NAMES[provider]}
                  </span>
                </span>
              );
            })}
          </div>
        )}

        {!IS_STORAGE_AVAILABLE && IS_THIRD_PARTY && (
          <div className="auth-panel__column">
            Disable third-party cookies blocking to sign in or open comments in{' '}
            <a
              class="auth-panel__pseudo-link"
              href={`${window.location.origin}/web/comments.html${window.location.search}`}
              target="_blank"
            >
              new page
            </a>
          </div>
        )}

        {!IS_STORAGE_AVAILABLE && !IS_THIRD_PARTY && (
          <div className="auth-panel__column">Allow cookies to sign in and comment</div>
        )}

        <div className="auth-panel__column">
          {user && user.admin && (
            <span
              className="auth-panel__pseudo-link auth-panel__admin-action"
              {...getHandleClickProps(() => this.toggleBlockedVisibility())}
              role="link"
            >
              {isBlockedVisible ? 'Hide' : 'Show'} blocked users
            </span>
          )}

          {user && user.admin && ' • '}

          {user && user.admin && (
            <span
              className="auth-panel__pseudo-link auth-panel__admin-action"
              {...getHandleClickProps(() => this.toggleCommentsAvailability())}
              role="link"
            >
              {isCommentsDisabled ? 'Enable' : 'Disable'} comments
            </span>
          )}

          {user && user.admin && ' • '}

          {!(user && user.admin) && props.postInfo.read_only && (
            <span className="auth-panel__readonly-label">Read-only</span>
          )}

          <span className="auth-panel__sort">
            Sort by{' '}
            <span className="auth-panel__select-label">
              {sortArray.find(x => 'selected' in x && x.selected!)!.label}
              <select className="auth-panel__select" onChange={this.onSortChange} onBlur={this.onSortChange}>
                {sortArray.map(sort => (
                  <option value={sort.value} selected={sort.selected}>
                    {sort.label}
                  </option>
                ))}
              </select>
            </span>
          </span>
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
