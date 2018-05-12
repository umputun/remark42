import { h, Component } from 'preact';

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
    } else {
      if (this.props.onBlockedUsersHide) this.props.onBlockedUsersHide();
    }

    this.setState({ isBlockedVisible: !this.state.isBlockedVisible });
  }

  render(props, { isUserIdVisible, isBlockedVisible }) {
    const { user, providers = [], sort } = props;

    const sortArray = getSortArray(sort);

    return (
      <div className={b('auth-panel', props, { loggedIn: !!user.id })}>
        {
          !!user.id && (
            <div className="auth-panel__column">
              You signed in as
              {' '}
              <strong className="auth-panel__username" onClick={this.toggleUserId}>{user.name}</strong>
              {isUserIdVisible && <span className="auth-panel__user-id"> ({user.id})</span>}.
              {' '}
              <span className="auth-panel__pseudo-link" onClick={props.onSignOut}>Sign out?</span>
            </div>
          )
        }

        {
          !user.id && (
            <div className="auth-panel__column">
              Sign in to comment using
              {' '}
              {
                providers.map((provider, i) => {
                  const comma = i === 0 ? '' : ', ';

                  return (
                    <span>
                      {comma}
                      <span
                        className="auth-panel__pseudo-link"
                        onClick={() => props.onSignIn(provider)}
                      >{provider}</span>
                    </span>
                  )
                })
              }
              {'.'}
            </div>
          )
        }

        <div className="auth-panel__column">
          {
            user.admin && (
              <span
                className="auth-panel__pseudo-link auth-panel__admin-action"
                onClick={this.toggleBlockedVisibility}
              >{isBlockedVisible ? 'Hide' : 'Show'} blocked</span>
            )
          }

          {user.admin && ' • '}

          {
            !!user.id && (
              <span className="auth-panel__sort">
                Sort by
                {' '}
                <select className="auth-panel__select" onChange={this.onSortChange}>
                  {
                    sortArray.map(sort => (
                      <option value={sort.value} selected={sort.selected}>{sort.label}</option>
                    ))
                  }
                </select>
              </span>
            )
          }
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
  ];

  return sortArray.map(sort => {
    if (sort.value === currentSort) {
      sort.selected = true;
    }

    return sort;
  });
}
