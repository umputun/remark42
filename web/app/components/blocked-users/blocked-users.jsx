import { h, Component } from 'preact';

import api from 'common/api'

export default class BlockedUsers extends Component {
  constructor(props) {
    super(props);

    this.state = {
      unblockedUsers: [],
    }
  }

  block(user) {
    if (confirm('Do you want to block this user?')) {
      api.blockUser({ id: user.id }).then(() => {
        this.setState({ unblockedUsers: this.state.unblockedUsers.filter(x => x !== user.id) });

        if (this.props.onBlock) this.props.onBlock(user.id);
      });
    }
  }

  unblock(user) {
    if (confirm('Do you want to unblock this user?')) {
      api.unblockUser({ id: user.id }).then(() => {
        this.setState({ unblockedUsers: this.state.unblockedUsers.concat([user.id]) });

        if (this.props.onUnblock) this.props.onUnblock(user.id);
      });
    }
  }

  render(props, { unblockedUsers }) {
    const { users } = props;

    return (
      <div className={b('blocked-users', props)}>
        {
          !users.length && (
            <p>There are no blocked users.</p>
          )
        }

        {
          !!users.length && (
            <p>List of blocked users:</p>
          )
        }

        {
          !!users.length && (
            <ul className="blocked-users__list">
              {
                users.map(user => {
                  const isUserUnblocked = unblockedUsers.includes(user.id);

                  return (
                    <li className={b('blocked-users__list-item', {}, { view: isUserUnblocked ? 'invisible' : null })}>
                      <span className="blocked-users__username">{user.id}</span>

                      {
                        isUserUnblocked && (
                          <span className="blocked-users__action" onClick={() => this.block(user)}>block</span>
                        )
                      }

                      {
                        !isUserUnblocked && (
                          <span className="blocked-users__action" onClick={() => this.unblock(user)}>unblock</span>
                        )
                      }
                    </li>
                  )
                })
              }
            </ul>
          )
        }
      </div>
    );
  }
}
