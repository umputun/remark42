import { h, Component } from 'preact';

import api from 'common/api';
import { getHandleClickProps } from 'common/accessibility';

export default class BlockedUsers extends Component {
  constructor(props) {
    super(props);

    this.state = {
      unblockedUsers: [],
    };
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
      <div className={b('blocked-users', props)} role="region" aria-label="Blocked users">
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
                      <span className="blocked-users__username">{user.name}</span>
                      {' '}
                      <span className="blocked-users__user-id">({user.id})</span>

                      {
                        isUserUnblocked && (
                          <span
                            {...getHandleClickProps(() => this.block(user))}
                            className="blocked-users__action">
                            block
                          </span>
                        )
                      }

                      {
                        !isUserUnblocked && (
                          <span
                            {...getHandleClickProps(() => this.unblock(user))}
                            className="blocked-users__action">
                            unblock
                          </span>
                        )
                      }
                    </li>
                  );
                })
              }
            </ul>
          )
        }
      </div>
    );
  }
}
