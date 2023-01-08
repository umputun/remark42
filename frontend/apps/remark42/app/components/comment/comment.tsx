import { h, JSX, Component, createRef, ComponentType } from 'preact';
import { FormattedMessage, IntlShape, defineMessages } from 'react-intl';
import b from 'bem-react-helper';
import clsx from 'clsx';

import { COMMENT_NODE_CLASSNAME_PREFIX } from 'common/constants';

import { StaticStore } from 'common/static-store';
import { debounce } from 'utils/debounce';
import { copy } from 'common/copy';
import { Theme, BlockTTL, Comment as CommentType, PostInfo, User, CommentMode, Profile } from 'common/types';
import { isUserAnonymous } from 'utils/isUserAnonymous';

import { Props as CommentFormProps } from 'components/comment-form';
import { Avatar } from 'components/avatar';
import { VerificationIcon } from 'components/icons/verification';
import { getPreview, uploadImage } from 'common/api';
import { postMessageToParent } from 'utils/post-message';
import { getBlockingDurations } from './getBlockingDurations';
import { boundActions } from './connected-comment';
import { CommentVotes } from './comment-votes';
import { CommentActions } from './comment-actions';

import styles from './comment.module.css';
import './styles';

export type CommentProps = {
  user: User | null;
  CommentForm?: ComponentType<CommentFormProps>;
  data: CommentType;
  repliesCount?: number;
  post_info?: PostInfo;
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
  editDeadline?: number;
  initial: boolean;
}

export class Comment extends Component<CommentProps, State> {
  votingPromise: Promise<unknown> = Promise.resolve();
  /** comment text node. Used in comment text copying */
  textNode = createRef<HTMLDivElement>();

  /**
   * Defines whether current client is admin
   */
  isAdmin = (): boolean => {
    return Boolean(this.props.user?.admin);
  };

  /**
   * Defines whether current client is not logged in
   */
  isGuest = (): boolean => {
    return this.props.user === null;
  };

  /**
   * Defines whether current client is logged in via `Anonymous provider`
   */
  isAnonymous = (): boolean => {
    return isUserAnonymous(this.props.user);
  };

  /**
   * Defines whether comment made by logged-in user
   */
  isCurrentUser = (): boolean => {
    return !this.isGuest() && this.props.data.user.id === this.props.user?.id;
  };

  updateState = (props: CommentProps) => {
    const newState: Partial<State> = {};

    if (props.inView) {
      newState.renderDummy = false;
    }

    // set comment edit timer
    if (this.isCurrentUser()) {
      const editDuration = StaticStore.config.edit_duration;
      const timeDiff = StaticStore.serverClientTimeDiff || 0;
      const editDeadline = new Date(props.data.time).getTime() + timeDiff + editDuration * 1000;

      newState.editDeadline = editDeadline > Date.now() ? editDeadline : undefined;
    }

    return newState;
  };

  state = {
    renderDummy: typeof this.props.inView === 'boolean' ? !this.props.inView : false,
    isCopied: false,
    editDeadline: undefined,
    voteErrorMessage: null,
    initial: true,
    ...this.updateState(this.props),
  };

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
    const profile: Profile = { ...this.props.data.user };

    if (this.props.user?.id === profile.id) {
      profile.current = '1';
    }

    postMessageToParent({ profile });
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
    const promptMessage = intl.formatMessage(value ? messages.verifyUser : messages.unverifyUser, { userName });

