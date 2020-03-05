/** @jsx createElement */
import { createElement, Component, FunctionComponent, Fragment } from 'preact';
import { useSelector } from 'react-redux';
import b from 'bem-react-helper';
import { IntlShape, useIntl, FormattedMessage, defineMessages } from 'react-intl';

import { User, Sorting, AuthProvider } from '@app/common/types';
import { COMMENT_NODE_CLASSNAME_PREFIX, MAX_SHOWN_ROOT_COMMENTS, THEMES, IS_MOBILE } from '@app/common/constants';
import { maxShownComments } from '@app/common/settings';

import { StaticStore } from '@app/common/static_store';
import { StoreState } from '@app/store';
import {
  fetchUser,
  logout,
  logIn,
  blockUser,
  unblockUser,
  fetchBlockedUsers,
  hideUser,
  unhideUser,
} from '@app/store/user/actions';
import { fetchComments } from '@app/store/comments/actions';
import { setCommentsReadOnlyState } from '@app/store/post_info/actions';
import { setTheme } from '@app/store/theme/actions';
import { setSort } from '@app/store/sort/actions';
import { addComment, updateComment } from '@app/store/comments/actions';

import { AuthPanel } from '@app/components/auth-panel';
import Settings from '@app/components/settings';
import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { CommentForm } from '@app/components/comment-form';
import { Preloader } from '@app/components/preloader';
import { Thread } from '@app/components/thread';
import { Button } from '@app/components/button';
import { uploadImage, getPreview } from '@app/common/api';
import { isUserAnonymous } from '@app/utils/isUserAnonymous';
import { bindActions } from '@app/utils/actionBinder';
import postMessage from '@app/utils/postMessage';
import { useActions } from '@app/hooks/useAction';

const mapStateToProps = (state: StoreState) => ({
  user: state.user,
  sort: state.sort,
  topComments: state.topComments,
  pinnedComments: state.pinnedComments.map(id => state.comments[id]).filter(c => !c.hidden),
  provider: state.provider,
  theme: state.theme,
  info: state.info,
  hiddenUsers: state.hiddenUsers,
  blockedUsers: state.bannedUsers,
  getPreview,
  uploadImage,
});

const boundActions = bindActions({
  fetchComments,
  fetchUser,
  fetchBlockedUsers,
  logIn,
  logOut: logout,
  setTheme,
  enableComments: () => setCommentsReadOnlyState(false),
  disableComments: () => setCommentsReadOnlyState(true),
  changeSort: setSort,
  blockUser,
  unblockUser,
  hideUser,
  unhideUser,
  addComment,
  updateComment,
});

type Props = ReturnType<typeof mapStateToProps> & typeof boundActions & { intl: IntlShape };

interface State {
  isUserLoading: boolean;
  isCommentsListLoading: boolean;
  isSettingsVisible: boolean;
  commentsShown: number;
  wasSomeoneUnblocked: boolean;
}

const messages = defineMessages({
  pinnedComments: {
    id: `root.pinned-comments`,
    defaultMessage: 'Pinned comments',
  },
});

/** main component fr main comments widget */
export class Root extends Component<Props, State> {
  state = {
    isUserLoading: true,
    isCommentsListLoading: true,
    commentsShown: maxShownComments,
    wasSomeoneUnblocked: false,
    isSettingsVisible: false,
  };

  componentWillMount() {
    const userloading = this.props.fetchUser().finally(() => this.setState({ isUserLoading: false }));

    Promise.all([userloading, this.props.fetchComments(this.props.sort)]).finally(() => {
      postMessage({ remarkIframeHeight: document.body.offsetHeight });
      this.setState({ isCommentsListLoading: false });
      setTimeout(this.checkUrlHash);
      window.addEventListener('hashchange', this.checkUrlHash);
    });

    window.addEventListener('message', this.onMessage.bind(this));
  }

  logIn = async (provider: AuthProvider): Promise<User | null> => {
    const user = await this.props.logIn(provider);

    await this.props.fetchComments(this.props.sort);

    return user;
  };

  logOut = async (): Promise<void> => {
    await this.props.logOut();
    await this.props.fetchComments(this.props.sort);
  };

