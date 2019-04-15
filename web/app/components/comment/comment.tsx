/** @jsx h */

import './styles';

import { h, Component, RenderableProps } from 'preact';
import b from 'bem-react-helper';

import { getHandleClickProps } from '@app/common/accessibility';
import { API_BASE, BASE_URL, COMMENT_NODE_CLASSNAME_PREFIX, BLOCKING_DURATIONS } from '@app/common/constants';

import { StaticStore } from '@app/common/static_store';
import debounce from '@app/utils/debounce';
import copy from '@app/common/copy';
import { Theme, BlockTTL, Comment as CommentType, PostInfo, User, CommentMode, Image } from '@app/common/types';
import { extractErrorMessageFromResponse, FetcherError } from '@app/utils/errorUtils';
import { isUserAnonymous } from '@app/utils/isUserAnonymous';

import { Input } from '@app/components/input';
import { AvatarIcon } from '@app/components/avatar-icon';
import Countdown from '../countdown';

export interface Props {
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
  level?: number;
  mix?: string;

  // actions are optional, as component has read-only mode, such as in last comments
  addComment?: (text: string, title: string, pid?: CommentType['id']) => Promise<void>;
  updateComment?: (id: CommentType['id'], text: string) => Promise<void>;
  removeComment?(id: CommentType['id']): Promise<void>;
  setReplyEditState?(id: CommentType['id'], mode: CommentMode): void;
  getPreview?: (text: string) => Promise<string>;
  putCommentVote?(id: CommentType['id'], value: number): Promise<void>;
  collapseToggle?: (id: CommentType['id']) => void;
  setPinState?(id: CommentType['id'], value: boolean): Promise<void>;
  blockUser?(id: User['id'], name: User['name'], ttl: BlockTTL): Promise<void>;
  unblockUser?(id: User['id']): Promise<void>;
  setVerifyStatus?(id: User['id'], value: boolean): Promise<void>;
  uploadImage?(image: File): Promise<Image>;
}

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
}

export class Comment extends Component<Props, State> {
  votingPromise: Promise<unknown>;
  /** comment text node. Used in comment text copying */
  textNode?: HTMLDivElement;

  constructor(props: Props) {
    super(props);

    this.state = {
      isCopied: false,
      editDeadline: null,
      voteErrorMessage: null,
      scoreDelta: 0,
      cachedScore: props.data.score,
    };

    this.votingPromise = Promise.resolve();

    this.updateState(props);

    this.toggleEditing = this.toggleEditing.bind(this);
    this.toggleReplying = this.toggleReplying.bind(this);
    this.blockUser = debounce(this.blockUser, 100).bind(this);
  }

  componentWillReceiveProps(nextProps: Props) {
    this.updateState(nextProps);
  }

  updateState(props: Props) {
    this.setState({
      scoreDelta: props.data.vote,
      cachedScore: props.data.score,
    });

    if (props.user) {
      const userId = props.user!.id;

      // set comment edit timer
      if (userId === props.data.user.id) {
        const editDuration = StaticStore.config.edit_duration;
        const timeDiff = StaticStore.serverClientTimeDiff || 0;
        let editDeadline: Date | null = new Date(new Date(props.data.time).getTime() + timeDiff + editDuration * 1000);
        if (editDeadline < new Date()) editDeadline = null;
        this.setState({
          editDeadline,
        });
      }
    }
  }

  toggleReplying() {
    const { editMode } = this.props;
    if (editMode === CommentMode.Reply) {
      this.props.setReplyEditState!(this.props.data.id, CommentMode.None);
    } else {
      this.props.setReplyEditState!(this.props.data.id, CommentMode.Reply);
    }
  }

  toggleEditing() {
    const { editMode } = this.props;
    if (editMode === CommentMode.Edit) {
      this.props.setReplyEditState!(this.props.data.id, CommentMode.None);
    } else {
      this.props.setReplyEditState!(this.props.data.id, CommentMode.Edit);
    }
  }

