import { h, Component } from 'preact';
import { FormattedMessage, IntlShape, useIntl } from 'react-intl';
import clsx from 'clsx';

import { User, Theme, PostInfo } from 'common/types';
import { IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from 'common/constants';
import { postMessageToParent } from 'utils/post-message';
import { getHandleClickProps } from 'common/accessibility';
import { StoreState } from 'store';
import { useTheme } from 'hooks/useTheme';
import { Button } from 'components/button';
import { Auth } from 'components/auth';
import { Avatar } from 'components/avatar';
import { SignOutIcon } from 'components/icons/signout';
import { IconButton } from 'components/icon-button/icon-button';
import { messages } from 'components/auth/auth.messsages';

import styles from './auth-panel.module.css';
import { SubscribeByRSS } from 'components/subscribe-by-rss';
import { StaticStore } from 'common/static-store';
import { SubscribeByEmail } from 'components/subscribe-by-email';

interface OwnProps {
  user: User | null;
  hiddenUsers: StoreState['hiddenUsers'];
  isCommentsDisabled: boolean;
  postInfo: PostInfo;

  signout(): Promise<void>;
  onCommentsChangeReadOnlyMode(readOnly: boolean): Promise<void>;
  onBlockedUsersShow(): void;
  onBlockedUsersHide(): void;
}

export interface Props extends OwnProps {
  intl: IntlShape;
  theme: Theme;
}

interface State {
  isBlockedVisible: boolean;
  anonymousUsernameInputValue: string;
}

class AuthPanelComponent extends Component<Props, State> {
  state = {
    isBlockedVisible: false,
    anonymousUsernameInputValue: 'anon',
  };

  toggleBlockedVisibility = () => {
    if (!this.state.isBlockedVisible) {
      if (this.props.onBlockedUsersShow) this.props.onBlockedUsersShow();
    } else if (this.props.onBlockedUsersHide) this.props.onBlockedUsersHide();

    this.setState({ isBlockedVisible: !this.state.isBlockedVisible });
  };

  toggleCommentsAvailability = () => {
    this.props.onCommentsChangeReadOnlyMode(!this.props.isCommentsDisabled);
  };

  renderAuthorized = (user: User) => {
    return (
      <div className={clsx('user', styles.user)}>
        <button
          className={clsx('user-profile-button', styles.userButton)}
          onClick={() => postMessageToParent({ profile: { ...user, current: '1' } })}
          title={this.props.intl.formatMessage(messages.openProfile)}
        >
          <div className={clsx('user-avatar', styles.userAvatar)}>
            <Avatar url={user.picture} />
          </div>
          {user.name}
        </button>{' '}
        {StaticStore.config.email_notifications && StaticStore.query.show_email_subscription && <SubscribeByEmail />}
        <IconButton title={this.props.intl.formatMessage(messages.signout)} onClick={this.props.signout}>
          <SignOutIcon />
        </IconButton>
      </div>
    );
  };

  renderThirdPartyWarning = () => {
    if (IS_STORAGE_AVAILABLE || !IS_THIRD_PARTY) return null;
    return (
      <div>
        <FormattedMessage
          id="authPanel.disabled-cookies"
          defaultMessage="Disable third-party cookies blocking to login or open comments in"
        />{' '}
        <a
          className="auth-panel__pseudo-link"
          href={`${window.location.origin}/web/comments.html${window.location.search}`}
          target="_blank"
          rel="noreferrer"
        >
          <FormattedMessage id="authPanel.new-page" defaultMessage="new page" />
        </a>
      </div>
    );
  };

  renderCookiesWarning = () => {
    if (IS_STORAGE_AVAILABLE || IS_THIRD_PARTY) {
      return null;
    }
    return (
      <div>
        <FormattedMessage id="authPanel.enable-cookies" defaultMessage="Allow cookies to login and comment" />
      </div>
    );
  };

  renderSettingsLabel = () => {
    return (
      <Button
        kind="link"
        mix="auth-panel__admin-action"
        {...getHandleClickProps(this.toggleBlockedVisibility)}
        role="link"
      >
        {this.state.isBlockedVisible ? (
          <FormattedMessage id="authPanel.hide-settings" defaultMessage="Hide settings" />
        ) : (
          <FormattedMessage id="authPanel.show-settings" defaultMessage="Show settings" />
        )}
      </Button>
    );
  };

  renderReadOnlySwitch = () => {
    const { isCommentsDisabled } = this.props;

    return (
      <Button
        kind="link"
        mix="auth-panel__admin-action"
        {...getHandleClickProps(this.toggleCommentsAvailability)}
        role="link"
      >
        {isCommentsDisabled ? (
          <FormattedMessage id="authPanel.enable-comments" defaultMessage="Enable comments" />
        ) : (
          <FormattedMessage id="authPanel.disable-comments" defaultMessage="Disable comments" />
        )}
      </Button>
    );
  };

  render({ user, postInfo }: Props, { isBlockedVisible }: State) {
    const { read_only } = postInfo;
    const isAdmin = user && user.admin;
    const isSettingsLabelVisible = Object.keys(this.props.hiddenUsers).length > 0 || isAdmin || isBlockedVisible;
    const isAuthorized = !!user;

    return (
      <div className={clsx('top-panel', styles.root, { 'top-panel_loggedin': isAuthorized })}>
        {user ? this.renderAuthorized(user) : read_only && <Auth />}
        {this.renderThirdPartyWarning()}
        {this.renderCookiesWarning()}
				{read_only && (
					<div>
						<FormattedMessage id="authPanel.read-only" defaultMessage="Read-only" />
					</div>
				)}
				{isSettingsLabelVisible && <div>{this.renderSettingsLabel()}</div>}
				{isAdmin && <div>{this.renderReadOnlySwitch()}</div>}
				<span className={styles.rss}>
					<SubscribeByRSS userId={user?.id} />
				</span>
      </div>
    );
  }
}

export function AuthPanel(props: OwnProps) {
  const intl = useIntl();
  const theme = useTheme();

  return <AuthPanelComponent intl={intl} theme={theme} {...props} />;
}
