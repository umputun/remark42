/** @jsx h */
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
        {!users.length && <p>There are no blocked users.</p>}

        {!!users.length && <p>List of blocked users:</p>}

        {!!users.length && (
          <ul className="blocked-users__list">
            {users.map(user => {
              const isUserUnblocked = unblockedUsers.includes(user.id);

              return (
                <li className={b('blocked-users__list-item', {}, { view: isUserUnblocked ? 'invisible' : null })}>
                  <span className="blocked-users__username">{user.name}</span>{' '}
                  <span className="blocked-users__user-id">({user.id})</span>
                  <span className="blocked-users__user-block-ttl"> until {formatTime(new Date(user.time))}</span>
                  {isUserUnblocked && (
                    <span {...getHandleClickProps(() => this.block(user))} className="blocked-users__action">
                      block
                    </span>
                  )}
                  {!isUserUnblocked && (
                    <span {...getHandleClickProps(() => this.unblock(user))} className="blocked-users__action">
                      unblock
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    );
  }
}

function formatTime(time) {
  // 'ru-RU' adds a dot as a separator
  const date = time.toLocaleDateString(['ru-RU'], { day: '2-digit', month: '2-digit', year: '2-digit' });

  // do it manually because Intl API doesn't add leading zeros to hours; idk why
  const hours = `0${time.getHours()}`.slice(-2);
  const mins = `0${time.getMinutes()}`.slice(-2);

  return `${date} at ${hours}:${mins}`;
}
