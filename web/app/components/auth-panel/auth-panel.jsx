/** @jsx h */
import { h, Component } from 'preact';

import Dropdown, { DropdownItem } from 'components/dropdown';
import Button from 'components/button';
import { PROVIDER_NAMES, IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from 'common/constants';
import { requestDeletion } from 'utils/email';
import { getHandleClickProps } from 'common/accessibility';

import UserId from './__user-id';

export default class AuthPanel extends Component {
  constructor(props) {
    super(props);

    this.toggleBlockedVisibility = this.toggleBlockedVisibility.bind(this);
    this.toggleCommentsAvailability = this.toggleCommentsAvailability.bind(this);
    this.onSortChange = this.onSortChange.bind(this);
  }

  onSortChange(e) {
    if (this.props.onSortChange) {
      this.props.onSortChange(e.target.value);
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
      if (this.props.onCommentsEnable) {
        this.props.onCommentsEnable();
      }
    } else {
      if (this.props.onCommentsDisable) {
        this.props.onCommentsDisable();
      }
    }
  }

  getUserTitle() {
    const { user } = this.props;
    return <span className="auth-panel__username">{user.name}</span>;
  }

  render(props, { isBlockedVisible }) {
    const { user, providers = [], sort, isCommentsDisabled } = props;

    const sortArray = getSortArray(sort);

    let loggedIn = !!user.id;
    return (
      <div className={b('auth-panel', props, { loggedIn })}>
        {loggedIn && (
          <div className="auth-panel__column">
            You signed in as{' '}
            <Dropdown title={user.name}>
              <DropdownItem separator>
                <UserId id={user.id} />
              </DropdownItem>

              <DropdownItem>
                <Button mods={{ kind: 'link' }} onClick={() => requestDeletion().then(props.onSignOut)}>
                  Request my data removal
                </Button>
              </DropdownItem>
            </Dropdown>{' '}
            <Button className="auth-panel__sign-out" mods={{ kind: 'link' }} onClick={props.onSignOut}>
              Sign out?
            </Button>
          </div>
        )}

        {IS_STORAGE_AVAILABLE && !loggedIn && (
          <div className="auth-panel__column">
            Sign in to comment using{' '}
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
            {'.'}
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
          {user.admin && (
            <span
              className="auth-panel__pseudo-link auth-panel__admin-action"
              {...getHandleClickProps(this.toggleBlockedVisibility)}
              role="link"
            >
              {isBlockedVisible ? 'Hide' : 'Show'} blocked users
            </span>
          )}

          {user.admin && ' • '}

          {user.admin && (
            <span
              className="auth-panel__pseudo-link auth-panel__admin-action"
              {...getHandleClickProps(this.toggleCommentsAvailability)}
              role="link"
            >
              {isCommentsDisabled ? 'Enable' : 'Disable'} comments
            </span>
          )}

          {user.admin && ' • '}

          <span className="auth-panel__sort">
            Sort by{' '}
            <span className="auth-panel__select-label">
              {sortArray.find(x => x.selected).label}
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

function getSortArray(currentSort) {
  const sortArray = [
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
  ];

  return sortArray.map(sort => {
    if (sort.value === currentSort) {
      sort.selected = true;
    }

    return sort;
  });
}
