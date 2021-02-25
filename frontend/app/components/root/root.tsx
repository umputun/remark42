import { h, Component, FunctionComponent, Fragment } from 'preact';
import { useSelector } from 'react-redux';
import b from 'bem-react-helper';
import { IntlShape, useIntl, FormattedMessage, defineMessages } from 'react-intl';
import classnames from 'classnames';

import type { Sorting } from 'common/types';
import type { StoreState } from 'store';
import {
  COMMENT_NODE_CLASSNAME_PREFIX,
  MAX_SHOWN_ROOT_COMMENTS,
  THEMES,
  IS_MOBILE,
  LS_EMAIL_KEY,
} from 'common/constants';
import { maxShownComments, url } from 'common/settings';

import { StaticStore } from 'common/static-store';
import {
  setUser,
  fetchUser,
  blockUser,
  unblockUser,
  fetchBlockedUsers,
  hideUser,
  unhideUser,
} from 'store/user/actions';
import { fetchComments, updateSorting, addComment, updateComment, unsetCommentMode } from 'store/comments/actions';
import { setCommentsReadOnlyState } from 'store/post-info/actions';
import { setTheme } from 'store/theme/actions';

import { Button } from 'components/button';
import Preloader from 'components/preloader';
import Settings from 'components/settings';
import AuthPanel from 'components/auth-panel';
import { CommentForm } from 'components/comment-form';
import { Thread } from 'components/thread';
import { ConnectedComment as Comment } from 'components/comment/connected-comment';
import { uploadImage, getPreview } from 'common/api';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { bindActions } from 'utils/actionBinder';
import postMessage from 'utils/postMessage';
import { useActions } from 'hooks/useAction';
import { setCollapse } from 'store/thread/actions';
import { logout } from 'components/auth/auth.api';

const mapStateToProps = (state: StoreState) => ({
  sort: state.comments.sort,
  isCommentsLoading: state.comments.isFetching,
  user: state.user,
  childToParentComments: Object.entries(state.comments.childComments).reduce(
    (accumulator: Record<string, string>, [key, children]) => {
      children.forEach((child) => (accumulator[child] = key));
      return accumulator;
    },
    {}
  ),
  collapsedThreads: state.collapsedThreads,
  topComments: state.comments.topComments,
  pinnedComments: state.comments.pinnedComments.map((id) => state.comments.allComments[id]).filter((c) => !c.hidden),
  theme: state.theme,
  info: state.info,
  hiddenUsers: state.hiddenUsers,
  blockedUsers: state.bannedUsers,
  getPreview,
  uploadImage,
});

const boundActions = bindActions({
  updateSorting,
  fetchComments,
  setUser,
  fetchUser,
  fetchBlockedUsers,
  setTheme,
  setCommentsReadOnlyState,
  blockUser,
  unblockUser,
  hideUser,
  unhideUser,
  addComment,
  updateComment,
  setCollapse,
  unsetCommentMode,
});

type Props = ReturnType<typeof mapStateToProps> & typeof boundActions & { intl: IntlShape };

interface State {
  isUserLoading: boolean;
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

const getCollapsedParents = (
  hash: string,
  childToParentComments: Record<string, string>,
  collapsedThreads: Record<string, boolean>
) => {
  const collapsedParents = [];
  let id = hash.replace(`#${COMMENT_NODE_CLASSNAME_PREFIX}`, '');

  while (childToParentComments[id]) {
    id = childToParentComments[id];
    if (collapsedThreads[id]) {
      collapsedParents.push(id);
    }
  }

  return collapsedParents;
};

/** main component fr main comments widget */
export class Root extends Component<Props, State> {
  state = {
    isUserLoading: true,
    commentsShown: maxShownComments,
    wasSomeoneUnblocked: false,
    isSettingsVisible: false,
  };

  componentWillMount() {
    const userloading = this.props.fetchUser().finally(() => this.setState({ isUserLoading: false }));

    Promise.all([userloading, this.props.fetchComments()]).finally(() => {
      postMessage({ remarkIframeHeight: document.body.offsetHeight });
      setTimeout(this.checkUrlHash);
      window.addEventListener('hashchange', this.checkUrlHash);
    });

    window.addEventListener('message', this.onMessage.bind(this));
  }

