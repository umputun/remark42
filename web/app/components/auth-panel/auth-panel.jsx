import { h, Component } from 'preact';

export default class AuthPanel extends Component {
  render(props, state) {
    const { user, providers = [] } = props;

    return (
      <div className={b('auth-panel', props)}>
        {
          !!user.name && (
            <span>You signed in as <strong>{user.name}</strong>. <span className="auth-panel__pseudo-link" onClick={props.onSignOut}>Sign out?</span></span>
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
