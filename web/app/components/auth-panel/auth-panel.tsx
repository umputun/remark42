/** @jsx h */
import { h, Component, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { PROVIDER_NAMES, IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from '@app/common/constants';
import { requestDeletion } from '@app/utils/email';
import { getHandleClickProps } from '@app/common/accessibility';
import { User, Provider, Sorting, Theme, PostInfo } from '@app/common/types';

import Dropdown, { DropdownItem } from '@app/components/dropdown';
import { Button } from '@app/components/button';
import { UserID } from './__user-id';

export interface Props {
  user: User | null;
  providers: Provider[];
  sort: Sorting;
  isCommentsDisabled: boolean;
  theme: Theme;
  postInfo: PostInfo;

  onSortChange(s: Sorting): Promise<void>;
  onSignIn(p: Provider): Promise<User | null>;
  onSignOut(): Promise<void>;
  onCommentsEnable(): Promise<boolean>;
  onCommentsDisable(): Promise<boolean>;
  onBlockedUsersShow(): void;
  onBlockedUsersHide(): void;
}

interface State {
  isBlockedVisible: boolean;
}

export class AuthPanel extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      isBlockedVisible: false,
    };

    this.toggleBlockedVisibility = this.toggleBlockedVisibility.bind(this);
    this.toggleCommentsAvailability = this.toggleCommentsAvailability.bind(this);
    this.onSortChange = this.onSortChange.bind(this);
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

  getUserTitle() {
    const { user } = this.props;
    return <span className="auth-panel__username">{user!.name}</span>;
  }

  render(props: RenderableProps<Props>, { isBlockedVisible }: State) {
    const { user, providers = [], sort, isCommentsDisabled } = props;
    const sortArray = getSortArray(sort);
    const loggedIn = !!user;
    const signInMessage = props.postInfo.read_only ? 'Sign in using ' : 'Sign in to comment using ';

    return (
      <div className={b('auth-panel', {}, { theme: props.theme, loggedIn })}>
        {user && (
          <div className="auth-panel__column">
            You signed in as{' '}
            <Dropdown title={user.name} theme={this.props.theme}>
              <DropdownItem separator={true}>
                <UserID id={user.id} theme={this.props.theme} />
              </DropdownItem>

              <DropdownItem>
                <Button
                  kind="link"
                  theme={this.props.theme}
                  onClick={() => requestDeletion().then(() => props.onSignOut())}
                >
                  Request my data removal
                </Button>
              </DropdownItem>
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

              return (
                <span>
                  {comma}
                  <span
                    className="auth-panel__pseudo-link"
                    {...getHandleClickProps(() => props.onSignIn(provider))}
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
