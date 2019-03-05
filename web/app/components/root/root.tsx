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
  Tree,
  Sorting,
  Theme,
  Provider,
  BlockTTL,
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
import { StoreState, StoreDispatch } from '@app/store';
import {
  fetchUser,
  logout,
  logIn,
  blockUser,
  unblockUser,
  fetchBlockedUsers,
  setBlockedVisibleState,
} from '@app/store/user/actions';
import { fetchComments } from '@app/store/comments/actions';
import { setCommentsReadOnlyState } from '@app/store/post_info/actions';
import { setTheme } from '@app/store/theme/actions';
import { setSort } from '@app/store/sort/actions';
import { addComment, updateComment } from '@app/store/comments/actions';

import { AuthPanel } from '@app/components/auth-panel';
import BlockedUsers from '@app/components/blocked-users';
import { ConnectedComment as Comment } from '@app/components/comment/connected-comment';
import { Input } from '@app/components/input';
import Preloader from '@app/components/preloader';
import { Thread } from '@app/components/thread';

interface Props {
  user: User | null;
  sort: Sorting;
  comments: Node[];
  pinnedComments: CommentType[];
  theme: Theme;
  info: PostInfo;
  bannedUsers: BlockedUser[];
  isBlockedVisible: boolean;

  fetchComments(sort: Sorting): Promise<Tree>;
  fetchUser(): Promise<User | null>;
  fetchBlockedUsers(): Promise<BlockedUser[]>;
  logIn(): Promise<User | null>;
  logOut(): Promise<void>;
  setTheme: (theme: Theme) => void;
  setBlockedVisible: (value: boolean) => boolean;
  changeSort(sort: Sorting): Promise<void>;
  enableComments(): Promise<boolean>;
  disableComments(): Promise<boolean>;
  getPreview(text: string): Promise<string>;
  blockUser(id: User['id'], name: User['name'], ttl: BlockTTL): Promise<void>;
  unblockUser(id: User['id']): Promise<void>;
  addComment(text: string, title: string, pid?: CommentType['id']): Promise<void>;
  updateComment(id: string, text: string): Promise<void>;
}

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
    });

    window.addEventListener('message', this.onMessage.bind(this));
  }

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

  onMessage(event: { data: any }) {
    try {
      const data = typeof event.data === 'string' ? JSON.parse(event.data) : event.data;
      if (data.theme && THEMES.includes(data.theme)) {
        this.props.setTheme(data.theme);
      }
    } catch (e) {
      console.error(e); // eslint-disable-line no-console
    }
  }

  onBlockedUsersShow() {
    this.props.fetchBlockedUsers().then(() => {
      this.props.setBlockedVisible(true);
    });
  }

  onBlockedUsersHide() {
    // if someone was unblocked let's reload comments
    if (this.state.wasSomeoneUnblocked) {
      this.props.fetchComments(this.props.sort);
    }
    this.props.setBlockedVisible(false),
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

    return (
      <div id={NODE_ID}>
        <div className={b('root', {}, { theme: props.theme })}>
          <AuthPanel
            theme={this.props.theme}
            user={this.props.user}
            sort={this.props.sort}
            providers={StaticStore.config.auth_providers}
            isCommentsDisabled={isCommentsDisabled}
            postInfo={this.props.info}
            onSignIn={this.props.logIn}
            onSignOut={this.props.logOut}
            onBlockedUsersShow={this.onBlockedUsersShow}
            onBlockedUsersHide={this.onBlockedUsersHide}
            onCommentsEnable={this.props.enableComments}
            onCommentsDisable={this.props.disableComments}
            onSortChange={this.props.changeSort}
          />

          {!this.props.isBlockedVisible && (
            <div className="root__main">
              {!isGuest && !isCommentsDisabled && (
                <Input
                  theme={props.theme}
                  mix="root__input"
                  mode="main"
                  userId={this.props.user!.id}
                  onSubmit={(text, title) => this.props.addComment(text, title)}
                  getPreview={this.props.getPreview}
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

          {this.props.isBlockedVisible && (
            <div className="root__main">
              <BlockedUsers
                users={this.props.bannedUsers}
                blockUser={this.props.blockUser}
                unblockUser={this.props.unblockUser}
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

const mapDispatchToProps = (dispatch: StoreDispatch) => {
  return {
    fetchComments: (sort: Sorting) => dispatch(fetchComments(sort)),
    fetchUser: () => dispatch(fetchUser()),
    fetchBlockedUsers: () => dispatch(fetchBlockedUsers()),
    setBlockedVisible: (value: boolean) => dispatch(setBlockedVisibleState(value)),
    logIn: (provider: Provider) => dispatch(logIn(provider)),
    logOut: () => dispatch(logout()),
    setTheme: (theme: Theme) => dispatch(setTheme(theme)),
    enableComments: () => dispatch(setCommentsReadOnlyState(false)),
    disableComments: () => dispatch(setCommentsReadOnlyState(true)),
    changeSort: (sort: Sorting) => dispatch(setSort(sort)),
    blockUser: (id: User['id'], name: User['name'], ttl: BlockTTL) => dispatch(blockUser(id, name, ttl)),
    unblockUser: (id: User['id']) => dispatch(unblockUser(id)),
    addComment: (text: string, pageTitle: string, pid?: CommentType['id']) =>
      dispatch(addComment(text, pageTitle, pid)),
    updateComment: (id: CommentType['id'], text: string) => dispatch(updateComment(id, text)),
  };
};

/** Root component connected to redux */
export const ConnectedRoot = connect(
  (state: StoreState) => ({
    user: state.user,
    sort: state.sort,
    isBlockedVisible: state.isBlockedVisible,
    comments: state.comments,
    pinnedComments: state.pinnedComments,
    theme: state.theme,
    info: state.info,
    bannedUsers: state.bannedUsers,
  }),
  mapDispatchToProps
)(Root);
