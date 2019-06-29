/** @jsx h */
import { h, Component, RenderableProps } from 'preact';
import { connect } from 'preact-redux';
import b from 'bem-react-helper';

import {
  User,
  Node,
  PostInfo,
  BlockedUser,
  Comment as CommentType,
  Sorting,
  Theme,
  AuthProvider,
} from '@app/common/types';
import {
  NODE_ID,
  COMMENT_NODE_CLASSNAME_PREFIX,
  MAX_SHOWN_ROOT_COMMENTS,
  THEMES,
  IS_MOBILE,
} from '@app/common/constants';
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
  setSettingsVisibleState,
  hideUser,
  unhideUser,
} from '@app/store/user/actions';
import { fetchComments, fetchNewComments } from '@app/store/comments/actions';
import { setCommentsReadOnlyState } from '@app/store/post_info/actions';
import { setTheme } from '@app/store/theme/actions';
import { setSort } from '@app/store/sort/actions';
import { addComment, updateComment } from '@app/store/comments/actions';

import { AuthPanel } from '@app/components/auth-panel';
import Settings from '@app/components/settings';
import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { Input } from '@app/components/input';
import Preloader from '@app/components/preloader';
import { Thread } from '@app/components/thread';
import { uploadImage, getPreview, connectToUpdatesStream } from '@app/common/api';
import { isUserAnonymous } from '@app/utils/isUserAnonymous';
import { bindActions } from '@app/utils/actionBinder';
import { throttle } from '@app/utils/throttle';