  toggleUserInfoVisibility() {
    if (window.parent) {
      const { user } = this.props.data;
      const data = JSON.stringify({ isUserInfoShown: true, user });
      window.parent.postMessage(data, '*');
    }
  }

  setPin(value: boolean) {
    const promptMessage = `Do you want to ${value ? 'pin' : 'unpin'} this comment?`;

    if (confirm(promptMessage)) {
      this.props.setPinState!(this.props.data.id, value);
    }
  }

  setVerify(value: boolean) {
    const userId = this.props.data.user.id;
    const promptMessage = `Do you want to ${value ? 'verify' : 'unverify'} this user?`;

    if (confirm(promptMessage)) {
      this.props.setVerifyStatus!(userId, value);
    }
  }

  onBlockUserClick(e: Event) {
    // blur event will be triggered by the confirm pop-up which will start
    // infinite loop of blur -> confirm -> blur -> ...
    // so we trigger the blur event manually and have debounce mechanism to prevent it
    if (e.type === 'change') {
      (e.target as HTMLElement).blur();
    }
    // we have to debounce the blockUser function calls otherwise it will be
    // called 2 times (by change event and by blur event)
    this.blockUser((e.target as HTMLOptionElement).value as BlockTTL);
  }

  blockUser(ttl: BlockTTL) {
    const { user } = this.props.data;

    const block_duration = BLOCKING_DURATIONS.find(el => el.value === ttl);
    // blocking duration may be undefined if user hasn't selected anything
    // and ttl equals "Blocking period"
    if (!block_duration) return;

    const duration = block_duration.label;
    if (confirm(`Do you want to block ${user.name} ${duration.toLowerCase()}?`)) {
      this.props.blockUser!(user.id, user.name, ttl);
    }
  }

  onUnblockUserClick() {
    const { user } = this.props.data;

    const promptMessage = `Do you want to unblock this user?`;

    if (confirm(promptMessage)) {
      this.props.unblockUser!(user.id);
    }
  }

  deleteComment() {
    if (confirm('Do you want to delete this comment?')) {
      this.props.setReplyEditState!(this.props.data.id, CommentMode.None);

      this.props.removeComment!(this.props.data.id);
    }
  }

  handleVoteError(e: FetcherError, originalScore: number, originalDelta: number) {
    this.setState({
      scoreDelta: originalDelta,
      cachedScore: originalScore,
      voteErrorMessage: extractErrorMessageFromResponse(e),
    });
  }

  sendVotingRequest(votingValue: number, originalScore: number, originalDelta: number) {
    this.votingPromise = this.votingPromise
      .then(() => this.props.putCommentVote!(this.props.data.id, votingValue))
      .catch(e => this.handleVoteError(e, originalScore, originalDelta));
  }

  increaseScore() {
    const { cachedScore, scoreDelta } = this.state;

    if (scoreDelta === 1) return;

    this.setState({
      scoreDelta: scoreDelta + 1,
      cachedScore: cachedScore + 1,
      voteErrorMessage: null,
    });

    this.sendVotingRequest(1, cachedScore, scoreDelta);
  }

  decreaseScore() {
    const { cachedScore, scoreDelta } = this.state;

    if (scoreDelta === -1) return;

    this.setState({
      scoreDelta: scoreDelta - 1,
      cachedScore: cachedScore - 1,
      voteErrorMessage: null,
    });

    this.sendVotingRequest(-1, cachedScore, scoreDelta);
  }

  async addComment(text: string, title: string, pid?: CommentType['id']) {
    await this.props.addComment!(text, title, pid);

    this.props.setReplyEditState!(this.props.data.id, CommentMode.None);
  }

  async updateComment(id: CommentType['id'], text: string) {
    await this.props.updateComment!(id, text);

    this.props.setReplyEditState!(this.props.data.id, CommentMode.None);
  }

