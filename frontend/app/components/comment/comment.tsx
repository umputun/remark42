import { h, JSX, Component, createRef, ComponentType } from 'preact';
import b from 'bem-react-helper';

import { getHandleClickProps } from 'common/accessibility';
import { API_BASE, BASE_URL, COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';

import { StaticStore } from 'common/static-store';
import debounce from 'utils/debounce';
import copy from 'common/copy';
import { Theme, BlockTTL, Comment as CommentType, PostInfo, User, CommentMode } from 'common/types';
import { extractErrorMessageFromResponse, FetcherError } from 'utils/errorUtils';
import { isUserAnonymous } from 'utils/isUserAnonymous';

import { CommentFormProps } from 'components/comment-form';
import { AvatarIcon } from 'components/avatar-icon';
import { Button } from 'components/button';
import Countdown from 'components/countdown';
import { getPreview, uploadImage } from 'common/api';
import postMessage from 'utils/postMessage';
import { FormattedMessage, useIntl, IntlShape, defineMessages } from 'react-intl';
import { getVoteMessage, VoteMessagesTypes } from './getVoteMessage';
import { getBlockingDurations } from './getBlockingDurations';
import { boundActions } from './connected-comment';

import './styles';

const messages = defineMessages({
  deleteMessage: {
    id: 'comment.delete-message',
    defaultMessage: 'Do you want to delete this comment?',
  },
  hideUserComments: {
    id: 'comment.hide-user-comment',
    defaultMessage: 'Do you want to hide comments of {userName}?',
  },
  pinComment: {
    id: 'comment.pin-comment',
    defaultMessage: 'Do you want to pin this comment?',
  },
  unpinComment: {
    id: 'comment.unpin-comment',
    defaultMessage: 'Do you want to unpin this comment?',
  },
  verifyUser: {
    id: 'comment.verify-user',
    defaultMessage: 'Do you want to verify {userName}?',
  },
  unverifyUser: {
    id: 'comment.unverify-user',
    defaultMessage: 'Do you want to unverify {userName}?',
  },
  blockUser: {
    id: 'comment.block-user',
    defaultMessage: 'Do you want to block {userName} {duration}?',
  },
  unblockUser: {
    id: 'comment.unblock-user',
    defaultMessage: 'Do you want to unblock this user?',
  },
  deletedComment: {
    id: 'comment.deleted-comment',
    defaultMessage: 'This comment was deleted',
  },
  controversy: {
    id: 'comment.controversy',
    defaultMessage: 'Controversy: {value}',
  },
  toggleVerification: {
    id: 'comment.toggle-verification',
    defaultMessage: 'Toggle verification',
  },
  verifiedUser: {
    id: 'comment.verified-user',
    defaultMessage: 'Verified user',
  },
  unverifiedUser: {
    id: 'comment.unverified-user',
    defaultMessage: 'Unverified user',
  },
  goToParent: {
    id: 'comment.go-to-parent',
    defaultMessage: 'Go to parent comment',
  },
  expiredTime: {
    id: 'comment.expired-time',
    defaultMessage: 'Editing time has expired.',
  },
});

export type CommentProps = {
  user: User | null;
  CommentForm: ComponentType<CommentFormProps> | null;
  data: CommentType;
  repliesCount?: number;
  post_info: PostInfo | null;
  /** whether comment's user is banned */
  isUserBanned?: boolean;
  isCommentsDisabled: boolean;
  /** edit mode: is comment should have reply, or edit Input */
  editMode?: CommentMode;
  /**
   * "main" view used in main case,
   * "pinned" view used in pinned block,
   * "user" is for user comments widget,
   * "preview" is for last comments page
   */
  view: 'main' | 'pinned' | 'user' | 'preview';
  /** defines whether comment should have reply/edit actions */
  disabled?: boolean;
  collapsed?: boolean;
  theme: Theme;
  inView?: boolean;
  level?: number;
  mix?: string;
  getPreview?: typeof getPreview;
  uploadImage?: typeof uploadImage;
  intl: IntlShape;
} & Partial<typeof boundActions>;

export interface State {
  renderDummy: boolean;
  isCopied: boolean;
  editDeadline: Date | null;
  voteErrorMessage: string | null;
  /**
   * delta of the score:
   * default is 0.
   * if user upvoted delta will be incremented
   * if downvoted delta will be decremented
   */
  scoreDelta: number;
  /**
   * score copied from props, that updates instantly,
   * without server response
   */
  cachedScore: number;
  initial: boolean;
}

class Comment extends Component<CommentProps, State> {
  votingPromise: Promise<unknown> = Promise.resolve();
  /** comment text node. Used in comment text copying */
  textNode = createRef<HTMLDivElement>();

  updateState = (props: CommentProps) => {
    const newState: Partial<State> = {
      scoreDelta: props.data.vote,
      cachedScore: props.data.score,
    };

    if (props.inView) {
      newState.renderDummy = false;
    }

    // set comment edit timer
    if (props.user && props.user.id === props.data.user.id) {
      const editDuration = StaticStore.config.edit_duration;
      const timeDiff = StaticStore.serverClientTimeDiff || 0;
      const editDeadline = new Date(new Date(props.data.time).getTime() + timeDiff + editDuration * 1000);

      if (editDeadline < new Date()) {
        newState.editDeadline = null;
      } else {
        newState.editDeadline = editDeadline;
      }
    }

    return newState;
  };

  state = {
    renderDummy: typeof this.props.inView === 'boolean' ? !this.props.inView : false,
    isCopied: false,
    editDeadline: null,
    voteErrorMessage: null,
    scoreDelta: 0,
    cachedScore: this.props.data.score,
    initial: true,
    ...this.updateState(this.props),
  };

  // getHandleClickProps = (handler?: (e: KeyboardEvent | MouseEvent) => void) => {
  //   if (this.state.initial) return null;
  //   if (this.props.inView === false) return null;
  //   return getHandleClickProps(handler);
  // };

  componentWillReceiveProps(nextProps: CommentProps) {
    this.setState(this.updateState(nextProps));
  }

  componentDidMount() {
    // eslint-disable-next-line react/no-did-mount-set-state
    this.setState({ initial: false });
  }

  toggleReplying = () => {
    const { editMode, setReplyEditState, data } = this.props;

    setReplyEditState?.({ id: data.id, state: editMode === CommentMode.Reply ? CommentMode.None : CommentMode.Reply });
  };

  toggleEditing = () => {
    const { editMode, setReplyEditState, data } = this.props;

    setReplyEditState?.({ id: data.id, state: editMode === CommentMode.Edit ? CommentMode.None : CommentMode.Edit });
  };

  toggleUserInfoVisibility = () => {
    if (!window.parent) {
      return;
    }

    const { user } = this.props.data;
    const data = JSON.stringify({ isUserInfoShown: true, user });

    window.parent.postMessage(data, '*');
  };

  togglePin = () => {
    const value = !this.props.data.pin;
    const intl = this.props.intl;
    const promptMessage = value ? intl.formatMessage(messages.pinComment) : intl.formatMessage(messages.unpinComment);

    if (window.confirm(promptMessage)) {
      this.props.setPinState!(this.props.data.id, value);
    }
  };

  toggleVerify = () => {
    const value = !this.props.data.user.verified;
    const userId = this.props.data.user.id;
    const intl = this.props.intl;
    const userName = this.props.data.user.name;
    const promptMessage = value
      ? intl.formatMessage(messages.verifyUser, { userName })
      : intl.formatMessage(messages.unverifyUser, { userName });

    if (window.confirm(promptMessage)) {
      this.props.setVerifiedStatus!(userId, value);
    }
  };

  onBlockUserClick = (e: Event) => {
    const target = e.target as HTMLOptionElement;
    // blur event will be triggered by the confirm pop-up which will start
    // infinite loop of blur -> confirm -> blur -> ...
    // so we trigger the blur event manually and have debounce mechanism to prevent it
    if (e.type === 'change') {
      target.blur();
    }
    // we have to debounce the blockUser function calls otherwise it will be
    // called 2 times (by change event and by blur event)
    this.blockUser(target.value as BlockTTL);
  };

  blockUser = debounce((ttl: BlockTTL): void => {
    const { user } = this.props.data;
    const blockingDurations = getBlockingDurations(this.props.intl);
    const blockDuration = blockingDurations.find((el) => el.value === ttl);
    // blocking duration may be undefined if user hasn't selected anything
    // and ttl equals "Blocking period"
    if (!blockDuration) return;

    const duration = blockDuration.label;
    const blockUser = this.props.intl.formatMessage(messages.blockUser, {
      userName: user.name,
      duration: duration.toLowerCase(),
    });
    if (window.confirm(blockUser)) {
      this.props.blockUser!(user.id, user.name, ttl);
    }
  }, 100);

  onUnblockUserClick = () => {
    const { user } = this.props.data;
    const unblockUser = this.props.intl.formatMessage(messages.unblockUser);

    if (window.confirm(unblockUser)) {
      this.props.unblockUser!(user.id);
    }
  };

  deleteComment = () => {
    const deleteComment = this.props.intl.formatMessage(messages.deleteMessage);

    if (window.confirm(deleteComment)) {
      this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });

      this.props.removeComment!(this.props.data.id);
    }
  };

  hideUser = () => {
    const hideUserComment = this.props.intl.formatMessage(messages.hideUserComments, {
      userName: this.props.data.user.name,
    });
    if (!window.confirm(hideUserComment)) return;
    this.props.hideUser!(this.props.data.user);
  };

  handleVoteError = (e: FetcherError, originalScore: number, originalDelta: number) => {
    this.setState({
      scoreDelta: originalDelta,
      cachedScore: originalScore,
      voteErrorMessage: extractErrorMessageFromResponse(e, this.props.intl),
    });
  };

  sendVotingRequest = (votingValue: number, originalScore: number, originalDelta: number) => {
    this.votingPromise = this.votingPromise
      .then(() => this.props.putCommentVote!(this.props.data.id, votingValue))
      .catch((e) => this.handleVoteError(e, originalScore, originalDelta));
  };

  increaseScore = () => {
    const { cachedScore, scoreDelta } = this.state;

    if (scoreDelta === 1) return;

    this.setState({
      scoreDelta: scoreDelta + 1,
      cachedScore: cachedScore + 1,
      voteErrorMessage: null,
    });

    this.sendVotingRequest(1, cachedScore, scoreDelta);
  };

  decreaseScore = () => {
    const { cachedScore, scoreDelta } = this.state;

    if (scoreDelta === -1) return;

    this.setState({
      scoreDelta: scoreDelta - 1,
      cachedScore: cachedScore - 1,
      voteErrorMessage: null,
    });

    this.sendVotingRequest(-1, cachedScore, scoreDelta);
  };

  addComment = async (text: string, title: string, pid?: CommentType['id']) => {
    await this.props.addComment!(text, title, pid);

    this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });
  };

  updateComment = async (id: CommentType['id'], text: string) => {
    await this.props.updateComment!(id, text);

    this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });
  };

  scrollToParent = (e: Event) => {
    const {
      data: { pid },
    } = this.props;

    e.preventDefault();

    const parentCommentNode = document.getElementById(`${COMMENT_NODE_CLASSNAME_PREFIX}${pid}`);

    if (parentCommentNode) {
      const top = parentCommentNode.getBoundingClientRect().top;
      if (!postMessage({ scrollTo: top })) {
        parentCommentNode.scrollIntoView();
      }
    }
  };

  copyComment = () => {
    const username = this.props.data.user.name;
    const time = this.props.data.time;
    const text = this.textNode.current!.textContent || '';

    copy(`<b>${username}</b>&nbsp;${time}<br>${text.replace(/\n+/g, '<br>')}`);

    this.setState({ isCopied: true }, () => {
      setTimeout(() => this.setState({ isCopied: false }), 3000);
    });
  };

  /**
   * Defines whether current client is admin
   */
  isAdmin = (): boolean => {
    return !!this.props.user && this.props.user.admin;
  };

  /**
   * Defines whether current client is not logged in
   */
  isGuest = (): boolean => {
    return !this.props.user;
  };

  /**
   * Defines whether current client is logged in via `Anonymous provider`
   */
  isAnonymous = (): boolean => {
    return isUserAnonymous(this.props.user);
  };

  /**
   * Defines whether comment made by logged in user
   */
  isCurrentUser = (): boolean => {
    if (this.isGuest()) return false;

    return this.props.data.user.id === this.props.user!.id;
  };

  /**
   * returns reason for disabled downvoting
   */
  getDownvoteDisabledReason = (): string | null => {
    const intl = this.props.intl;
    if (!(this.props.view === 'main' || this.props.view === 'pinned'))
      return getVoteMessage(VoteMessagesTypes.ONLY_POST_PAGE, intl);
    if (this.props.post_info!.read_only) return getVoteMessage(VoteMessagesTypes.READONLY, intl);
    if (this.props.data.delete) return getVoteMessage(VoteMessagesTypes.DELETED, intl);
    if (this.isCurrentUser()) return getVoteMessage(VoteMessagesTypes.OWN_COMMENT, intl);
    if (StaticStore.config.positive_score && this.props.data.score < 1)
      return getVoteMessage(VoteMessagesTypes.ONLY_POSITIVE, intl);
    if (this.isGuest()) return getVoteMessage(VoteMessagesTypes.GUEST, intl);
    if (this.isAnonymous() && !StaticStore.config.anon_vote) return getVoteMessage(VoteMessagesTypes.ANONYMOUS, intl);
    return null;
  };

  /**
   * returns reason for disabled upvoting
   */
  getUpvoteDisabledReason = (): string | null => {
    const intl = this.props.intl;
    if (!(this.props.view === 'main' || this.props.view === 'pinned'))
      return getVoteMessage(VoteMessagesTypes.ONLY_POST_PAGE, intl);
    if (this.props.post_info!.read_only) return getVoteMessage(VoteMessagesTypes.READONLY, intl);
    if (this.props.data.delete) return getVoteMessage(VoteMessagesTypes.DELETED, intl);
    if (this.isCurrentUser()) return getVoteMessage(VoteMessagesTypes.OWN_COMMENT, intl);
    if (this.isGuest()) return getVoteMessage(VoteMessagesTypes.GUEST, intl);
    if (this.isAnonymous() && !StaticStore.config.anon_vote) return getVoteMessage(VoteMessagesTypes.ANONYMOUS, intl);
    return null;
  };

  getCommentControls = (): JSX.Element[] => {
    const isAdmin = this.isAdmin();
    const isCurrentUser = this.isCurrentUser();
    const controls: JSX.Element[] = [];

    if (this.props.data.delete) {
      return controls;
    }

    if (!(this.props.view === 'main' || this.props.view === 'pinned')) {
      return controls;
    }

    if (isAdmin) {
      controls.push(
        this.state.isCopied ? (
          <span className="comment__control comment__control_view_inactive">
            <FormattedMessage id="comment.copied" defaultMessage="Copied!" />
          </span>
        ) : (
          <Button kind="link" {...getHandleClickProps(this.copyComment)} mix="comment__control">
            <FormattedMessage id="comment.copy" defaultMessage="Copy" />
          </Button>
        )
      );

      controls.push(
        <Button kind="link" {...getHandleClickProps(this.togglePin)} mix="comment__control">
          {this.props.data.pin ? (
            <FormattedMessage id="comment.unpin" defaultMessage="Unpin" />
          ) : (
            <FormattedMessage id="comment.pin" defaultMessage="Pin" />
          )}
        </Button>
      );
    }

    if (!isCurrentUser) {
      controls.push(
        <Button kind="link" {...getHandleClickProps(this.hideUser)} mix="comment__control">
          <FormattedMessage id="comment.hide" defaultMessage="Hide" />
        </Button>
      );
    }

    if (isAdmin) {
      if (this.props.isUserBanned) {
        controls.push(
          <Button kind="link" {...getHandleClickProps(this.onUnblockUserClick)} mix="comment__control">
            <FormattedMessage id="comment.unblock" defaultMessage="Unblock" />
          </Button>
        );
      }
      const blockingDurations = getBlockingDurations(this.props.intl);
      if (this.props.user!.id !== this.props.data.user.id && !this.props.isUserBanned) {
        controls.push(
          <span className="comment__control comment__control_select-label">
            <FormattedMessage id="comment.block" defaultMessage="Block" />
            <select className="comment__control_select" onBlur={this.onBlockUserClick} onChange={this.onBlockUserClick}>
              <option disabled selected value={undefined}>
                <FormattedMessage id="comment.blocking-period" defaultMessage="Blocking period" />
              </option>
              {blockingDurations.map((block) => (
                <option value={block.value}>{block.label}</option>
              ))}
            </select>
          </span>
        );
      }

      if (!this.props.data.delete) {
        controls.push(
          <Button kind="link" {...getHandleClickProps(this.deleteComment)} mix="comment__control">
            <FormattedMessage id="comment.delete" defaultMessage="Delete" />
          </Button>
        );
      }
    }
    return controls;
  };

  render(props: CommentProps, state: State) {
    const isAdmin = this.isAdmin();
    const isGuest = this.isGuest();
    const isCurrentUser = this.isCurrentUser();

    const isReplying = props.editMode === CommentMode.Reply;
    const isEditing = props.editMode === CommentMode.Edit;
    const lowCommentScore = StaticStore.config.low_score;
    const downvotingDisabledReason = this.getDownvoteDisabledReason();
    const isDownvotingDisabled = downvotingDisabledReason !== null;
    const upvotingDisabledReason = this.getUpvoteDisabledReason();
    const isUpvotingDisabled = upvotingDisabledReason !== null;
    const editable = props.repliesCount === 0 && state.editDeadline;
    const scoreSignEnabled = !StaticStore.config.positive_score;
    const uploadImageHandler = this.isAnonymous() ? undefined : this.props.uploadImage;
    const commentControls = this.getCommentControls();
    const intl = props.intl;
    const CommentForm = this.props.CommentForm;

    /**
     * CommentType adapted for rendering
     */

    const o = {
      ...props.data,
      controversyText: intl.formatMessage(messages.controversy, {
        value: (props.data.controversy || 0).toFixed(2),
      }),
      text:
        props.view === 'preview'
          ? getTextSnippet(props.data.text)
          : props.data.delete
          ? intl.formatMessage(messages.deletedComment)
          : props.data.text,
      time: new Date(props.data.time),
      orig: isEditing
        ? props.data.orig &&
          props.data.orig.replace(/&[#A-Za-z0-9]+;/gi, (entity) => {
            const span = document.createElement('span');
            span.innerHTML = entity;
            return span.innerText;
          })
        : props.data.orig,
      score: {
        value: Math.abs(state.cachedScore),
        sign: !scoreSignEnabled ? '' : state.cachedScore > 0 ? '+' : state.cachedScore < 0 ? '−' : null,
        view: state.cachedScore > 0 ? 'positive' : state.cachedScore < 0 ? 'negative' : undefined,
      },
      user: {
        ...props.data.user,
        picture:
          props.data.user.picture.indexOf(API_BASE) === 0
            ? `${BASE_URL}${props.data.user.picture}`
            : props.data.user.picture,
      },
    };

    const defaultMods = {
      disabled: props.disabled,
      pinned: props.data.pin,
      // TODO: we also have critical_score, so we need to collapse comments with it in future
      useless:
        !!props.isUserBanned ||
        !!props.data.delete ||
        (props.view !== 'preview' && props.data.score < lowCommentScore && !props.data.pin && !props.disabled),
      // TODO: add default view mod or don't?
      guest: isGuest,
      view: props.view === 'main' || props.view === 'pinned' ? props.data.user.admin && 'admin' : props.view,
      replying: props.view === 'main' && isReplying,
      editing: props.view === 'main' && isEditing,
      theme: props.view === 'preview' ? undefined : props.theme,
      level: props.level,
      collapsed: props.collapsed,
    };

    if (props.view === 'preview') {
      return (
        <article className={b('comment', { mix: props.mix }, defaultMods)}>
          <div className="comment__body">
            {!!o.title && (
              <div className="comment__title">
                <a className="comment__title-link" href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`}>
                  {o.title}
                </a>
              </div>
            )}
            <div className="comment__info">
              {!!o.title && o.user.name}

              {!o.title && (
                <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__username">
                  {o.user.name}
                </a>
              )}
            </div>{' '}
            <div
              className={b('comment__text', { mix: b('raw-content', {}, { theme: props.theme }) })}
              // eslint-disable-next-line react/no-danger
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          </div>
        </article>
      );
    }

    if (this.state.renderDummy && !props.editMode) {
      const [width, height] = this.base
        ? [(this.base as Element).scrollWidth, (this.base as Element).scrollHeight]
        : [100, 100];
      return (
        <article
          id={props.disabled ? undefined : `${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`}
          style={{
            width: `${width}px`,
            height: `${height}px`,
          }}
        />
      );
    }
    const goToParentMessage = intl.formatMessage(messages.goToParent);
    return (
      <article
        className={b('comment', { mix: this.props.mix }, defaultMods)}
        id={props.disabled ? undefined : `${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`}
      >
        {props.view === 'user' && o.title && (
          <div className="comment__title">
            <a className="comment__title-link" href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`}>
              {o.title}
            </a>
          </div>
        )}
        <div className="comment__info">
          {props.view !== 'user' && !props.collapsed && (
            <AvatarIcon mix="comment__avatar" theme={this.props.theme} picture={o.user.picture} />
          )}

          {props.view !== 'user' && (
            <span
              {...getHandleClickProps(this.toggleUserInfoVisibility)}
              className="comment__username"
              title={o.user.id}
            >
              {o.user.name}
            </span>
          )}

          {isAdmin && props.view !== 'user' && (
            <span
              {...getHandleClickProps(this.toggleVerify)}
              aria-label={intl.formatMessage(messages.toggleVerification)}
              title={
                o.user.verified
                  ? intl.formatMessage(messages.verifiedUser)
                  : intl.formatMessage(messages.unverifiedUser)
              }
              className={b('comment__verification', {}, { active: o.user.verified, clickable: true })}
            />
          )}

          {!isAdmin && !!o.user.verified && props.view !== 'user' && (
            <span
              title={intl.formatMessage(messages.verifiedUser)}
              className={b('comment__verification', {}, { active: true })}
            />
          )}

          <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__time">
            <FormatTime time={o.time} />
          </a>

          {!!props.level && props.level > 0 && props.view === 'main' && (
            <a
              className="comment__link-to-parent"
              href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`}
              aria-label={goToParentMessage}
              title={goToParentMessage}
              onClick={(e) => this.scrollToParent(e)}
            >
              {' '}
            </a>
          )}

          {props.isUserBanned && props.view !== 'user' && (
            <span className="comment__status">
              <FormattedMessage id="comment.blocked-user" defaultMessage="Blocked" />
            </span>
          )}

          {isAdmin && !props.isUserBanned && props.data.delete && (
            <span className="comment__status">
              <FormattedMessage id="comment.deleted-user" defaultMessage="Deleted" />
            </span>
          )}

          <span className={b('comment__score', {}, { view: o.score.view })}>
            <span
              className={b(
                'comment__vote',
                {},
                { type: 'up', selected: state.scoreDelta === 1, disabled: isUpvotingDisabled }
              )}
              aria-disabled={state.scoreDelta === 1 || isUpvotingDisabled ? 'true' : 'false'}
              {...getHandleClickProps(isUpvotingDisabled ? undefined : this.increaseScore)}
              title={upvotingDisabledReason || undefined}
            >
              Vote up
            </span>

            <span className="comment__score-value" title={o.controversyText}>
              {o.score.sign}
              {o.score.value}
            </span>

            <span
              className={b(
                'comment__vote',
                {},
                { type: 'down', selected: state.scoreDelta === -1, disabled: isDownvotingDisabled }
              )}
              aria-disabled={state.scoreDelta === -1 || isUpvotingDisabled ? 'true' : 'false'}
              {...getHandleClickProps(isDownvotingDisabled ? undefined : this.decreaseScore)}
              title={downvotingDisabledReason || undefined}
            >
              Vote down
            </span>
          </span>
        </div>
        <div className="comment__body">
          {!!state.voteErrorMessage && (
            <div className="voting__error" role="alert">
              <FormattedMessage
                id="comment.vote-error"
                defaultMessage="Voting error: {voteErrorMessage}"
                values={{ voteErrorMessage: state.voteErrorMessage }}
              />
            </div>
          )}

          {(!props.collapsed || props.view === 'pinned') && (
            <div
              className={b('comment__text', { mix: b('raw-content', {}, { theme: props.theme }) })}
              ref={this.textNode}
              // eslint-disable-next-line react/no-danger
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          )}

          {(!props.collapsed || props.view === 'pinned') && (
            <div className="comment__actions">
              {!props.data.delete && !props.isCommentsDisabled && !props.disabled && props.view === 'main' && (
                <Button kind="link" {...getHandleClickProps(this.toggleReplying)} mix="comment__action">
                  {isReplying ? (
                    <FormattedMessage id="comment.cancel" defaultMessage="Cancel" />
                  ) : (
                    <FormattedMessage id="comment.reply" defaultMessage="Reply" />
                  )}
                </Button>
              )}
              {!props.data.delete &&
                !props.disabled &&
                !!o.orig &&
                isCurrentUser &&
                (editable || isEditing) &&
                props.view === 'main' && [
                  <Button
                    kind="link"
                    {...getHandleClickProps(this.toggleEditing)}
                    mix={['comment__action', 'comment__action_type_edit']}
                  >
                    {isEditing ? (
                      <FormattedMessage id="comment.cancel" defaultMessage="Cancel" />
                    ) : (
                      <FormattedMessage id="comment.edit" defaultMessage="Edit" />
                    )}
                  </Button>,
                  !isAdmin && (
                    <Button
                      kind="link"
                      {...getHandleClickProps(this.deleteComment)}
                      mix={['comment__action', 'comment__action_type_delete']}
                    >
                      <FormattedMessage id="comment.delete" defaultMessage="Delete" />
                    </Button>
                  ),
                  state.editDeadline && (
                    <Countdown
                      className="comment__edit-timer"
                      time={state.editDeadline}
                      onTimePassed={() =>
                        this.setState({
                          editDeadline: null,
                        })
                      }
                    />
                  ),
                ]}

              {commentControls.length > 0 && <span className="comment__controls">{commentControls}</span>}
            </div>
          )}
        </div>

        {CommentForm && isReplying && props.view === 'main' && (
          <CommentForm
            id={o.id}
            intl={this.props.intl}
            user={props.user}
            theme={props.theme}
            mode="reply"
            mix="comment__input"
            onSubmit={(text: string, title: string) => this.addComment(text, title, o.id)}
            onCancel={this.toggleReplying}
            getPreview={this.props.getPreview!}
            autofocus={true}
            uploadImage={uploadImageHandler}
            simpleView={StaticStore.config.simple_view}
          />
        )}

        {CommentForm && isEditing && props.view === 'main' && (
          <CommentForm
            id={o.id}
            intl={this.props.intl}
            user={props.user}
            theme={props.theme}
            value={o.orig}
            mode="edit"
            mix="comment__input"
            onSubmit={(text: string) => this.updateComment(props.data.id, text)}
            onCancel={this.toggleEditing}
            getPreview={this.props.getPreview!}
            errorMessage={state.editDeadline === null ? intl.formatMessage(messages.expiredTime) : undefined}
            autofocus={true}
            uploadImage={uploadImageHandler}
            simpleView={StaticStore.config.simple_view}
          />
        )}
      </article>
    );
  }
}

function getTextSnippet(html: string) {
  const LENGTH = 100;
  const tmp = document.createElement('div');
  tmp.innerHTML = html.replace('</p><p>', ' ');

  const result = tmp.innerText || '';
  const snippet = result.substr(0, LENGTH);

  return snippet.length === LENGTH && result.length !== LENGTH ? `${snippet}...` : snippet;
}

function FormatTime({ time }: { time: Date }) {
  const intl = useIntl();

  return (
    <FormattedMessage
      id="comment.time"
      defaultMessage="{day} at {time}"
      values={{
        day: intl.formatDate(time),
        time: intl.formatTime(time),
      }}
    />
  );
}

export default Comment;