const boundActions = bindActions({
  fetchComments,
  fetchNewComments,
  fetchUser,
  fetchBlockedUsers,
  setSettingsVisible: setSettingsVisibleState,
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

type Props = {
  user: User | null;
  sort: Sorting;
  comments: Node[];
  pinnedComments: CommentType[];
  theme: Theme;
  info: PostInfo;
  hiddenUsers: StoreState['hiddenUsers'];
  blockedUsers: BlockedUser[];
  isSettingsVisible: boolean;
  getPreview: typeof getPreview;
  uploadImage: typeof uploadImage;
  connectToUpdatesStream?: typeof connectToUpdatesStream;
} & typeof boundActions;

interface State {
  isLoaded: boolean;
  isCommentsListLoading: boolean;
  commentsShown: number;
  wasSomeoneUnblocked: boolean;
}

/** main component fr main comments widget */
export class Root extends Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      isLoaded: false,
      isCommentsListLoading: false,
      commentsShown: maxShownComments,
      wasSomeoneUnblocked: false,
    };

    this.onBlockedUsersShow = this.onBlockedUsersShow.bind(this);
    this.onBlockedUsersHide = this.onBlockedUsersHide.bind(this);
    this.onUnblockSomeone = this.onUnblockSomeone.bind(this);
    this.showMore = this.showMore.bind(this);
  }

  async componentWillMount() {
    Promise.all([this.props.fetchUser(), this.props.fetchComments(this.props.sort)]).finally(() => {
      this.setState({
        isLoaded: true,
      });

      setTimeout(this.checkUrlHash);
      window.addEventListener('hashchange', this.checkUrlHash);

      const onMessage = throttle(() => this.props.fetchNewComments(), 10000);

      this.props.connectToUpdatesStream &&
        this.props.connectToUpdatesStream({
          onMessage,
        });
    });

    window.addEventListener('message', this.onMessage.bind(this));
  }

  logIn = async (p: AuthProvider): Promise<User | null> => {
    const user = await this.props.logIn(p);
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
          window.parent.postMessage(JSON.stringify({ scrollTo: comment.getBoundingClientRect().top }), '*');
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

  async onBlockedUsersShow() {
    if (this.props.user && this.props.user.admin) {
      await this.props.fetchBlockedUsers();
    }
    this.props.setSettingsVisible(true);
  }

  async onBlockedUsersHide() {
    // if someone was unblocked let's reload comments
    if (this.state.wasSomeoneUnblocked) {
      this.props.fetchComments(this.props.sort);
    }
    this.props.setSettingsVisible(false);
    this.setState({
      wasSomeoneUnblocked: false,
    });
  }

  async changeSort(sort: Sorting) {
    if (sort === this.props.sort) return;
    this.setState({ isCommentsListLoading: true });
    await this.props.changeSort(sort).catch(() => {});
    this.setState({ isCommentsListLoading: false });
  }

  onUnblockSomeone() {
    this.setState({ wasSomeoneUnblocked: true });
  }

  showMore() {
    this.setState({
      commentsShown: this.state.commentsShown + MAX_SHOWN_ROOT_COMMENTS,
    });
  }

  /**
   * Defines whether current client is logged in via `Anonymous provider`
   */
  isAnonymous(): boolean {
    return isUserAnonymous(this.props.user);
  }

  render(props: RenderableProps<Props>, { isLoaded, isCommentsListLoading, commentsShown }: State) {
    if (!isLoaded) {
      return (
        <div id={NODE_ID}>
          <div className={b('root', {}, { theme: props.theme })}>
            <Preloader mix="root__preloader" />
          </div>
        </div>
      );
    }

    const isGuest = !props.user;
    const isCommentsDisabled = !!props.info.read_only;
    const imageUploadHandler = this.isAnonymous() ? undefined : this.props.uploadImage;

    return (
      <div id={NODE_ID}>
        <div className={b('root', {}, { theme: props.theme })}>
          <AuthPanel
            theme={this.props.theme}
            user={this.props.user}
            hiddenUsers={this.props.hiddenUsers}
            sort={this.props.sort}
            providers={StaticStore.config.auth_providers}
            isCommentsDisabled={isCommentsDisabled}
            postInfo={this.props.info}
            onSignIn={this.logIn}
            onSignOut={this.logOut}
            onBlockedUsersShow={this.onBlockedUsersShow}
            onBlockedUsersHide={this.onBlockedUsersHide}
            onCommentsEnable={this.props.enableComments}
            onCommentsDisable={this.props.disableComments}
            onSortChange={this.props.changeSort}
          />

          {!this.props.isSettingsVisible && (
            <div className="root__main">
              {!isGuest && !isCommentsDisabled && (
                <Input
                  theme={props.theme}
                  mix="root__input"
                  mode="main"
                  userId={this.props.user!.id}
                  onSubmit={(text, title) => this.props.addComment(text, title)}
                  getPreview={this.props.getPreview}
                  uploadImage={imageUploadHandler}
                />
              )}

              {this.props.pinnedComments.length > 0 && (
                <div className="root__pinned-comments" role="region" aria-label="Pinned comments">
                  {this.props.pinnedComments.map(comment => (
                    <Comment view="pinned" data={comment} level={0} disabled={true} mix="root__pinned-comment" />
                  ))}
                </div>
              )}

              {!!this.props.comments.length && !isCommentsListLoading && (
                <div className="root__threads" role="list">
                  {(IS_MOBILE ? this.props.comments.slice(0, commentsShown) : this.props.comments).map(thread => (
                    <Thread
                      key={thread.comment.id}
                      mix="root__thread"
                      level={0}
                      data={thread}
                      getPreview={this.props.getPreview}
                    />
                  ))}

                  {commentsShown < this.props.comments.length && IS_MOBILE && (
                    <button className="root__show-more" onClick={this.showMore}>
                      Show more
                    </button>
                  )}
                </div>
              )}

              {isCommentsListLoading && (
                <div className="root__threads" role="list">
                  <Preloader mix="root__preloader" />
                </div>
              )}
            </div>
          )}

          {this.props.isSettingsVisible && (
            <div className="root__main">
              <Settings
                user={this.props.user}
                hiddenUsers={this.props.hiddenUsers}
                blockedUsers={this.props.blockedUsers}
                blockUser={this.props.blockUser}
                unblockUser={this.props.unblockUser}
                hideUser={this.props.hideUser}
                unhideUser={this.props.unhideUser}
                onUnblockSomeone={this.onUnblockSomeone}
              />
            </div>
          )}

          <p className="root__copyright" role="contentinfo">
            Powered by{' '}
            <a href="https://remark42.com/" className="root__copyright-link">
              Remark42
            </a>
          </p>
        </div>
      </div>
    );
  }
}

/** Root component connected to redux */
export const ConnectedRoot = connect(
  (
    state: StoreState
  ): Pick<
    Props,
    | 'user'
    | 'sort'
    | 'isSettingsVisible'
    | 'comments'
    | 'pinnedComments'
    | 'theme'
    | 'info'
    | 'hiddenUsers'
    | 'blockedUsers'
    | 'getPreview'
    | 'uploadImage'
    | 'connectToUpdatesStream'
  > => ({
    user: state.user,
    sort: state.sort,
    isSettingsVisible: state.isSettingsVisible,
    comments: state.comments,
    pinnedComments: state.pinnedComments,
    theme: state.theme,
    info: state.info,
    hiddenUsers: state.hiddenUsers,
    blockedUsers: state.bannedUsers,
    getPreview,
    uploadImage,
    connectToUpdatesStream,
  }),
  boundActions
)(Root);
