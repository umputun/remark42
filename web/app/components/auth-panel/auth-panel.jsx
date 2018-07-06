/** @jsx h */
import { h, Component } from 'preact';

import { PROVIDER_NAMES } from 'common/constants';
import { getHandleClickProps } from 'common/accessibility';

export default class AuthPanel extends Component {
  constructor(props) {
    super(props);

    this.toggleUserId = this.toggleUserId.bind(this);
    this.toggleBlockedVisibility = this.toggleBlockedVisibility.bind(this);
    this.onSortChange = this.onSortChange.bind(this);
  }

  onSortChange(e) {
    if (this.props.onSortChange) {
      this.props.onSortChange(e.target.value);
    }
  }

  toggleUserId() {
    this.setState({ isUserIdVisible: !this.state.isUserIdVisible });
  }

  toggleBlockedVisibility() {
    if (!this.state.isBlockedVisible) {
      if (this.props.onBlockedUsersShow) this.props.onBlockedUsersShow();
    } else if (this.props.onBlockedUsersHide) this.props.onBlockedUsersHide();

    this.setState({ isBlockedVisible: !this.state.isBlockedVisible });
  }

  render(props, { isUserIdVisible, isBlockedVisible }) {
    const { user, providers = [], sort } = props;

    const sortArray = getSortArray(sort);

    let loggedIn = !!user.id;
    return (
      <div className={b('auth-panel', props, { loggedIn })}>
        {loggedIn && (
          <div className="auth-panel__column">
            You signed in as{' '}
            <strong {...getHandleClickProps(this.toggleUserId)} className="auth-panel__username">
              {user.name}
            </strong>
            {isUserIdVisible && <span className="auth-panel__user-id"> ({user.id})</span>}.{' '}
            <span {...getHandleClickProps(props.onSignOut)} className="auth-panel__pseudo-link" role="link" >
              Sign out?
            </span>
          </div>
        )}

        {!loggedIn && (
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

        <div className="auth-panel__column">
          {user.admin && (
            <span
              className="auth-panel__pseudo-link auth-panel__admin-action"
               {...getHandleClickProps(this.toggleBlockedVisibility)}
              role="link"
            >
              {isBlockedVisible ? 'Hide' : 'Show'} blocked
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