    if (window.confirm(promptMessage)) {
      this.props.setVerifiedStatus!(userId, value);
    }
  };

  onBlockUserClick = (evt: Event) => {
    const target = evt.currentTarget;

    if (target instanceof HTMLOptionElement) {
      // we have to debounce the blockUser function calls otherwise it will be
      // called 2 times (by change event and by blur event)
      this.blockUser(target.value as BlockTTL);
    }
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

  unblockUser = () => {
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

  addComment = async (text: string, title: string, pid?: CommentType['id']) => {
    await this.props.addComment!(text, title, pid);

    this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });
  };

  updateComment = async (id: CommentType['id'], text: string) => {
    await this.props.updateComment!(id, text);

    this.props.setReplyEditState!({ id: this.props.data.id, state: CommentMode.None });
  };

  scrollToParent = (evt: Event) => {
    const { pid } = this.props.data;
    const parentCommentNode = document.getElementById(`${COMMENT_NODE_CLASSNAME_PREFIX}${pid}`);

    evt.preventDefault();

    if (!parentCommentNode) {
      return;
    }

    const top = parentCommentNode.getBoundingClientRect().top;

    if (postMessageToParent({ scrollTo: top })) {
      return;
    }

    parentCommentNode.scrollIntoView();
  };

  get isVotesDisabled(): boolean {
    return (
      this.props.view !== 'main' ||
      this.props.post_info?.read_only ||
      this.props.data.delete ||
      this.isCurrentUser() ||
      this.isGuest() ||
      (!StaticStore.config.anon_vote && this.isAnonymous())
    );
  }

  copyComment = async () => {
    const { name } = this.props.data.user;
    const time = getLocalDatetime(this.props.intl, new Date(this.props.data.time));
    const text = this.textNode.current?.textContent ?? '';

    try {
      await copy(`${name} ${time}\n${text}`);
    } catch (e) {
      console.log(e);
    }

    this.setState({ isCopied: true }, () => {
      setTimeout(() => this.setState({ isCopied: false }), 3000);
    });
  };

  render(props: CommentProps, state: State): JSX.Element {
    const isAdmin = this.isAdmin();
    const isGuest = this.isGuest();
    const isCurrentUser = this.isCurrentUser();

    const isReplying = props.editMode === CommentMode.Reply;
    const isEditing = props.editMode === CommentMode.Edit;
    const uploadImageHandler = this.isAnonymous() ? undefined : this.props.uploadImage;
    const intl = props.intl;
    const CommentForm = this.props.CommentForm || null;

    /**
     * CommentType adapted for rendering
     */

    const o = {
      ...props.data,
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
      user: props.data.user,
    };

    const defaultMods = {
      disabled: props.disabled,
      pinned: props.data.pin,
      // TODO: we also have critical_score, so we need to collapse comments with it in future
      useless:
        !!props.isUserBanned ||
        !!props.data.delete ||
        (props.view !== 'preview' &&
          props.data.score < StaticStore.config.low_score &&
          !props.data.pin &&
          !props.disabled),
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
            <div className="comment__avatar">
              <Avatar url={o.user.picture} />
            </div>
          )}

          <div className={styles.user}>
            {props.view !== 'user' && (
              <button onClick={() => this.toggleUserInfoVisibility()} className="comment__username">
                {o.user.name}
              </button>
            )}
            {isAdmin && props.view !== 'user' && (
              <button
                className={styles.verificationButton}
                onClick={this.toggleVerify}
                title={intl.formatMessage(messages.toggleVerification)}
              >
                <VerificationIcon
                  title={intl.formatMessage(o.user.verified ? messages.verifiedUser : messages.unverifiedUser)}
                  className={clsx(styles.verificationIcon, !o.user.verified && styles.verificationIconInactive)}
                />
              </button>
            )}
            {!isAdmin && !!o.user.verified && props.view !== 'user' && (
              <VerificationIcon className={styles.verificationIcon} title={intl.formatMessage(messages.verifiedUser)} />
            )}
            {o.user.paid_sub && (
              <img
                width={12}
                height={12}
                src={require('assets/social/patreon.svg').default}
                alt={intl.formatMessage(messages.paidPatreon)}
              />
            )}
          </div>

          <a href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`} className="comment__time">
            {getLocalDatetime(this.props.intl, o.time)}
          </a>

          {!!props.level && props.level > 0 && props.view === 'main' && (
            <a
              className={styles.threadStarterAnchor}
              href={`${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`}
              title={goToParentMessage}
              onClick={(e) => this.scrollToParent(e)}
            >
              <svg width="7" height="11" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 7 11" aria-hidden>
                <path
                  fill="currentColor"
                  d="M.815 5.905L2.915 4v7H4.08V4l2.105 1.905.815-.74-3.5-3.17L0 5.165l.815.74zM0 1.045h7V0H0v1.045z"
                />
              </svg>
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
          {this.props.view !== 'pinned' && (
            <CommentVotes
              id={this.props.data.id}
              vote={props.data.vote}
              votes={props.data.score}
              controversy={props.data.controversy}
              disabled={this.isVotesDisabled}
            />
          )}
        </div>
        <div className="comment__body">
          {(!props.collapsed || props.view === 'pinned') && (
            <div
              className={b('comment__text', { mix: b('raw-content', {}, { theme: props.theme }) })}
              ref={this.textNode}
              // eslint-disable-next-line react/no-danger
              dangerouslySetInnerHTML={{ __html: o.text }}
            />
          )}

          {(!props.collapsed || !this.props.data.delete) && props.view !== 'pinned' && (
            <CommentActions
              admin={isAdmin}
              pinned={props.data.pin}
              copied={state.isCopied}
              editing={isEditing}
              replying={isReplying}
              editable={props.repliesCount === 0 && state.editDeadline !== undefined}
              editDeadline={state.editDeadline}
              readOnly={props.post_info?.read_only}
              onToggleReplying={this.toggleReplying}
              onDisableEditing={() => this.setState({ editDeadline: undefined })}
              currentUser={isCurrentUser}
              bannedUser={props.isUserBanned}
              onCopy={this.copyComment}
              onTogglePin={this.togglePin}
              onToggleEditing={this.toggleEditing}
              onDelete={this.deleteComment}
              onHideUser={this.hideUser}
              onBlockUser={this.blockUser}
              onUnblockUser={this.unblockUser}
            />
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
            errorMessage={state.editDeadline === undefined ? intl.formatMessage(messages.expiredTime) : undefined}
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

function getLocalDatetime(intl: IntlShape, date: Date) {
  return intl.formatMessage(messages.commentTime, {
    day: intl.formatDate(date),
    time: intl.formatTime(date),
  });
}

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
  commentTime: {
    id: 'comment.time',
    defaultMessage: '{day} at {time}',
  },
  paidPatreon: {
    id: 'comment.paid-patreon',
    defaultMessage: 'Patreon Paid Subscriber',
  },
});
