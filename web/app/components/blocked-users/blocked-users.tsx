/** @jsx h */
import { h, Component, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { User, BlockedUser, Theme, BlockTTL } from '@app/common/types';
import { getHandleClickProps } from '@app/common/accessibility';

interface Props {
  theme: Theme;
  users: BlockedUser[];
  blockUser(id: User['id'], name: string, ttl: BlockTTL): Promise<void>;
  unblockUser(id: User['id']): Promise<void>;
  onUnblockSomeone(): void;
}

interface State {
  /**
   * cached copy so we can
   * reapply block on unblocked user
   */
  users: BlockedUser[];
  unblockedUsers: (User['id'])[];
}

export default class BlockedUsers extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      users: props.users.slice(),
      unblockedUsers: [],
    };
  }

  block(user: BlockedUser) {
    if (confirm(`Do you want to block ${user.name}?`)) {
      this.setState({
        unblockedUsers: this.state.unblockedUsers.filter(x => x !== user.id),
      });
      this.props.blockUser(user.id, user.name, 'permanently');
    }
  }

  unblock(user: BlockedUser) {
    if (confirm(`Do you want to unblock ${user.name}?`)) {
      this.setState({ unblockedUsers: this.state.unblockedUsers.concat([user.id]) });
      this.props.unblockUser(user.id);
      this.props.onUnblockSomeone();
    }
  }

  render({ theme }: RenderableProps<Props>, { users, unblockedUsers }: State) {
    return (
      <div className={b('blocked-users', {}, { theme })} role="region" aria-label="Blocked users">
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
                  <span className="blocked-users__user-block-ttl"> {formatTime(new Date(user.time))}</span>
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

const currentYear = new Date().getFullYear();

function formatTime(time: Date): string {
  // let's assume that if block ttl is more than 50 years then user blocked permanently
  if (time.getFullYear() - currentYear >= 50) return 'permanently';

  // 'ru-RU' adds a dot as a separator
  const date = time.toLocaleDateString(['ru-RU'], { day: '2-digit', month: '2-digit', year: '2-digit' });

  // do it manually because Intl API doesn't add leading zeros to hours; idk why
  const hours = `0${time.getHours()}`.slice(-2);
  const mins = `0${time.getMinutes()}`.slice(-2);

  return `until ${date} at ${hours}:${mins}`;
}
