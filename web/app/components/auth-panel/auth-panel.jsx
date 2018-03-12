import { h, Component } from 'preact';

export default class AuthPanel extends Component {
  constructor(props) {
    super(props);

    this.toggleUserId = this.toggleUserId.bind(this);
  }

  toggleUserId() {
    this.setState({ isUserIdVisible: !this.state.isUserIdVisible });
  }

  render(props, { isUserIdVisible }) {
    const { user, providers = [] } = props;

    return (
      <div className={b('auth-panel', props)}>
        {
          !!user.name && (
            <span>
              You signed in as
              {' '}
              <strong className="auth-panel__username" onClick={this.toggleUserId}>{user.name}</strong>
              {isUserIdVisible && <span className="auth-panel__user-id"> ({user.id})</span>}.
              {' '}
              <span className="auth-panel__pseudo-link" onClick={props.onSignOut}>Sign out?</span>
            </span>
          )
        }

        {
          !user.name && (
            <span>
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
            </span>
          )
        }
      </div>
    );
  }
}
