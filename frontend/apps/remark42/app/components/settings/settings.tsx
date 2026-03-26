import { h, Component } from 'preact';
import clsx from 'clsx';

import { User, BlockedUser, Theme, BlockTTL } from 'common/types';
import { getHandleClickProps } from 'common/accessibility';
import { StoreState } from 'store';
import { defineMessages, IntlShape, FormattedMessage, useIntl } from 'react-intl';
import { useTheme } from 'hooks/useTheme';
import styles from './settings.module.css';

interface Props {
  theme: Theme;
  intl: IntlShape;
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
  unblockedUsers: User['id'][];
  hiddenUsers: { [id: string]: User };
  unhiddenUsers: User['id'][];
}

const messages = defineMessages({
  blockUser: {
    id: 'settings.block-user',
    defaultMessage: 'Do you want to block {userName}?',
  },
  unblockUser: {
    id: 'settings.unblock-user',
    defaultMessage: 'Do you want to unblock {userName}?',
  },
  hiddenUsers: {
    id: 'settings.hidden-users-title',
    defaultMessage: 'Hidden users',
  },
  blockedUsers: {
    id: 'settings.blocked-users-title',
    defaultMessage: 'Blocked users',
  },
});

class SettingsComponent extends Component<Props, State> {
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
    if (!window.confirm(this.props.intl.formatMessage(messages.blockUser, { userName: user.name }))) return;
    this.setState({
      unblockedUsers: this.state.unblockedUsers.filter((x) => x !== user.id),
    });
    this.props.blockUser(user.id, user.name, 'permanently');
  };

  unblock = (user: BlockedUser) => {
    if (!window.confirm(this.props.intl.formatMessage(messages.unblockUser, { userName: user.name }))) return;
    this.setState({ unblockedUsers: this.state.unblockedUsers.concat([user.id]) });
    this.props.unblockUser(user.id);
    this.props.onUnblockSomeone();
  };

  hide = (user: User) => {
    this.setState({
      unhiddenUsers: this.state.unhiddenUsers.filter((x) => x !== user.id),
    });
    this.props.hideUser(user);
  };

  unhide = (user: User) => {
    this.setState({ unhiddenUsers: this.state.unhiddenUsers.concat([user.id]) });
    this.props.unhideUser(user.id);
    this.props.onUnblockSomeone();
  };

  __isUserHidden = (user: User): boolean => {
    return !this.state.unhiddenUsers.includes(user.id);
  };

  render({ user, theme }: Props, { blockedUsers, unblockedUsers, unhiddenUsers }: State) {
    const hiddenUsersList = Object.values(this.state.hiddenUsers);
    const intl = this.props.intl;
    return (
      <div className={clsx(styles.root, theme === 'dark' ? styles.themeDark : styles.themeLight)}>
        <div className={styles.section} role="region" aria-label={intl.formatMessage(messages.hiddenUsers)}>
          <h3>
            <FormattedMessage id="settings.hidden-user-header" defaultMessage="Hidden users:" />
          </h3>
          {!hiddenUsersList.length && (
            <h4 className={styles.dimmed}>
              <FormattedMessage id="settings.no-hidden-users" defaultMessage="There are no hidden users." />
            </h4>
          )}
          {!!hiddenUsersList.length && (
            <ul className={styles.list}>
              {hiddenUsersList.map((user) => {
                const isUserUnhidden = unhiddenUsers.includes(user.id);

                return (
                  <li key={user.id} className={styles.listItem}>
                    <span className={clsx(styles.username, isUserUnhidden && styles.invisible)} title={user.id}>
                      {user.name ? user.name : <FormattedMessage id="settings.unknown" defaultMessage="unknown" />}
                    </span>
                    {this.__isUserHidden(user) ? (
                      <span className={styles.action} {...getHandleClickProps(() => this.unhide(user))}>
                        <FormattedMessage id="settings.show" defaultMessage="show" />
                      </span>
                    ) : (
                      <span className={styles.action} {...getHandleClickProps(() => this.hide(user))}>
                        <FormattedMessage id="settings.hide" defaultMessage="hide" />
                      </span>
                    )}
                    <div>
                      <span className={styles.userId}>
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
          <div className={styles.section} role="region" aria-label={intl.formatMessage(messages.blockedUsers)}>
            <h3>
              <FormattedMessage id="settings.blocked-users-header" defaultMessage="Blocked users:" />
            </h3>

            {!blockedUsers.length && (
              <h4 className={styles.dimmed}>
                <FormattedMessage id="settings.no-blocked-users" defaultMessage="There are no blocked users." />
              </h4>
            )}

            {!!blockedUsers.length && (
              <ul className={styles.list}>
                {blockedUsers.map((user) => {
                  const isUserUnblocked = unblockedUsers.includes(user.id);

                  return (
                    <li key={user.id} className={styles.listItem}>
                      <span className={clsx(styles.username, isUserUnblocked && styles.invisible)} title={user.id}>
                        {user.name ? user.name : <FormattedMessage id="settings.unknown" defaultMessage="unknown" />}
                      </span>
                      <span>
                        {' '}
                        <FormatTime time={new Date(user.time)} />
                      </span>
                      {isUserUnblocked && (
                        <span {...getHandleClickProps(() => this.block(user))} className={styles.action}>
                          <FormattedMessage id="settings.block" defaultMessage="block" />
                        </span>
                      )}
                      {!isUserUnblocked && (
                        <span {...getHandleClickProps(() => this.unblock(user))} className={styles.action}>
                          <FormattedMessage id="settings.unblock" defaultMessage="unblock" />
                        </span>
                      )}
                      <div>
                        <span className={styles.userId}>
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

export function Settings(props: Omit<Props, 'theme'>) {
  const theme = useTheme();

  return <SettingsComponent theme={theme} {...props} />;
}

function FormatTime({ time }: { time: Date }) {
  const intl = useIntl();
  const currentYear = new Date().getFullYear();
  // let's assume that if block ttl is more than 50 years then user blocked permanently
  if (time.getFullYear() - currentYear >= 50)
    return <FormattedMessage id="settings.permanently" defaultMessage="permanently" />;

  return (
    <FormattedMessage
      id="settings.block-time"
      defaultMessage="until {day} at {time}"
      values={{
        day: intl.formatDate(time),
        time: intl.formatTime(time),
      }}
    />
  );
}
