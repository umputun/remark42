import { h, Component, Fragment } from 'preact';
import { useSelector } from 'react-redux';
import { FormattedMessage, defineMessages, IntlShape, useIntl } from 'react-intl';
import b from 'bem-react-helper';

import { User, Sorting, Theme, PostInfo } from 'common/types';
import { IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from 'common/constants';
import { requestDeletion } from 'utils/email';
import postMessage from 'utils/postMessage';
import { getHandleClickProps } from 'common/accessibility';
import { StoreState } from 'store';
import { Dropdown, DropdownItem } from 'components/dropdown';
import { Button } from 'components/button';
import Auth from 'components/auth';

import useTheme from 'hooks/useTheme';

export interface OwnProps {
  user: User | null;
  hiddenUsers: StoreState['hiddenUsers'];
  isCommentsDisabled: boolean;
  postInfo: PostInfo;

  onSortChange(s: Sorting): Promise<void>;
  onSignOut(): Promise<void>;
  onCommentsChangeReadOnlyMode(readOnly: boolean): Promise<void>;
  onBlockedUsersShow(): void;
  onBlockedUsersHide(): void;
}

export interface Props extends OwnProps {
  intl: IntlShape;
  theme: Theme;
  sort: Sorting;
}

interface State {
  isBlockedVisible: boolean;
  anonymousUsernameInputValue: string;
  sortSelectFocused: boolean;
}

export class AuthPanel extends Component<Props, State> {
  state = {
    isBlockedVisible: false,
    anonymousUsernameInputValue: 'anon',
    sortSelectFocused: false,
  };

  onSortChange = (e: Event) => {
    const { value } = e.target as HTMLOptionElement;
    this.props.onSortChange(value as Sorting);
  };

  onSortFocus = () => {
    this.setState({ sortSelectFocused: true });
  };

  onSortBlur = (e: Event) => {
    this.setState({ sortSelectFocused: false });
    this.onSortChange(e);
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

  toggleUserInfoVisibility = () => {
    const { user } = this.props;

    if (window.parent && user) {
      postMessage({ isUserInfoShown: true, user });
    }
  };

  renderAuthorized = (user: User) => {
    const { onSignOut, theme } = this.props;
    const isUserAnonymous = user && user.id.substr(0, 10) === 'anonymous_';

    return (
      <>
        <FormattedMessage id="authPanel.logged-as" defaultMessage="You logged in as" />{' '}
        <Dropdown title={user.name} titleClass="auth-panel__user-dropdown-title" theme={theme}>
          <DropdownItem separator={!isUserAnonymous}>
            <div
              id={user.id}
              className={b('auth-panel__user-id', {}, { theme })}
              {...getHandleClickProps(this.toggleUserInfoVisibility)}
            >
              {user.id}
            </div>
          </DropdownItem>

          {!isUserAnonymous && (
            <DropdownItem>
              <Button theme={theme} onClick={() => requestDeletion().then(onSignOut)}>
                <FormattedMessage id="authPanel.request-to-delete-data" defaultMessage="Request my data removal" />
              </Button>
            </DropdownItem>
          )}
        </Dropdown>{' '}
        <Button kind="link" theme={theme} onClick={onSignOut}>
          <FormattedMessage id="authPanel.logout" defaultMessage="Logout?" />
        </Button>
      </>
    );
  };

  renderThirdPartyWarning = () => {
    if (IS_STORAGE_AVAILABLE || !IS_THIRD_PARTY) return null;
    return (
      <div className="auth-panel__column">
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
    if (IS_STORAGE_AVAILABLE || IS_THIRD_PARTY) return null;
    return (
      <div className="auth-panel__column">
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

  renderSort = () => {
    const { sort } = this.props;
    const { sortSelectFocused } = this.state;
    const sortArray = getSortArray(sort, this.props.intl);

    return (
      <span className="auth-panel__sort">
        <FormattedMessage id="commentSort.sort-by" defaultMessage="Sort by" />{' '}
        <span className="auth-panel__select-label">
          <span className={b('auth-panel__select-label-value', {}, { focused: sortSelectFocused })}>
            {sortArray.find((x) => 'selected' in x && x.selected!)!.label}
          </span>
          <select
            className="auth-panel__select"
            onChange={this.onSortChange}
            onFocus={this.onSortFocus}
            onBlur={this.onSortBlur}
          >
            {sortArray.map((sort) => (
              <option value={sort.value} selected={sort.selected}>
                {sort.label}
              </option>
            ))}
          </select>
        </span>
      </span>
    );
  };

  render({ user, postInfo, theme }: Props, { isBlockedVisible }: State) {
    const { read_only } = postInfo;
    const isAdmin = user && user.admin;
    const isSettingsLabelVisible = Object.keys(this.props.hiddenUsers).length > 0 || isAdmin || isBlockedVisible;

    return (
      <div className={b('auth-panel', {}, { theme, loggedIn: !!user })}>
        <div className="auth-panel__column">{user ? this.renderAuthorized(user) : read_only && <Auth />}</div>
        {this.renderThirdPartyWarning()}
        {this.renderCookiesWarning()}
        <div className="auth-panel__column">
          {isSettingsLabelVisible && this.renderSettingsLabel()}
          {isSettingsLabelVisible && ' • '}
          {isAdmin && this.renderReadOnlySwitch()}
          {isAdmin && ' • '}
          {!isAdmin && read_only && (
            <span className="auth-panel__readonly-label">
              <FormattedMessage id="authPanel.read-only" defaultMessage="Read-only" />
            </span>
          )}

          {this.renderSort()}
        </div>
      </div>
    );
  }
}

const sortMessages = defineMessages({
  best: {
    id: 'commentsSort.best',
    defaultMessage: 'Best',
  },
  worst: {
    id: 'commentsSort.worst',
    defaultMessage: 'Worst',
  },
  newest: {
    id: 'commentsSort.newest',
    defaultMessage: 'Newest',
  },
  oldest: {
    id: 'commentsSort.oldest',
    defaultMessage: 'Oldest',
  },
  recentlyUpdated: {
    id: 'commentsSort.recently-updated',
    defaultMessage: 'Recently updated',
  },
  leastRecentlyUpdated: {
    id: 'commentsSort.least-recently-updated',
    defaultMessage: 'Least recently updated',
  },
  mostControversial: {
    id: 'commentsSort.most-controversial',
    defaultMessage: 'Most controversial',
  },
  leastControversial: {
    id: 'commentsSort.least-controversial',
    defaultMessage: 'Least controversial',
  },
});

function getSortArray(currentSort: Sorting, intl: IntlShape) {
  const sortArray: {
    value: Sorting;
    label: string;
    selected?: boolean;
  }[] = [
    {
      value: '-score',
      label: intl.formatMessage(sortMessages.best),
    },
    {
      value: '+score',
      label: intl.formatMessage(sortMessages.worst),
    },
    {
      value: '-time',
      label: intl.formatMessage(sortMessages.newest),
    },
    {
      value: '+time',
      label: intl.formatMessage(sortMessages.oldest),
    },
    {
      value: '-active',
      label: intl.formatMessage(sortMessages.recentlyUpdated),
    },
    {
      value: '+active',
      label: intl.formatMessage(sortMessages.leastRecentlyUpdated),
    },
    {
      value: '-controversy',
      label: intl.formatMessage(sortMessages.mostControversial),
    },
    {
      value: '+controversy',
      label: intl.formatMessage(sortMessages.leastControversial),
    },
  ];

  return sortArray.map((sort) => {
    if (sort.value === currentSort) {
      sort.selected = true;
    }

    return sort;
  });
}

export default function AuthPanelConnected(props: OwnProps) {
  const intl = useIntl();
  const theme = useTheme();
  const sort = useSelector<StoreState, Sorting>((state) => state.comments.sort);

  return <AuthPanel intl={intl} theme={theme} sort={sort} {...props} />;
}