  scrollToParent(e: Event) {
    const {
      data: { pid },
    } = this.props;

    e.preventDefault();

    const parentCommentNode = document.getElementById(`${COMMENT_NODE_CLASSNAME_PREFIX}${pid}`);

    if (parentCommentNode) {
      parentCommentNode.scrollIntoView();
    }
  }

  toggleCollapse() {
    this.props.setReplyEditState!(this.props.data.id, CommentMode.None);

    this.props.collapseToggle!(this.props.data.id);
  }

  copyComment({ username, time }: { username: string; time: string }) {
    const text = this.textNode!.textContent || '';

    copy(`<b>${username}</b>&nbsp;${time}<br>${text.replace(/\n+/g, '<br>')}`);

    this.setState({ isCopied: true }, () => {
      setTimeout(() => this.setState({ isCopied: false }), 3000);
    });
  }

  /**
   * Defines whether current client is admin
   */
  isAdmin(): boolean {
    return !!this.props.user && this.props.user.admin;
  }

  /**
   * Defines whether current client is not logged in
   */
  isGuest(): boolean {
    return !this.props.user;
  }

  /**
   * Defines whether current client is logged in via `Anonymous provider`
   */
  isAnonymous(): boolean {
    return isUserAnonymous(this.props.user);
  }

  /**
   * Defines whether comment made by logged in user
   */
  isCurrentUser(): boolean {
    if (this.isGuest()) return false;

    return this.props.data.user.id === this.props.user!.id;
  }

  /**
   * returns reason for disabled downvoting
   */
  getDownvoteDisabledReason(): string | null {
    if (!(this.props.view === 'main' || this.props.view === 'pinned')) return "Voting allowed only on post's page";
    if (this.props.post_info!.read_only) return "Can't vote on read-only topics";
    if (this.props.data.delete) return "Can't vote for deleted comment";
    if (this.isCurrentUser()) return "Can't vote for your own comment";
    if (StaticStore.config.positive_score && this.props.data.score < 1) return 'Only positive score allowed';
    if (this.isGuest()) return 'Sign in to vote';
    if (this.isAnonymous()) return "Anonymous users can't vote";
    return null;
  }

  /**
   * returns reason for disabled upvoting
   */
  getUpvoteDisabledReason(): string | null {
    if (!(this.props.view === 'main' || this.props.view === 'pinned')) return "Voting allowed only on post's page";
    if (this.props.post_info!.read_only) return "Can't vote on read-only topics";
    if (this.props.data.delete) return "Can't vote for deleted comment";
    if (this.isCurrentUser()) return "Can't vote for your own comment";
    if (this.isGuest()) return 'Sign in to vote';
    if (this.isAnonymous()) return "Anonymous users can't vote";
    return null;
  }

