import { h, Component, Fragment } from 'preact';
import { useSelector } from 'store/context';
import clsx from 'clsx';

import type { IntlShape } from 'common/intl';
import { useIntl, FormattedMessage, defineMessages } from 'common/intl';
import { getDirection } from 'common/direction';
import 'styles/global.css';
import type { StoreState } from 'store';
import { COMMENT_NODE_CLASSNAME_PREFIX, MAX_SHOWN_ROOT_COMMENTS, IS_MOBILE } from 'common/constants';
import { maxShownComments, noFooter, url } from 'common/settings';

import {
  fetchUser,
  blockUser,
  unblockUser,
  fetchBlockedUsers,
  hideUser,
  unhideUser,
  signout,
} from 'store/user/actions';
import { fetchComments, loadMoreComments, addComment, updateComment } from 'store/comments/actions';
import { setCommentsReadOnlyState } from 'store/post-info/actions';

// TODO: make this button as default for all cases and replace current `components/Button`
import { Button } from 'components/auth/components/button';
import { Preloader } from 'components/preloader';
import { Settings } from 'components/settings';
import { AuthPanel } from 'components/auth-panel';
import { SortPicker } from 'components/sort-picker';
import { CommentForm } from 'components/comment-form';
import { Thread } from 'components/thread';
import { ConnectedComment as Comment } from 'components/comment/connected-comment';
import { uploadImage, getPreview } from 'common/api';
import { isUserAnonymous } from 'utils/isUserAnonymous';
import { bindActions } from 'utils/actionBinder';
import { postMessageToParent, parseMessage, isFromParent, updateIframeHeight } from 'utils/post-message';
import { answerReaderAsks } from 'utils/reader';
import { useActions } from 'hooks/useAction';
import { RevealedComments } from 'components/root/in-view/in-view';
import { setCollapse } from 'store/thread/actions';

import styles from './root.module.css';

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
  countLeft: state.comments.page.countLeft,
  pinnedComments: state.comments.pinnedComments.map((id) => state.comments.allComments[id]).filter((c) => !c.hidden),
  theme: state.theme,
  info: state.info,
  hiddenUsers: state.hiddenUsers,
  blockedUsers: state.bannedUsers,
  getPreview,
  uploadImage,
});

const boundActions = bindActions({
  fetchComments,
  loadMoreComments,
  fetchUser,
  fetchBlockedUsers,
  setCommentsReadOnlyState,
  blockUser,
  unblockUser,
  hideUser,
  unhideUser,
  addComment,
  updateComment,
  setCollapse,
  signout,
});

type Props = ReturnType<typeof mapStateToProps> & typeof boundActions & { intl: IntlShape };

interface State {
  isUserLoading: boolean;
  isSettingsVisible: boolean;
  commentsShown: number;
  isLoadingMore: boolean;
  wasSomeoneUnblocked: boolean;
  revealedComments: ReadonlySet<string>;
}

const messages = defineMessages({
  pinnedComments: {
    id: 'root.pinned-comments',
    defaultMessage: 'Pinned comments',
  },
});

const getCollapsedParent = (hash: string, childToParentComments: Record<string, string>) => {
  let id = hash.replace(`#${COMMENT_NODE_CLASSNAME_PREFIX}`, '');
  while (childToParentComments[id]) {
    id = childToParentComments[id];
  }

  return id;
};

/**
 * Comments around the given one, in the order they appear: as many each way as the screen could
 * hold, counting each as half the height of a placeholder so that short comments still fill it.
 */
const getCommentsAround = (comment: Element): Set<string> => {
  const all = Array.from(document.querySelectorAll(`article[id^="${COMMENT_NODE_CLASSNAME_PREFIX}"]`));
  const index = all.indexOf(comment);
  const reach = Math.ceil(window.screen.height / 50);

  return new Set(
    all
      .slice(Math.max(0, index - reach), index + reach + 1)
      .map((node) => node.id.slice(COMMENT_NODE_CLASSNAME_PREFIX.length))
  );
};

/** main component fr main comments widget */
export class Root extends Component<Props, State> {
  state = {
    isUserLoading: true,
    // without max_shown_comments a phone starts with the first MAX_SHOWN_ROOT_COMMENTS and a desktop
    // with every comment loaded, as before the setting applied to desktop
    commentsShown: maxShownComments ?? (IS_MOBILE ? MAX_SHOWN_ROOT_COMMENTS : Infinity),
    isLoadingMore: false,
    wasSomeoneUnblocked: false,
    isSettingsVisible: false,
    revealedComments: new Set<string>(),
  };

  heightObserver: ResizeObserver | null = null;
  heightFallbackTimeout: number | null = null;
  stopAnsweringReaderAsks: (() => void) | null = null;

