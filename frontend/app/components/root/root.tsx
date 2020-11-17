/** @jsx createElement */
import { createElement, Component, FunctionComponent, Fragment } from 'preact';
import { useSelector } from 'react-redux';
import b from 'bem-react-helper';
import { IntlShape, useIntl, FormattedMessage, defineMessages } from 'react-intl';

import { AuthProvider, Sorting } from '@app/common/types';
import {
  COMMENT_NODE_CLASSNAME_PREFIX,
  MAX_SHOWN_ROOT_COMMENTS,
  THEMES,
  IS_MOBILE,
  LS_EMAIL_KEY,
} from '@app/common/constants';
import { maxShownComments, url } from '@app/common/settings';

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
import { fetchComments, updateSorting } from '@app/store/comments/actions';
import { setCommentsReadOnlyState } from '@app/store/post_info/actions';
import { setTheme } from '@app/store/theme/actions';
import { addComment, updateComment } from '@app/store/comments/actions';

import AuthPanel from '@app/components/auth-panel';
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
import { setCollapse } from '@app/store/thread/actions';

const mapStateToProps = (state: StoreState) => ({
  sort: state.comments.sort,
  isCommentsLoading: state.comments.isFetching,
  user: state.user,
  childToParentComments: Object.entries(state.comments.childComments).reduce(
    (accumulator: Record<string, string>, [key, children]) => {
      children.forEach(child => (accumulator[child] = key));
      return accumulator;
    },
    {}
  ),
  collapsedThreads: state.collapsedThreads,
  topComments: state.comments.topComments,
  pinnedComments: state.comments.pinnedComments.map(id => state.comments.allComments[id]).filter(c => !c.hidden),
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
  fetchUser,
  fetchBlockedUsers,
  logIn,
  logOut: logout,
  setTheme,
  setCommentsReadOnlyState,
  blockUser,
  unblockUser,
  hideUser,
  unhideUser,
  addComment,
  updateComment,
  setCollapse,
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

  logIn = async (provider: AuthProvider) => {
    const user = await this.props.logIn(provider);

    await this.props.fetchComments();

    return user;
  };

  logOut = async () => {
    await this.props.logOut();
    localStorage.removeItem(LS_EMAIL_KEY);
    await this.props.fetchComments();
  };

  checkUrlHash = (e: Event & { newURL?: string }) => {
    const hash = e ? `#${e.newURL!.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      if (!document.querySelector(hash)) {
        const ids = getCollapsedParents(hash, this.props.childToParentComments, this.props.collapsedThreads);
        ids.forEach(id => this.props.setCollapse(id, false));
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

  /**
   * Defines whether current client is logged in via `Anonymous provider`
   */
  isAnonymous = () => isUserAnonymous(this.props.user);

  render(props: Props, { isUserLoading, commentsShown, isSettingsVisible }: State) {
    if (isUserLoading) {
      return <Preloader mix="root__preloader" />;
    }

    const isCommentsDisabled = props.info.read_only!;
    const imageUploadHandler = this.isAnonymous() ? undefined : this.props.uploadImage;

    return (
      <Fragment>
        <AuthPanel
          user={this.props.user}
          hiddenUsers={this.props.hiddenUsers}
          onSortChange={this.changeSort}
          isCommentsDisabled={isCommentsDisabled}
          postInfo={this.props.info}
          onSignIn={this.logIn}
          onSignOut={this.logOut}
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
            <Fragment>
              {!isCommentsDisabled && (
                <CommentForm
                  id={encodeURI(url || '')}
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

              {!!this.props.topComments.length && !props.isCommentsLoading && (
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

              {props.isCommentsLoading && (
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
