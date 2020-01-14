/** @jsx createElement */

import './styles';

import { createElement, JSX, Component, createRef } from 'preact';
import b from 'bem-react-helper';

import { getHandleClickProps } from '@app/common/accessibility';
import { API_BASE, BASE_URL, COMMENT_NODE_CLASSNAME_PREFIX, BLOCKING_DURATIONS } from '@app/common/constants';

import { StaticStore } from '@app/common/static_store';
import debounce from '@app/utils/debounce';
import copy from '@app/common/copy';
import { Theme, BlockTTL, Comment as CommentType, PostInfo, User, CommentMode } from '@app/common/types';
import { extractErrorMessageFromResponse, FetcherError } from '@app/utils/errorUtils';
import { isUserAnonymous } from '@app/utils/isUserAnonymous';

import { CommentForm } from '@app/components/comment-form';
import { AvatarIcon } from '@app/components/avatar-icon';
import { Button } from '@app/components/button';
import Countdown from '@app/components/countdown';
import { boundActions } from './connected-comment';
import { getPreview, uploadImage } from '@app/common/api';
import postMessage from '@app/utils/postMessage';

export type Props = {
  user: User | null;
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
} & Partial<typeof boundActions>;

export interface State {
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

export class Comment extends Component<Props, State> {
  votingPromise: Promise<unknown>;
  /** comment text node. Used in comment text copying */
  textNode = createRef<HTMLDivElement>();

  constructor(props: Props) {
    super(props);

    this.state = {
      isCopied: false,
      editDeadline: null,
      voteErrorMessage: null,
      scoreDelta: 0,
      cachedScore: props.data.score,
      initial: true,
      ...this.updateState(props),
    };

    this.votingPromise = Promise.resolve();

    this.toggleEditing = this.toggleEditing.bind(this);
    this.toggleReplying = this.toggleReplying.bind(this);
    this.blockUser = debounce(this.blockUser, 100).bind(this);
  }

  // getHandleClickProps = (handler?: (e: KeyboardEvent | MouseEvent) => void) => {
  //   if (this.state.initial) return null;
  //   if (this.props.inView === false) return null;
  //   return getHandleClickProps(handler);
  // };

  componentWillReceiveProps(nextProps: Props) {
    this.setState(this.updateState(nextProps));
  }

  componentDidMount() {
    this.setState({ initial: false });
  }

