/** @jsx createElement */
import { createElement, Component } from 'preact';
import b from 'bem-react-helper';

import { User, BlockedUser, Theme, BlockTTL } from '@app/common/types';
import { getHandleClickProps } from '@app/common/accessibility';
import { StoreState } from '@app/store';

interface Props {
  theme: Theme;
  user: StoreState['user'];
  blockedUsers: BlockedUser[];
  hiddenUsers: StoreState['hiddenUsers'];
  blockUser(id: User['id'], name: string, ttl: BlockTTL): Promise<void>;
  unblockUser(id: User['id']): Promise<void>;
  hideUser(user: User): void;
  unhideUser(userid: User['id']): void;
  onUnblockSomeone(): void;
}

interface State {
  /**
   * cached copy so we can
   * reapply block on unblocked user
   */
  blockedUsers: BlockedUser[];
  unblockedUsers: (User['id'])[];
  hiddenUsers: { [id: string]: User };
  unhiddenUsers: (User['id'])[];
}

export default class Settings extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      blockedUsers: props.blockedUsers.slice(),
      unblockedUsers: [],
      hiddenUsers: { ...props.hiddenUsers },
      unhiddenUsers: [],
    };
  }

  block = (user: BlockedUser) => {
    if (!confirm(`Do you want to block ${user.name}?`)) return;
    this.setState({
      unblockedUsers: this.state.unblockedUsers.filter(x => x !== user.id),
    });
    this.props.blockUser(user.id, user.name, 'permanently');
  };

  unblock = (user: BlockedUser) => {
    if (!confirm(`Do you want to unblock ${user.name}?`)) return;
    this.setState({ unblockedUsers: this.state.unblockedUsers.concat([user.id]) });
    this.props.unblockUser(user.id);
    this.props.onUnblockSomeone();
  };

  hide = (user: User) => {
    this.setState({
      unhiddenUsers: this.state.unhiddenUsers.filter(x => x !== user.id),
    });
    this.props.hideUser(user);
  };

  unhide = (user: User) => {
    this.setState({ unhiddenUsers: this.state.unhiddenUsers.concat([user.id]) });
    this.props.unhideUser(user.id);
    this.props.onUnblockSomeone();
  };

  __isUserHidden = (user: User): boolean => {
    if (this.state.unhiddenUsers.indexOf(user.id) === -1) return true;
    return false;
  };

  render({ user, theme }: Props, { blockedUsers, unblockedUsers, unhiddenUsers }: State) {
    const hiddenUsersList = Object.values(this.state.hiddenUsers);
    return (
      <div className={b('settings', {}, { theme })}>
        <div className="settings__section settings__hidden-users" role="region" aria-label="Hidden users">
          <h3>Hidden users:</h3>
          {!hiddenUsersList.length && <h4 className="settings__dimmed">There are no hidden users.</h4>}
          {!!hiddenUsersList.length && (
            <ul className="settings__list">
              {hiddenUsersList.map(user => {
                const isUserUnhidden = unhiddenUsers.includes(user.id);

                return (
                  <li className="settings__list-item">
                    <span
                      className={['settings__username', isUserUnhidden ? 'settings__invisible' : null].join(' ')}
                      title={user.id}
                    >
                      {user.name || 'unknown'}
                    </span>
                    {this.__isUserHidden(user) ? (
                      <span className="settings__action" {...getHandleClickProps(() => this.unhide(user))}>
                        show
                      </span>
                    ) : (
                      <span className="settings__action" {...getHandleClickProps(() => this.hide(user))}>
                        hide
                      </span>
                    )}
                    <div>
                      <span className="settings__user-id">
                        id: <span>{user.id}</span>
                      </span>
                    </div>
                  </li>
                );
              })}
            </ul>
          )}
        </div>
        {user && user.admin && (
          <div className="settings__section settings__blocked-users" role="region" aria-label="Blocked users">
            <h3>Blocked users:</h3>

            {!blockedUsers.length && <h4 className="settings__dimmed">There are no blocked users.</h4>}

            {!!blockedUsers.length && (
              <ul className="settings__list settings__blocked-users-list">
                {blockedUsers.map(user => {
                  const isUserUnblocked = unblockedUsers.includes(user.id);

                  return (
                    <li className="settings__list-item">
                      <span
                        className={['settings__username', isUserUnblocked ? 'settings__invisible' : null].join(' ')}
                        title={user.id}
                      >
                        {user.name || 'unknown'}
                      </span>
                      <span className="settings__blocked-users-user-block-ttl"> {formatTime(new Date(user.time))}</span>
                      {isUserUnblocked && (
                        <span {...getHandleClickProps(() => this.block(user))} className="settings__action">
                          block
                        </span>
                      )}
                      {!isUserUnblocked && (
                        <span {...getHandleClickProps(() => this.unblock(user))} className="settings__action">
                          unblock
                        </span>
                      )}
                      <div>
                        <span className="settings__user-id">
                          id: <span>{user.id}</span>
                        </span>
                      </div>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>
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