  startHeightReporting = () => {
    if (this.heightObserver) {
      return;
    }

    updateIframeHeight();
    this.heightObserver = new ResizeObserver(() => updateIframeHeight());
    this.heightObserver.observe(document.body);
  };

  componentDidMount() {
    // if the user fetch hangs, report the preloader height anyway so the parent page
    // is not left with an unbounded iframe
    this.heightFallbackTimeout = window.setTimeout(this.startHeightReporting, 5000);

    const userloading = this.props.fetchUser().finally(() =>
      // start reporting iframe height only after the global preloader is replaced with
      // real content, so the parent page never shrinks the iframe to the preloader size
      this.setState({ isUserLoading: false }, this.startHeightReporting)
    );

    Promise.all([userloading, this.props.fetchComments()]).finally(() => {
      setTimeout(this.checkUrlHash);
      window.addEventListener('hashchange', this.checkUrlHash);
    });

    window.addEventListener('message', this.onMessage);
    this.stopAnsweringReaderAsks = answerReaderAsks(() => this.props.user);
  }

  componentWillUnmount() {
    if (this.heightFallbackTimeout !== null) {
      window.clearTimeout(this.heightFallbackTimeout);
    }
    this.heightObserver?.disconnect();
    this.stopAnsweringReaderAsks?.();
  }

  /** `restLoaded` is set when the thread was loaded to its end for this link, which happens once */
  checkUrlHash = (e?: Event & { newURL: string }, restLoaded = false) => {
    const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      if (!document.getElementById(hash.slice(1))) {
        const id = getCollapsedParent(hash, this.props.childToParentComments);
        if (!restLoaded && this.props.topComments.indexOf(id) === -1 && this.props.countLeft > 0) {
          // the comment is on a page not loaded yet: the rest of the thread comes first, and then the
          // link is followed again from the location, which holds the same hash
          this.props.loadMoreComments({ rest: true }).then(
            () => this.checkUrlHash(undefined, true),
            (err: unknown) => console.error('Remark42: failed to load the linked comment', err)
          );
          return;
        }
        const indexHash = this.props.topComments.findIndex((item) => item === id);
        const multiplierCollapsed = Math.ceil(indexHash / MAX_SHOWN_ROOT_COMMENTS);
        this.setState(
          {
            commentsShown: this.state.commentsShown + MAX_SHOWN_ROOT_COMMENTS * multiplierCollapsed,
          },
          () => setTimeout(() => this.toMessage(hash), 500)
        );
      } else {
        this.toMessage(hash);
      }
    }
  };

  toMessage = (hash: string) => {
    // the hash comes from the embedding page and the answer goes back to it, so it is looked up
    // as the comment id it stands for and never as a selector, which could match anything in here
    const target = document.getElementById(hash.slice(1));
    if (!target) {
      return;
    }

    // comments out of view are placeholders that grow once they come into view, and the frame
    // reports its height only while it is on screen. Scrolled to placeholder positions on a page
    // still sized for the preloader, the host page stops short with its footer in view and then
    // grows under the reader. So the comments the reader will see are rendered first, and the
    // height they take goes out before the position, both measured now.
    this.setState({ revealedComments: getCommentsAround(target) }, () => {
      const comment = document.getElementById(hash.slice(1));
      if (!comment) {
        return;
      }
      updateIframeHeight();
      postMessageToParent({ scrollTo: comment.getBoundingClientRect().top });
      comment.classList.add('comment_highlighting');
      setTimeout(() => {
        comment.classList.remove('comment_highlighting');
      }, 5e3);
    });
  };

  onMessage = (event: MessageEvent) => {
    if (!isFromParent(event)) {
      return;
    }

    const data = parseMessage(event);

    // the theme is applied, store included, by the listener in remark.tsx, which is in place
    // before this component mounts
    // the profile asking for a sign-out. Nothing has been ended yet: the request is made here
    if (data.signout === true) {
      this.props.signout();
    }
  };

  onBlockedUsersShow = async () => {
    if (this.props.user && this.props.user.admin) {
      await this.props.fetchBlockedUsers();
    }
    this.setState({ isSettingsVisible: true });
  };

  onBlockedUsersHide = async () => {
    // if someone was unblocked let's reload comments
    if (this.state.wasSomeoneUnblocked) {
      this.props.fetchComments({ keepLoaded: true });
    }
    this.setState({
      wasSomeoneUnblocked: false,
      isSettingsVisible: false,
    });
  };

  onUnblockSomeone = () => {
    this.setState({ wasSomeoneUnblocked: true });
  };

  /** reveals the loaded threads that are hidden, and once there are none, loads the next page */
  showMore = () => {
    const { commentsShown } = this.state;
    const loaded = this.props.topComments.length;

    if (commentsShown < loaded) {
      this.setState({ commentsShown: commentsShown + MAX_SHOWN_ROOT_COMMENTS });
      return;
    }

    this.setState({ isLoadingMore: true });
    this.props
      .loadMoreComments()
      .then(
        () => this.setState({ commentsShown: Math.max(commentsShown, loaded) + MAX_SHOWN_ROOT_COMMENTS }),
        // the button stays for another try
        (err: unknown) => console.error('Remark42: failed to load more comments', err)
      )
      .finally(() => this.setState({ isLoadingMore: false }));
  };

  render(props: Props, { isUserLoading, commentsShown, isLoadingMore, isSettingsVisible, revealedComments }: State) {
    if (isUserLoading) {
      return <Preloader className={styles.preloader} />;
    }

    const isCommentsDisabled = props.info.read_only!;
    const imageUploadHandler = isUserAnonymous(this.props.user) ? undefined : this.props.uploadImage;

    return (
      <Fragment>
        <AuthPanel
          user={this.props.user}
          hiddenUsers={this.props.hiddenUsers}
          isCommentsDisabled={isCommentsDisabled}
          postInfo={this.props.info}
          signout={this.props.signout}
          onBlockedUsersShow={this.onBlockedUsersShow}
          onBlockedUsersHide={this.onBlockedUsersHide}
          onCommentsChangeReadOnlyMode={this.props.setCommentsReadOnlyState}
        />
        <div>
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
                  mix={styles.input}
                  mode="main"
                  user={props.user}
                  onSubmit={(text: string, title: string) => this.props.addComment(text, title)}
                  getPreview={this.props.getPreview}
                  uploadImage={imageUploadHandler}
                />
              )}
              {this.props.pinnedComments.length > 0 && (
                <div
                  className={styles.pinnedComments}
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
                      mix={styles.pinnedComment}
                    />
                  ))}
                </div>
              )}
              <div className={styles.sortPicker} data-nosnippet>
                <SortPicker />
              </div>
              <RevealedComments.Provider value={revealedComments}>
                <Comments
                  commentsShown={commentsShown}
                  isLoading={props.isCommentsLoading}
                  isLoadingMore={isLoadingMore}
                  topComments={props.topComments}
                  countLeft={props.countLeft}
                  showMore={this.showMore}
                />
              </RevealedComments.Provider>
            </>
          )}
        </div>
      </Fragment>
    );
  }
}