  render(props: RenderableProps<Props>, state: State) {
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

    /**
     * CommentType adapted for rendering
     */
    const o = {
      ...props.data,
      controversyText: `Controversy: ${(props.data.controversy || 0).toFixed(2)}`,
      text: props.data.text.length
        ? props.view === 'preview'
          ? getTextSnippet(props.data.text)
          : props.data.text
        : this.props.isUserBanned
        ? 'This user was blocked'
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
        <div className="comment__body">
          <div className="comment__info">
            {props.view !== 'user' && <AvatarIcon theme={this.props.theme} picture={o.user.picture} />}

            {props.view !== 'user' && (
              <span
                {...getHandleClickProps(() => this.toggleUserInfoVisibility())}
                className="comment__username"
                title={o.user.id}
              >
                {o.user.name}
              </span>
            )}

            {isAdmin && props.view !== 'user' && (
              <span
                {...getHandleClickProps(() => this.setVerify(!o.user.verified))}
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

            {isAdmin && props.isUserBanned && props.view !== 'user' && <span className="comment__status">Blocked</span>}

            {isAdmin && !props.isUserBanned && props.data.delete && <span className="comment__status">Deleted</span>}

            {!props.disabled && props.view === 'main' && (
              <span
                {...getHandleClickProps(() => this.toggleCollapse())}
                className={b('comment__action', {}, { type: 'collapse', selected: props.collapsed })}
              >
                {props.collapsed ? '+' : '−'}
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
                {...getHandleClickProps(isUpvotingDisabled ? undefined : () => this.increaseScore())}
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
                {...getHandleClickProps(isDownvotingDisabled ? undefined : () => this.decreaseScore())}
                title={downvotingDisabledReason || undefined}
              >
                Vote down
              </span>
            </span>
          </div>

          {!!state.voteErrorMessage && (
            <div className="voting__error" role="alert">
              Voting error: {state.voteErrorMessage}
            </div>
          )}

          {(!props.collapsed || props.view === 'pinned') && (
            <div
              className={b('comment__text', { mix: b('raw-content', {}, { theme: props.theme }) })}
              ref={r => (this.textNode = r)}
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          )}

          {(!props.collapsed || props.view === 'pinned') && (
            <div className="comment__actions">
              {!props.data.delete && !props.isCommentsDisabled && !props.disabled && !isGuest && props.view === 'main' && (
                <span {...getHandleClickProps(() => this.toggleReplying())} className="comment__action">
                  {isReplying ? 'Cancel' : 'Reply'}
                </span>
              )}

              {!props.data.delete &&
                !props.disabled &&
                !!o.orig &&
                isCurrentUser &&
                (editable || isEditing) &&
                props.view === 'main' && [
                  <span
                    {...getHandleClickProps(() => this.toggleEditing())}
                    className="comment__action comment__action_type_edit"
                  >
                    {isEditing ? 'Cancel' : 'Edit'}
                  </span>,
                  !isAdmin && (
                    <span
                      {...getHandleClickProps(() => this.deleteComment())}
                      className="comment__action comment__action_type_delete"
                    >
                      Delete
                    </span>
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

              {!props.data.delete && isAdmin && (
                <span className="comment__controls">
                  {!state.isCopied && (
                    <span
                      {...getHandleClickProps(() => this.copyComment({ username: o.user.name, time: o.time }))}
                      className="comment__control"
                    >
                      Copy
                    </span>
                  )}

                  {state.isCopied && <span className="comment__control comment__control_view_inactive">Copied!</span>}

                  {(props.view === 'main' || props.view === 'pinned') && (
                    <span {...getHandleClickProps(() => this.setPin(!props.data.pin))} className="comment__control">
                      {props.data.pin ? 'Unpin' : 'Pin'}
                    </span>
                  )}

                  {props.isUserBanned && (
                    <span {...getHandleClickProps(() => this.onUnblockUserClick())} className="comment__control">
                      Unblock
                    </span>
                  )}

                  {props.user!.id !== props.data.user.id && !props.isUserBanned && (
                    <span className="comment__control comment__control_select-label">
                      Block
                      <select
                        className="comment__control_select"
                        onBlur={e => this.onBlockUserClick(e)}
                        onChange={e => this.onBlockUserClick(e)}
                      >
                        <option disabled selected value={undefined}>
                          {' '}
                          Blocking period{' '}
                        </option>
                        {BLOCKING_DURATIONS.map(block => (
                          <option value={block.value}>{block.label}</option>
                        ))}
                      </select>
                    </span>
                  )}

                  {!props.data.delete && (
                    <span {...getHandleClickProps(() => this.deleteComment())} className="comment__control">
                      Delete
                    </span>
                  )}
                </span>
              )}
            </div>
          )}
        </div>

        {isReplying && props.view === 'main' && (
          <Input
            theme={props.theme}
            value=""
            mode="reply"
            mix="comment__input"
            onSubmit={(text, title) => this.addComment(text, title, o.id)}
            onCancel={this.toggleReplying}
            getPreview={this.props.getPreview!}
            autofocus={true}
            uploadImage={uploadImageHandler}
          />
        )}

        {isEditing && props.view === 'main' && (
          <Input
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