  changeSort = async (sort: Sorting) => {
    if (sort === this.props.sort) return;

    await this.props.updateSorting(sort);
  };

  logout = async () => {
    await logout();
    this.props.setUser();
    this.props.unsetCommentMode();
    localStorage.removeItem(LS_EMAIL_KEY);
    await this.props.fetchComments();
  };

  checkUrlHash = (e: Event & { newURL: string }) => {
    const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      if (!document.querySelector(hash)) {
        const ids = getCollapsedParents(hash, this.props.childToParentComments, this.props.collapsedThreads);
        ids.forEach((id) => this.props.setCollapse(id, false));
      }

      setTimeout(() => {
        const comment = document.querySelector(hash);
        if (comment) {
          postMessage({ scrollTo: comment.getBoundingClientRect().top });
          comment.classList.add('comment_highlighting');
          setTimeout(() => {
            comment.classList.remove('comment_highlighting');
          }, 5e3);
        }
      }, 500);
    }
  };

  onMessage(event: { data: string | object }) {
    if (!event.data) {
      return;
    }

    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.theme && THEMES.includes(data.theme)) {
        this.props.setTheme(data.theme);
      }
    } catch (e) {}
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
      this.props.fetchComments();
    }
    this.setState({
      wasSomeoneUnblocked: false,
      isSettingsVisible: false,
    });
  };

  onUnblockSomeone = () => {
    this.setState({ wasSomeoneUnblocked: true });
  };

  showMore = () => {
    this.setState({
      commentsShown: this.state.commentsShown + MAX_SHOWN_ROOT_COMMENTS,
    });
  };

  render(props: Props, { isUserLoading, commentsShown, isSettingsVisible }: State) {
    if (isUserLoading) {
      return <Preloader mix="root__preloader" />;
    }

    const isCommentsDisabled = props.info.read_only!;
    const imageUploadHandler = isUserAnonymous(this.props.user) ? undefined : this.props.uploadImage;

    return (
      <Fragment>
        <AuthPanel
          user={this.props.user}
          hiddenUsers={this.props.hiddenUsers}
          onSortChange={this.changeSort}
          isCommentsDisabled={isCommentsDisabled}
          postInfo={this.props.info}
          onSignOut={this.logout}
          onBlockedUsersShow={this.onBlockedUsersShow}
          onBlockedUsersHide={this.onBlockedUsersHide}
          onCommentsChangeReadOnlyMode={this.props.setCommentsReadOnlyState}
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
            <>
              {!isCommentsDisabled && (
                <CommentForm
                  id={encodeURI(url || '')}
                  intl={this.props.intl}
                  theme={props.theme}
                  mix="root__input"
                  mode="main"
                  user={props.user}
                  onSubmit={(text: string, title: string) => this.props.addComment(text, title)}
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
                  {this.props.pinnedComments.map((comment) => (
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

              {!!this.props.topComments.length && !props.isCommentsLoading && (
                <div className="root__threads" role="list">
                  {(IS_MOBILE && commentsShown < this.props.topComments.length
                    ? this.props.topComments.slice(0, commentsShown)
                    : this.props.topComments
                  ).map((id) => (
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

              {props.isCommentsLoading && (
                <div className="root__threads" role="list">
                  <Preloader mix="root__preloader" />
                </div>
              )}
            </>
          )}
        </div>
      </Fragment>
    );
  }
}

const CopyrightLink = (title: string) => (
  <a class="root__copyright-link" href="https://remark42.com/">
    {title}
  </a>
);

/** Root component connected to redux */
export const ConnectedRoot: FunctionComponent = () => {
  const props = useSelector(mapStateToProps);
  const actions = useActions(boundActions);
  const intl = useIntl();

  return (
    <div className={classnames(b('root', {}, { theme: props.theme }), props.theme)}>
      <Root {...props} {...actions} intl={intl} />
      <p className="root__copyright" role="contentinfo">
        <FormattedMessage
          id="root.powered-by"
          defaultMessage="Powered by <a>Remark42</a>"
          values={{ a: CopyrightLink }}
        />
      </p>
    </div>
  );
};
