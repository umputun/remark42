import { h, Component, Fragment } from 'preact';
import { useEffect } from 'preact/hooks';
import { useSelector } from 'react-redux';
import b from 'bem-react-helper';
import { IntlShape, useIntl, FormattedMessage, defineMessages } from 'react-intl';
import clsx from 'clsx';

import 'styles/global.css';
import type { StoreState } from 'store';
import { COMMENT_NODE_CLASSNAME_PREFIX, MAX_SHOWN_ROOT_COMMENTS, THEMES, IS_MOBILE } from 'common/constants';
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
import { fetchComments, addComment, updateComment } from 'store/comments/actions';
import { setCommentsReadOnlyState } from 'store/post-info/actions';
import { setTheme } from 'store/theme/actions';

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
import { postMessageToParent, parseMessage } from 'utils/post-message';
import { useActions } from 'hooks/useAction';
import { setCollapse } from 'store/thread/actions';

import styles from './root.module.css';

/**
 * Sends size of the iframe to parent window
 */
export function updateIframeHeight() {
  postMessageToParent({ height: document.body.offsetHeight });
}

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
  fetchComments,
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
  signout,
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

/** main component fr main comments widget */
export class Root extends Component<Props, State> {
  state = {
    isUserLoading: true,
    commentsShown: maxShownComments,
    wasSomeoneUnblocked: false,
    isSettingsVisible: false,
  };

  componentDidMount() {
    const userloading = this.props.fetchUser().finally(() => this.setState({ isUserLoading: false }));

    Promise.all([userloading, this.props.fetchComments()]).finally(() => {
      setTimeout(this.checkUrlHash);
      window.addEventListener('hashchange', this.checkUrlHash);
    });

    window.addEventListener('message', this.onMessage);
  }

  checkUrlHash = (e: Event & { newURL: string }) => {
    const hash = e ? `#${e.newURL.split('#')[1]}` : window.location.hash;

    if (hash.indexOf(`#${COMMENT_NODE_CLASSNAME_PREFIX}`) === 0) {
      if (e) e.preventDefault();

      if (!document.querySelector(hash)) {
        const id = getCollapsedParent(hash, this.props.childToParentComments);
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
    const comment = document.querySelector(hash);
    if (comment) {
      postMessageToParent({ scrollTo: comment.getBoundingClientRect().top });
      comment.classList.add('comment_highlighting');
      setTimeout(() => {
        comment.classList.remove('comment_highlighting');
      }, 5e3);
    }
  };

  onMessage = (event: MessageEvent) => {
    const data = parseMessage(event);

    if (data.signout === true) {
      this.props.signout(false);
    }

    if (!data.theme || !THEMES.includes(data.theme)) {
      return;
    }

    this.props.setTheme(data.theme);
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
      return <Preloader className="root__preloader" />;
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
              <div className={clsx('sort-picker', styles.sortPicker)}>
                <SortPicker />
              </div>
              <Comments
                commentsShown={commentsShown}
                isLoading={props.isCommentsLoading}
                topComments={props.topComments}
                showMore={this.showMore}
              />
            </>
          )}
        </div>
      </Fragment>
    );
  }
}

interface CommentsProps {
  isLoading: boolean;
  topComments: string[];
  commentsShown: number;
  showMore(): void;
}
function Comments({ isLoading, topComments, commentsShown, showMore }: CommentsProps) {
  const renderComments =
    IS_MOBILE && commentsShown < topComments.length ? topComments.slice(0, commentsShown) : topComments;
  const isShowMoreButtonVisible = IS_MOBILE && commentsShown < topComments.length;

  return (
    <div className="root__threads" role="list">
      {isLoading ? (
        <Preloader className="root__preloader" />
      ) : (
        <>
          {topComments.length > 0 &&
            renderComments.map((id) => (
              <Thread key={`thread-${id}`} id={id} mix="root__thread" level={0} getPreview={getPreview} />
            ))}
          {isShowMoreButtonVisible && (
            <Button className={clsx('more-comments', styles.moreComments)} onClick={showMore}>
              <FormattedMessage id="root.show-more" defaultMessage="Show more" />
            </Button>
          )}
        </>
      )}
    </div>
  );
}

const CopyrightLink = (title: string) => (
  <a class="root__copyright-link" href="https://remark42.com/">
    {title}
  </a>
);

/** Root component connected to redux */
export function ConnectedRoot() {
  const intl = useIntl();
  const props = useSelector(mapStateToProps);
  const actions = useActions(boundActions);

  useEffect(() => {
    const observer = new ResizeObserver(updateIframeHeight);

    updateIframeHeight();
    observer.observe(document.body);
    return () => observer.disconnect();
  }, []);

  return (
    <div className={clsx(b('root', {}, { theme: props.theme }), props.theme)}>
      <Root {...props} {...actions} intl={intl} />
      {!noFooter && (
        <p className="root__copyright" role="contentinfo">
          <FormattedMessage
            id="root.powered-by"
            defaultMessage="Powered by <a>Remark42</a>"
            values={{ a: CopyrightLink }}
          />
        </p>
      )}
    </div>
  );
}
