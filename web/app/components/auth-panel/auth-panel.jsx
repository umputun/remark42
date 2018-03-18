import { h, Component } from 'preact';

export default class AuthPanel extends Component {
  constructor(props) {
    super(props);

    this.toggleUserId = this.toggleUserId.bind(this);
    this.toggleBlockedVisibility = this.toggleBlockedVisibility.bind(this);
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
    const { user, providers = [] } = props;

    return (
      <div className={b('auth-panel', props)}>
        {
          !!user.name && (
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
          !user.name && (
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

        {
          user.admin && (
            <div className="auth-panel__column">
              <span
                className="auth-panel__pseudo-link"
                onClick={this.toggleBlockedVisibility}
              >{isBlockedVisible ? 'hide' : 'show'} blocked</span>
            </div>
          )
        }
      </div>
    );
  }
}