  checkUrlHash(
    e: Event & {
      newURL?: string;
    }
  ) {
    const hash = e ? `#${e.newURL!.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      const comment = document.querySelector(hash);

      if (comment) {
        setTimeout(() => {
          postMessage({ scrollTo: comment.getBoundingClientRect().top });
        }, 500);
      }
    }
  }

  onMessage(event: { data: string | object }) {
    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.theme && THEMES.includes(data.theme)) {
        this.props.setTheme(data.theme);
      }
    } catch (e) {
      console.error(e); // eslint-disable-line no-console
    }
  }

  onBlockedUsersShow = async () => {
    if (this.props.user && this.props.user.admin) {
      await this.props.fetchBlockedUsers();
    }
    this.setState({ isSettingsVisible: true });
  };

  onBlockedUsersHide = async () => {
    // if someone was unblocked let's reload comments
    if (this.state.wasSomeoneUnblocked) {
      this.props.fetchComments(this.props.sort);
    }
    this.setState({
      wasSomeoneUnblocked: false,
      isSettingsVisible: false,
    });
  };

  async changeSort(sort: Sorting) {
    if (sort === this.props.sort) return;
    this.setState({ isCommentsListLoading: true });
    await this.props.changeSort(sort).catch(() => {});
    this.setState({ isCommentsListLoading: false });
  }

  onUnblockSomeone = () => {
    this.setState({ wasSomeoneUnblocked: true });
  };

  showMore = () => {
    this.setState({
      commentsShown: this.state.commentsShown + MAX_SHOWN_ROOT_COMMENTS,
    });
  };

  /**
   * Defines whether current client is logged in via `Anonymous provider`
   */
  isAnonymous(): boolean {
    return isUserAnonymous(this.props.user);
  }

  render(props: Props, { isUserLoading, isCommentsListLoading, commentsShown, isSettingsVisible }: State) {
    if (isUserLoading) {
      return <Preloader mix="root__preloader" />;
    }

    const isGuest = !props.user;
    const isCommentsDisabled = !!props.info.read_only;
    const imageUploadHandler = this.isAnonymous() ? undefined : this.props.uploadImage;

    return (
      <Fragment>
        <AuthPanel
          theme={this.props.theme}
          user={this.props.user}
          hiddenUsers={this.props.hiddenUsers}
          sort={this.props.sort}
          isCommentsDisabled={isCommentsDisabled}
          postInfo={this.props.info}
          providers={StaticStore.config.auth_providers}
          provider={this.props.provider}
          onSignIn={this.logIn}
          onSignOut={this.logOut}
          onBlockedUsersShow={this.onBlockedUsersShow}
          onBlockedUsersHide={this.onBlockedUsersHide}
          onCommentsEnable={this.props.enableComments}
          onCommentsDisable={this.props.disableComments}
          onSortChange={this.props.changeSort}
        />
        <div className="root__main">
          {isSettingsVisible ? (
            <Settings
              intl={this.props.intl}
              user={this.props.user}
              hiddenUsers={this.props.hiddenUsers}
              blockedUsers={this.props.blockedUsers}
              blockUser={this.props.blockUser}
              unblockUser={this.props.unblockUser}
              hideUser={this.props.hideUser}
              unhideUser={this.props.unhideUser}
              onUnblockSomeone={this.onUnblockSomeone}
            />
          ) : (
            <Fragment>
              {!isGuest && !isCommentsDisabled && (
                <CommentForm
                  intl={this.props.intl}
                  theme={props.theme}
                  mix="root__input"
                  mode="main"
                  user={props.user}
                  onSubmit={(text, title) => this.props.addComment(text, title)}
                  getPreview={this.props.getPreview}
                  uploadImage={imageUploadHandler}
                  simpleView={StaticStore.config.simple_view}
                />
              )}

              {this.props.pinnedComments.length > 0 && (
                <div
                  className="root__pinned-comments"
                  role="region"
                  aria-label={this.props.intl.formatMessage(messages.pinnedComments)}
                >
                  {this.props.pinnedComments.map(comment => (
                    <Comment
                      CommentForm={CommentForm}
                      intl={this.props.intl}
                      key={`pinned-comment-${comment.id}`}
                      view="pinned"
                      data={comment}
                      level={0}
                      disabled={true}
                      mix="root__pinned-comment"
                    />
                  ))}
                </div>
              )}

              {!!this.props.topComments.length && !isCommentsListLoading && (
                <div className="root__threads" role="list">
                  {(IS_MOBILE && commentsShown < this.props.topComments.length
                    ? this.props.topComments.slice(0, commentsShown)
                    : this.props.topComments
                  ).map(id => (
                    <Thread
                      key={`thread-${id}`}
                      id={id}
                      mix="root__thread"
                      level={0}
                      getPreview={this.props.getPreview}
                    />
                  ))}

                  {commentsShown < this.props.topComments.length && IS_MOBILE && (
                    <Button kind="primary" size="middle" mix="root__show-more" onClick={this.showMore}>
                      <FormattedMessage id="root.show-more" defaultMessage="Show more" />
                    </Button>
                  )}
                </div>
              )}

              {isCommentsListLoading && (
                <div className="root__threads" role="list">
                  <Preloader mix="root__preloader" />
                </div>
              )}
            </Fragment>
          )}
        </div>
      </Fragment>
    );
  }
}

/** Root component connected to redux */
export const ConnectedRoot: FunctionComponent = () => {
  const props = useSelector(mapStateToProps);
  const actions = useActions(boundActions);
  const intl = useIntl();

  return (
    <div className={b('root', {}, { theme: props.theme })}>
      <Root {...props} {...actions} intl={intl} />
      <p className="root__copyright" role="contentinfo">
        <FormattedMessage
          id="root.powered-by"
          defaultMessage="Powered by <a>Remark42</a>"
          values={{
            a: (title: string) => (
              <a class="root__copyright-link" href="https://remark42.com/">
                {title}
              </a>
            ),
          }}
        />
      </p>
    </div>
  );
};