interface CommentsProps {
  isLoading: boolean;
  isLoadingMore: boolean;
  topComments: string[];
  /** comments on the server after the loaded ones */
  countLeft: number;
  commentsShown: number;
  showMore(): void;
}
function Comments({ isLoading, isLoadingMore, topComments, countLeft, commentsShown, showMore }: CommentsProps) {
  const hasHiddenComments = commentsShown < topComments.length;
  const renderComments = hasHiddenComments ? topComments.slice(0, commentsShown) : topComments;
  const isShowMoreButtonVisible = hasHiddenComments || countLeft > 0;

  return (
    <div className={styles.threads}>
      {isLoading ? (
        <Preloader className={styles.preloader} />
      ) : (
        <>
          {renderComments.length > 0 && (
            // the list holds the threads only: the preloader and the button below are not its items
            <div role="list">
              {renderComments.map((id) => (
                <Thread key={`thread-${id}`} id={id} level={0} getPreview={getPreview} />
              ))}
            </div>
          )}
          {isShowMoreButtonVisible && (
            <Button className={styles.moreComments} onClick={showMore} disabled={isLoadingMore}>
              <span data-nosnippet>
                <FormattedMessage id="root.show-more" defaultMessage="Show more" />
              </span>
            </Button>
          )}
        </>
      )}
    </div>
  );
}

const CopyrightLink = (title: string) => (
  <a class={styles.copyrightLink} href="https://remark42.com/">
    {title}
  </a>
);

/** Root component connected to the store */
export function ConnectedRoot() {
  const intl = useIntl();
  const props = useSelector(mapStateToProps);
  const actions = useActions(boundActions);

  return (
    <div
      dir={getDirection(intl.locale)}
      className={clsx(styles.root, props.theme === 'dark' ? styles.themeDark : styles.themeLight, props.theme)}
    >
      <Root {...props} {...actions} intl={intl} />
      {!noFooter && (
        <p className={styles.copyright} role="contentinfo">
          <span data-nosnippet>
            <FormattedMessage
              id="root.powered-by"
              defaultMessage="Powered by <a>Remark42</a>"
              values={{ a: CopyrightLink }}
            />
          </span>
        </p>
      )}
    </div>
  );
}