  updateState = (props: Props) => {
    const newState: Partial<State> = {
      scoreDelta: props.data.vote,
      cachedScore: props.data.score,
    };

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

  toggleReplying = () => {
    const { editMode } = this.props;
    if (editMode === CommentMode.Reply) {
      this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });
    } else {
      this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.Reply });
    }
  };

  toggleEditing = () => {
    const { editMode } = this.props;
    if (editMode === CommentMode.Edit) {
      this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });
    } else {
      this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.Edit });
    }
  };

  toggleUserInfoVisibility = () => {
    if (window.parent) {
      const { user } = this.props.data;
      const data = JSON.stringify({ isUserInfoShown: true, user });
      window.parent.postMessage(data, '*');
    }
  };

  togglePin = () => {
    const value = !this.props.data.pin;
    const promptMessage = `Do you want to ${value ? 'pin' : 'unpin'} this comment?`;

    if (confirm(promptMessage)) {
      this.props.setPinState!(this.props.data.id, value);
    }
  };

  toggleVerify = () => {
    const value = !this.props.data.user.verified;
    const userId = this.props.data.user.id;
    const promptMessage = `Do you want to ${value ? 'verify' : 'unverify'} ${this.props.data.user.name}?`;

    if (confirm(promptMessage)) {
      this.props.setVerifyStatus!(userId, value);
    }
  };

  onBlockUserClick = (e: Event) => {
    // blur event will be triggered by the confirm pop-up which will start
    // infinite loop of blur -> confirm -> blur -> ...
    // so we trigger the blur event manually and have debounce mechanism to prevent it
    if (e.type === 'change') {
      (e.target as HTMLElement).blur();
    }
    // we have to debounce the blockUser function calls otherwise it will be
    // called 2 times (by change event and by blur event)
    this.blockUser((e.target as HTMLOptionElement).value as BlockTTL);
  };

  blockUser = (ttl: BlockTTL) => {
    const { user } = this.props.data;

    const block_duration = BLOCKING_DURATIONS.find(el => el.value === ttl);
    // blocking duration may be undefined if user hasn't selected anything
    // and ttl equals "Blocking period"
    if (!block_duration) return;

    const duration = block_duration.label;
    if (confirm(`Do you want to block ${user.name} ${duration.toLowerCase()}?`)) {
      this.props.blockUser!(user.id, user.name, ttl);
    }
  };

  onUnblockUserClick = () => {
    const { user } = this.props.data;

    const promptMessage = `Do you want to unblock this user?`;

    if (confirm(promptMessage)) {
      this.props.unblockUser!(user.id);
    }
  };

  deleteComment = () => {
    if (confirm('Do you want to delete this comment?')) {
      this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });

      this.props.removeComment!(this.props.data.id);
    }
  };

  hideUser = () => {
    if (!confirm(`Do you want to hide comments of ${this.props.data.user.name}?`)) return;
    this.props.hideUser!(this.props.data.user);
  };

  handleVoteError = (e: FetcherError, originalScore: number, originalDelta: number) => {
    this.setState({
      scoreDelta: originalDelta,
      cachedScore: originalScore,
      voteErrorMessage: extractErrorMessageFromResponse(e),
    });
  };

  sendVotingRequest = (votingValue: number, originalScore: number, originalDelta: number) => {
    this.votingPromise = this.votingPromise
      .then(() => this.props.putCommentVote!(this.props.data.id, votingValue))
      .catch(e => this.handleVoteError(e, originalScore, originalDelta));
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
    if (!(this.props.view === 'main' || this.props.view === 'pinned')) return "Voting allowed only on post's page";
    if (this.props.post_info!.read_only) return "Can't vote on read-only topics";
    if (this.props.data.delete) return "Can't vote for deleted comment";
    if (this.isCurrentUser()) return "Can't vote for your own comment";
    if (StaticStore.config.positive_score && this.props.data.score < 1) return 'Only positive score allowed';
    if (this.isGuest()) return 'Sign in to vote';
    if (this.isAnonymous() && !StaticStore.config.anon_vote) return "Anonymous users can't vote";
    return null;
  };

  /**
   * returns reason for disabled upvoting
   */
  getUpvoteDisabledReason = (): string | null => {
    if (!(this.props.view === 'main' || this.props.view === 'pinned')) return "Voting allowed only on post's page";
    if (this.props.post_info!.read_only) return "Can't vote on read-only topics";
    if (this.props.data.delete) return "Can't vote for deleted comment";
    if (this.isCurrentUser()) return "Can't vote for your own comment";
    if (this.isGuest()) return 'Sign in to vote';
    if (this.isAnonymous() && !StaticStore.config.anon_vote) return "Anonymous users can't vote";
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
          <span className="comment__control comment__control_view_inactive">Copied!</span>
        ) : (
          <Button kind="link" {...getHandleClickProps(this.copyComment)} mix="comment__control">
            Copy
          </Button>
        )
      );

      controls.push(
        <Button kind="link" {...getHandleClickProps(this.togglePin)} mix="comment__control">
          {this.props.data.pin ? 'Unpin' : 'Pin'}
        </Button>
      );
    }

    if (!isCurrentUser) {
      controls.push(
        <Button kind="link" {...getHandleClickProps(this.hideUser)} mix="comment__control">
          Hide
        </Button>
      );
    }

    if (isAdmin) {
      if (this.props.isUserBanned) {
        controls.push(
          <Button kind="link" {...getHandleClickProps(this.onUnblockUserClick)} mix="comment__control">
            Unblock
          </Button>
        );
      }

      if (this.props.user!.id !== this.props.data.user.id && !this.props.isUserBanned) {
        controls.push(
          <span className="comment__control comment__control_select-label">
            Block
            <select className="comment__control_select" onBlur={this.onBlockUserClick} onChange={this.onBlockUserClick}>
              <option disabled selected value={undefined}>
                {' '}
                Blocking period{' '}
              </option>
              {BLOCKING_DURATIONS.map(block => (
                <option value={block.value}>{block.label}</option>
              ))}
            </select>
          </span>
        );
      }

      if (!this.props.data.delete) {
        controls.push(
          <Button kind="link" {...getHandleClickProps(this.deleteComment)} mix="comment__control">
            Delete
          </Button>
        );
      }
    }
    return controls;
  };

  render(props: Props, state: State) {
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

    /**
     * CommentType adapted for rendering
     */
    const o = {
      ...props.data,
      controversyText: `Controversy: ${(props.data.controversy || 0).toFixed(2)}`,
      text:
        props.view === 'preview'
          ? getTextSnippet(props.data.text)
          : props.data.delete
          ? 'This comment was deleted'
          : props.data.text,
      time: formatTime(new Date(props.data.time)),
      orig: isEditing
        ? props.data.orig &&
          props.data.orig.replace(/&[#A-Za-z0-9]+;/gi, entity => {
            const span = document.createElement('span');
            span.innerHTML = entity;
            return span.innerText;
          })
        : props.data.orig,
      score: {
        value: Math.abs(state.cachedScore),
        sign: !scoreSignEnabled ? '' : state.cachedScore > 0 ? '+' : state.cachedScore < 0 ? '−' : null,
        view: state.cachedScore > 0 ? 'positive' : state.cachedScore < 0 ? 'negative' : null,
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
      theme: props.view === 'preview' ? null : props.theme,
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
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          </div>
        </article>
      );
    }

    if (!props.editMode && this.props.inView === false) {
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
              aria-label="Toggle verification"
              title={o.user.verified ? 'Verified user' : 'Unverified user'}
              className={b('comment__verification', {}, { active: o.user.verified, clickable: true })}
            />
          )}

          {!isAdmin && !!o.user.verified && props.view !== 'user' && (
            <span title="Verified user" className={b('comment__verification', {}, { active: true })} />
          )}

          <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__time">
            {o.time}
          </a>

          {!!props.level && props.level > 0 && props.view === 'main' && (
            <a
              className="comment__link-to-parent"
              href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`}
              aria-label="Go to parent comment"
              title="Go to parent comment"
              onClick={e => this.scrollToParent(e)}
            >
              {' '}
            </a>
          )}

          {props.isUserBanned && props.view !== 'user' && <span className="comment__status">Blocked</span>}

          {isAdmin && !props.isUserBanned && props.data.delete && <span className="comment__status">Deleted</span>}

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
              Voting error: {state.voteErrorMessage}
            </div>
          )}

          {(!props.collapsed || props.view === 'pinned') && (
            <div
              className={b('comment__text', { mix: b('raw-content', {}, { theme: props.theme }) })}
              ref={this.textNode}
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          )}

          {(!props.collapsed || props.view === 'pinned') && (
            <div className="comment__actions">
              {!props.data.delete && !props.isCommentsDisabled && !props.disabled && !isGuest && props.view === 'main' && (
                <Button kind="link" {...getHandleClickProps(this.toggleReplying)} mix="comment__action">
                  {isReplying ? 'Cancel' : 'Reply'}
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
                    {isEditing ? 'Cancel' : 'Edit'}
                  </Button>,
                  !isAdmin && (
                    <Button
                      kind="link"
                      {...getHandleClickProps(this.deleteComment)}
                      mix={['comment__action', 'comment__action_type_delete']}
                    >
                      Delete
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

        {isReplying && props.view === 'main' && (
          <CommentForm
            user={props.user}
            theme={props.theme}
            value=""
            mode="reply"
            mix="comment__input"
            onSubmit={(text, title) => this.addComment(text, title, o.id)}
            onCancel={this.toggleReplying}
            getPreview={this.props.getPreview!}
            autofocus={true}
            uploadImage={uploadImageHandler}
            simpleView={StaticStore.config.simple_view}
          />
        )}

        {isEditing && props.view === 'main' && (
          <CommentForm
            user={props.user}
            theme={props.theme}
            value={o.orig}
            mode="edit"
            mix="comment__input"
            onSubmit={(text, _title) => this.updateComment(props.data.id, text)}
            onCancel={this.toggleEditing}
            getPreview={this.props.getPreview!}
            errorMessage={state.editDeadline === null ? 'Editing time has expired.' : undefined}
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

function formatTime(time: Date) {
  // 'ru-RU' adds a dot as a separator
  const date = time.toLocaleDateString(['ru-RU'], { day: '2-digit', month: '2-digit', year: '2-digit' });

  // do it manually because Intl API doesn't add leading zeros to hours; idk why
  const hours = `0${time.getHours()}`.slice(-2);
  const mins = `0${time.getMinutes()}`.slice(-2);

  return `${date} at ${hours}:${mins}`;
}
