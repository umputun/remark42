// A comment in plain JavaScript: its author, time, votes, text and actions, and the reply or edit
// form under it. mountComment fills an article the caller owns and keeps it current: update()
// changes the nodes in place, so an open form keeps its text, focus and caret while the comment
// redraws, and destroy() stops the comment's timers and its parts.

import { classes, placeChildren, svgElement, SVG_NS } from '../../common/dom.js';
import { defineMessages } from '../../common/intl-message.ts';
import { COMMENT_NODE_CLASSNAME_PREFIX } from '../../common/constants.config.ts';
import { webAssetUrl } from '../../common/web-asset.ts';
import { StaticStore } from '../../common/static-store.ts';
import { copy } from '../../common/copy.ts';
import { CommentMode } from '../../common/types.ts';
import { isUserAnonymous } from '../../utils/isUserAnonymous.ts';
import { postMessageToParent } from '../../utils/post-message.ts';
import { issueReaderToken } from '../../utils/reader.ts';
import { avatarElement } from '../avatar/avatar-element.js';
import { verificationIcon } from '../icons/verification-icon.js';
import { mountCommentForm } from '../comment-form/comment-form-element.js';
import { getBlockingDurations } from './getBlockingDurations.ts';
import { mountVotes } from './votes.js';
import { mountActions } from './actions.js';
import styles from './comment.module.css';
import './raw-content.css';

// how long Copy shows that the text was copied, and how long a pick of a blocking period waits
// for the select to settle before it asks
const COPIED_MS = 3000;
const BLOCK_WAIT_MS = 100;
// a delay past 2^31-1 ms fires at once, so a longer edit window is waited out in steps
const LONGEST_TIMEOUT = 2 ** 31 - 1;

/** @typedef {import('../../common/types.ts').Comment} CommentData */
/** @typedef {import('../../common/types.ts').User} User */
/** @typedef {import('../../common/types.ts').BlockTTL} BlockTTL */
/** @typedef {import('../../common/intl-message.ts').IntlShape} IntlShape */

/**
 * @typedef {object} CommentProps
 * @property {CommentData} data
 * @property {User | null} user  the reader
 * @property {number} [repliesCount]
 * @property {import('../../common/types.ts').PostInfo} [post_info]
 * @property {boolean} [isUserBanned]  the author is blocked
 * @property {CommentMode} [editMode]  whether the reply or the edit form is open
 * @property {'main' | 'pinned' | 'user'} view  the thread, the pinned block, or a profile
 * @property {boolean} [disabled]  a copy shown outside the thread, with no id of its own
 * @property {boolean} [collapsed]
 * @property {import('../../common/types.ts').Theme} theme
 * @property {boolean} [inView]  false draws an empty placeholder of the comment's size
 * @property {number} [level]
 * @property {string} [mix]
 */

/**
 * @typedef {object} CommentActionsDeps  what the reader's actions change; each may be missing
 * @property {(text: string, title: string, pid?: string) => Promise<unknown>} [addComment]
 * @property {(id: string, text: string) => Promise<unknown>} [updateComment]
 * @property {(id: string) => unknown} [removeComment]
 * @property {(state: { id: string, state: CommentMode }) => unknown} [setReplyEditState]
 * @property {(id: string, value: boolean) => unknown} [setPinState]
 * @property {(id: string, name: string, ttl: BlockTTL) => unknown} [blockUser]
 * @property {(id: string) => unknown} [unblockUser]
 * @property {(user: User) => unknown} [hideUser]
 * @property {(id: string, value: boolean) => unknown} [setVerifiedStatus]
 */

/**
 * @typedef {object} CommentFormBase  what every form under a comment shares
 * @property {() => string} nextTextareaId  the id the field of a form that appears takes
 * @property {() => Promise<Record<string, string>>} loadEmoji
 * @property {import('../../common/types.ts').Theme} theme  the widget's theme
 * @property {import('../auth/sign-in.js').SignInDeps} signIn
 * @property {IntlShape} rssIntl
 * @property {import('../comment-form/comment-form-element.js').CommentFormDeps['email']} email
 * @property {import('../comment-form/comment-form-element.js').CommentFormDeps['telegram']} telegram
 */

/**
 * @typedef {object} CommentDeps
 * @property {IntlShape} intl  the comment's own text, its questions and its forms
 * @property {IntlShape} partsIntl  the actions under it, which take the widget's locale
 * @property {import('./votes.js').VotesDeps} votes
 * @property {CommentActionsDeps} actions
 * @property {(text: string) => Promise<string>} [getPreview]
 * @property {(image: File) => Promise<import('../../common/types.ts').Image>} [uploadImage]
 * @property {CommentFormBase} [form]  none means the comment draws no reply or edit form
 */

/**
 * @param {HTMLElement} root  an empty article; its attributes and children are this component's
 * @param {CommentProps} initial
 * @param {CommentDeps} initialDeps
 */
export function mountComment(root, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  let destroyed = false;
  let mounted = false;
  let rendering = false;
  let again = false;

  // what the preact component kept as its state
  let renderDummy = typeof props.inView === 'boolean' ? !props.inView : false;
  let isCopied = false;
  /** @type {number | undefined} */
  let editDeadline = deadlineFor(props);
  /** @type {number | undefined} */
  let watchedDeadline;
  /** the class last written on the article, which a redraw writes again only when it changes */
  /** @type {string | undefined} */
  let drawnClass;

  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let deadlineTimer;
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let copiedTimer;
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let blockTimer;

  const title = document.createElement('div');
  title.className = styles.title;
  const titleLink = document.createElement('a');
  titleLink.className = styles.titleLink;
  title.append(titleLink);

  const info = document.createElement('div');
  info.className = styles.info;
  info.setAttribute('data-nosnippet', 'true');

  const avatarBox = document.createElement('div');
  avatarBox.className = styles.avatar;
  // made when it is first shown: an image loads its source even while it is off the page, and a
  // comment out of view, collapsed or in a profile shows none
  /** @type {ReturnType<typeof avatarElement> | undefined} */
  let avatar;

  const user = document.createElement('div');
  user.className = styles.user;
  const username = document.createElement('button');
  username.className = styles.username;
  username.addEventListener('click', () => toggleUserInfoVisibility());

  const verificationButton = document.createElement('button');
  verificationButton.className = styles.verificationButton;
  verificationButton.addEventListener('click', () => toggleVerify());
  // the admin's badge takes its title before its class, the reader's its class before its title,
  // as the preact component passed them
  const adminBadge = verificationIcon(12, { title: '', class: '' });
  verificationButton.append(adminBadge);
  const readerBadge = verificationIcon(12, { class: styles.verificationIcon, title: '' });

  // its source is set when it is first shown, for the reason the avatar is made then
  const patreon = document.createElement('img');
  patreon.width = 12;
  patreon.height = 12;

  const time = document.createElement('a');
  time.href = '';
  time.className = styles.time;
  const timeText = document.createElement('time');
  time.append(timeText);

  const parentLink = document.createElement('a');
  parentLink.className = styles.threadStarterAnchor;
  parentLink.href = '';
  parentLink.title = '';
  parentLink.addEventListener('click', (evt) => scrollToParent(evt));
  parentLink.append(
    svgElement('svg', { width: '7', height: '11', xmlns: SVG_NS, viewBox: '0 0 7 11', 'aria-hidden': 'true' }, [
      svgElement('path', {
        fill: 'currentColor',
        d: 'M.815 5.905L2.915 4v7H4.08V4l2.105 1.905.815-.74-3.5-3.17L0 5.165l.815.74zM0 1.045h7V0H0v1.045z',
      }),
    ])
  );

  const blocked = document.createElement('span');
  blocked.className = styles.status;
  const deleted = document.createElement('span');
  deleted.className = styles.status;

  const body = document.createElement('div');
  body.className = styles.body;
  const text = document.createElement('div');
  text.className = classes(styles.text, 'raw-content');
  text.dir = 'auto';
  /** @type {string | undefined} */
  let textHtml;

  /** @type {{ root: HTMLSpanElement, votes: ReturnType<typeof mountVotes> } | undefined} */
  let votes;
  /** @type {{ root: HTMLDivElement, actions: ReturnType<typeof mountActions> } | undefined} */
  let actions;
  /**
   * @type {{
   *   mode: CommentMode,
   *   root: HTMLDivElement,
   *   textareaId: string,
   *   form?: ReturnType<typeof mountCommentForm>,
   * } | undefined}
   */
  let form;

  /** @param {CommentProps} p */
  function deadlineFor(p) {
    // the reader is taken from the props the deadline is computed for: a comment stays mounted
    // while the reader signs in or out, and that is exactly when the two differ. Whoever does not
    // own the comment has no deadline
    const isOwn = p.user !== null && p.data.user.id === p.user?.id;
    if (!isOwn) {
      return undefined;
    }
    if (p.user?.admin && StaticStore.config.admin_edit) {
      return Infinity;
    }
    const editDuration = StaticStore.config.edit_duration;
    const timeDiff = StaticStore.serverClientTimeDiffMs || 0;
    const deadline = new Date(p.data.time).getTime() + timeDiff + editDuration * 1000;
    return deadline > Date.now() ? deadline : undefined;
  }

  const isAdmin = () => Boolean(props.user?.admin);
  const isGuest = () => props.user === null;
  const isAnonymous = () => isUserAnonymous(props.user);
  const isCurrentUser = () => !isGuest() && props.data.user.id === props.user?.id;
  const say = (/** @type {keyof typeof messages} */ name, /** @type {Record<string, string>} */ values = {}) =>
    /** @type {string} */ (deps.intl.formatMessage(messages[name], values));

  function avatarProps() {
    const { user: author } = props.data;
    return { url: author.picture, size: 24, identicon: isUserAnonymous(author) };
  }

  function isVotesDisabled() {
    return Boolean(
      props.view !== 'main' ||
      props.post_info?.read_only ||
      props.data.delete ||
      isCurrentUser() ||
      isGuest() ||
      (!StaticStore.config.anon_vote && isAnonymous())
    );
  }

  function setReplyEditState(/** @type {CommentMode} */ state) {
    deps.actions.setReplyEditState?.({ id: props.data.id, state });
  }

  function toggleReplying() {
    setReplyEditState(props.editMode === CommentMode.Reply ? CommentMode.None : CommentMode.Reply);
  }

  function toggleEditing() {
    setReplyEditState(props.editMode === CommentMode.Edit ? CommentMode.None : CommentMode.Edit);
  }

  function toggleUserInfoVisibility() {
    // the page embedding the widget receives this message, so it carries the fields the profile
    // shows and nothing else of the record, which for an admin holds the commenter's IP hash. Nor
    // does it say whether the author is the reader: the profile frame asks this frame that itself
    const { id, name, picture } = props.data.user;
    postMessageToParent({ profile: { id, name, picture, token: issueReaderToken() } });
  }

  function togglePin() {
    const value = !props.data.pin;
    if (window.confirm(say(value ? 'pinComment' : 'unpinComment'))) {
      deps.actions.setPinState?.(props.data.id, value);
    }
  }

  function toggleVerify() {
    const value = !props.data.user.verified;
    const { id, name } = props.data.user;
    if (window.confirm(say(value ? 'verifyUser' : 'unverifyUser', { userName: name }))) {
      deps.actions.setVerifiedStatus?.(id, value);
    }
  }

  /** @param {BlockTTL} ttl */
  function blockUser(ttl) {
    // the select reports a pick at once, and the question waits for it to settle
    clearTimeout(blockTimer);
    blockTimer = setTimeout(() => {
      const { user: author } = props.data;
      const blockDuration = getBlockingDurations(deps.intl).find((el) => el.value === ttl);
      // the placeholder option has no duration
      if (!blockDuration) {
        return;
      }
      // only the first letter is lowered, so the label reads mid-sentence while languages that
      // capitalise nouns, such as German, keep them capitalised
      const duration = blockDuration.label;
      const question = say('blockUser', {
        userName: author.name,
        duration: duration.charAt(0).toLowerCase() + duration.slice(1),
      });
      if (window.confirm(question)) {
        deps.actions.blockUser?.(author.id, author.name, ttl);
      }
    }, BLOCK_WAIT_MS);
  }

  function unblockUser() {
    const { user: author } = props.data;
    if (window.confirm(say('unblockUser', { userName: author.name }))) {
      deps.actions.unblockUser?.(author.id);
    }
  }

  function deleteComment() {
    if (window.confirm(say('deleteMessage'))) {
      setReplyEditState(CommentMode.None);
      deps.actions.removeComment?.(props.data.id);
    }
  }

  function hideUser() {
    if (!window.confirm(say('hideUserComments', { userName: props.data.user.name }))) {
      return;
    }
    deps.actions.hideUser?.(props.data.user);
  }

  /** @param {Event} evt */
  function scrollToParent(evt) {
    const parentNode = document.getElementById(`${COMMENT_NODE_CLASSNAME_PREFIX}${props.data.pid}`);
    evt.preventDefault();
    if (!parentNode) {
      return;
    }
    const top = parentNode.getBoundingClientRect().top;
    if (postMessageToParent({ scrollTo: top })) {
      return;
    }
    parentNode.scrollIntoView();
  }

  async function copyComment() {
    const { name } = props.data.user;
    const when = localDatetime(deps.intl, new Date(props.data.time));
    // the text as shown, which a collapsed comment does not show
    const content = text.isConnected ? (text.textContent ?? '') : '';
    try {
      await copy(`${name} ${when}\n${content}`);
    } catch (e) {
      console.log(e);
    }
    if (destroyed) {
      return;
    }
    isCopied = true;
    render();
    clearTimeout(copiedTimer);
    copiedTimer = setTimeout(() => {
      isCopied = false;
      render();
    }, COPIED_MS);
  }

  /** clears the deadline when it passes, whether or not the countdown next to Edit is shown */
  function watchEditDeadline() {
    clearTimeout(deadlineTimer);
    deadlineTimer = undefined;
    const deadline = editDeadline;
    if (deadline === undefined || !Number.isFinite(deadline)) {
      return;
    }
    deadlineTimer = setTimeout(
      () => {
        deadlineTimer = undefined;
        if (Date.now() < deadline) {
          watchEditDeadline();
          return;
        }
        editDeadline = undefined;
        render();
      },
      Math.min(deadline - Date.now(), LONGEST_TIMEOUT)
    );
  }

  /** @param {CommentMode} mode */
  function formProps(mode) {
    const o = props.data;
    /** @type {import('../comment-form/comment-form-element.js').CommentFormProps} */
    const p =
      mode === CommentMode.Reply
        ? {
            id: o.id,
            textareaId: '',
            user: props.user,
            theme: props.theme,
            mode: 'reply',
            mix: styles.input,
            autofocus: true,
          }
        : {
            id: o.id,
            textareaId: '',
            user: props.user,
            theme: props.theme,
            value: o.orig,
            mode: 'edit',
            isReply: Boolean(o.pid),
            mix: styles.input,
            errorMessage: editDeadline === undefined ? say('expiredTime') : undefined,
            autofocus: true,
          };
    return p;
  }

  /**
   * @param {CommentMode} mode
   * @param {CommentFormBase} base
   * @returns {import('../comment-form/comment-form-element.js').CommentFormDeps}
   */
  function formDeps(mode, base) {
    const uploadImage = isAnonymous() ? undefined : deps.uploadImage;
    return {
      intl: deps.intl,
      onSubmit:
        mode === CommentMode.Reply
          ? async (value, pageTitle) => {
              await deps.actions.addComment?.(value, pageTitle, props.data.id);
              setReplyEditState(CommentMode.None);
            }
          : async (value) => {
              await deps.actions.updateComment?.(props.data.id, value);
              setReplyEditState(CommentMode.None);
            },
      getPreview: (value) => /** @type {NonNullable<CommentDeps['getPreview']>} */ (deps.getPreview)(value),
      onCancel: mode === CommentMode.Reply ? toggleReplying : toggleEditing,
      uploadImage: uploadImage && ((image) => uploadImage(image)),
      loadEmoji: base.loadEmoji,
      theme: base.theme,
      signIn: base.signIn,
      rss: { props: { userId: props.user?.id ?? null, theme: base.theme }, intl: base.rssIntl },
      email: base.email,
      telegram: base.telegram,
    };
  }

  /** the form under the comment, or none, as the mode and view ask */
  function placeForm() {
    const mode = props.editMode ?? CommentMode.None;
    const wanted = deps.form && props.view === 'main' && mode !== CommentMode.None ? mode : CommentMode.None;
    if (form && form.mode !== wanted) {
      form.form?.destroy();
      form = undefined;
    }
    if (wanted === CommentMode.None || !deps.form) {
      return;
    }
    if (form) {
      form.form?.update({ ...formProps(wanted), textareaId: form.textareaId }, formDeps(wanted, deps.form));
      return;
    }
    // mounted by mountPlacedForm once its element is on the page: the form sizes its field to the
    // text when it mounts, and a detached field has no height to measure. The field's id is taken
    // once, when the form appears, as the preact form took it
    form = { mode: wanted, root: document.createElement('div'), textareaId: deps.form.nextTextareaId() };
  }

  /** mounts a form placeForm made, now that its element is on the page */
  function mountPlacedForm() {
    if (!form || form.form || !deps.form) {
      return;
    }
    form.form = mountCommentForm(
      form.root,
      { ...formProps(form.mode), textareaId: form.textareaId },
      formDeps(form.mode, deps.form)
    );
  }

  function placeVotes() {
    if (props.view === 'pinned') {
      votes?.votes.destroy();
      votes = undefined;
      return;
    }
    const votesProps = {
      id: props.data.id,
      vote: props.data.vote,
      votes: props.data.score,
      controversy: props.data.controversy,
      disabled: isVotesDisabled(),
    };
    if (votes) {
      votes.votes.update(votesProps, deps.votes);
      return;
    }
    const votesRoot = document.createElement('span');
    votes = { root: votesRoot, votes: mountVotes(votesRoot, votesProps, deps.votes) };
  }

  function placeActions() {
    if (!((!props.collapsed || !props.data.delete) && props.view !== 'pinned')) {
      actions?.actions.destroy();
      actions = undefined;
      return;
    }
    const admin = isAdmin();
    const editing = props.editMode === CommentMode.Edit;
    /** @type {import('./actions.js').ActionsProps} */
    const actionsProps = {
      admin,
      pinned: props.data.pin,
      copied: isCopied,
      editing,
      replying: props.editMode === CommentMode.Reply,
      editable:
        editDeadline !== undefined &&
        (props.repliesCount === 0 || (admin && Boolean(StaticStore.config.admin_edit))) &&
        // the server refuses a commenter's edit on a read-only post; an admin's still goes through.
        // An editor already open keeps its Cancel
        (!props.post_info?.read_only || admin || editing),
      editDeadline,
      readOnly: props.post_info?.read_only,
      onToggleReplying: toggleReplying,
      onDisableEditing: () => {
        editDeadline = undefined;
        render();
      },
      currentUser: isCurrentUser(),
      bannedUser: props.isUserBanned,
      onCopy: () => copyComment(),
      onTogglePin: togglePin,
      onToggleEditing: toggleEditing,
      onDelete: deleteComment,
      onHideUser: hideUser,
      onBlockUser: blockUser,
      onUnblockUser: unblockUser,
    };
    if (actions) {
      actions.actions.update(actionsProps, deps.partsIntl);
      return;
    }
    const actionsRoot = document.createElement('div');
    actions = { root: actionsRoot, actions: mountActions(actionsRoot, actionsProps, deps.partsIntl) };
  }

  function render() {
    if (destroyed) {
      return;
    }
    // a part may call back while it is being drawn, as the countdown does when it runs out: the
    // comment then draws once more after this pass instead of inside it
    if (rendering) {
      again = true;
      return;
    }
    rendering = true;
    try {
      do {
        again = false;
        draw();
      } while (again && !destroyed);
    } finally {
      rendering = false;
    }
  }

  function draw() {
    const o = props.data;
    const id = props.disabled ? undefined : `${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`;

    if (watchedDeadline !== editDeadline) {
      watchedDeadline = editDeadline;
      watchEditDeadline();
    }

    if (renderDummy && !props.editMode) {
      const [width, height] = mounted ? [root.scrollWidth, root.scrollHeight] : [100, 100];
      // the placeholder has no class, as the preact one had none
      if (drawnClass !== undefined) {
        root.removeAttribute('class');
        drawnClass = undefined;
      }
      setId(id);
      root.style.width = `${width}px`;
      root.style.height = `${height}px`;
      destroyParts();
      placeChildren(root, []);
      return;
    }

    const admin = isAdmin();
    const isReplying = props.editMode === CommentMode.Reply;
    const isEditing = props.editMode === CommentMode.Edit;
    // TODO: we also have critical_score, so we need to collapse comments with it in future
    const isUseless =
      Boolean(props.isUserBanned) ||
      Boolean(o.delete) ||
      (o.score < StaticStore.config.low_score && !o.pin && !props.disabled);

    let viewClass;
    if (props.view === 'main' || props.view === 'pinned') {
      viewClass = o.user.admin ? styles.viewAdmin : undefined;
    } else if (props.view === 'user') {
      viewClass = styles.viewUser;
    }

    const rootClass = classes(
      styles.root,
      props.theme === 'dark' ? styles.themeDark : styles.themeLight,
      viewClass,
      isUseless && styles.useless,
      props.view === 'main' && isReplying && styles.replying,
      props.view === 'main' && isEditing && styles.editing,
      props.level === 6 && styles.level6,
      props.collapsed && styles.collapsed,
      props.mix
    );
    // only when it differs, as preact wrote it: the thread adds a class of its own to highlight the
    // comment a link points at, and a redraw must leave it on
    if (rootClass !== drawnClass) {
      root.className = rootClass;
      drawnClass = rootClass;
    }
    setId(id);
    // the placeholder's size goes, and the preact component left the attribute empty
    if (root.hasAttribute('style')) {
      root.style.width = '';
      root.style.height = '';
    }

    const link = `${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.id}`;
    /** @type {Node[]} */
    const top = [];

    if (props.view === 'user' && o.title) {
      titleLink.href = link;
      titleLink.textContent = o.title;
      top.push(title);
    }

    /** @type {Node[]} */
    const infoParts = [];
    if (props.view !== 'user' && !props.collapsed) {
      if (avatar) {
        avatar.update(avatarProps());
      } else {
        avatar = avatarElement(avatarProps());
        avatarBox.append(avatar.img);
      }
      infoParts.push(avatarBox);
    }

    /** @type {Node[]} */
    const userParts = [];
    if (props.view !== 'user') {
      username.textContent = o.user.name;
      userParts.push(username);
    }
    if (admin && props.view !== 'user') {
      verificationButton.title = say('toggleVerification');
      adminBadge.setAttribute('title', say(o.user.verified ? 'verifiedUser' : 'unverifiedUser'));
      adminBadge.setAttribute(
        'class',
        classes(styles.verificationIcon, !o.user.verified && styles.verificationIconInactive)
      );
      userParts.push(verificationButton);
    }
    if (!admin && o.user.verified && props.view !== 'user') {
      readerBadge.setAttribute('title', say('verifiedUser'));
      userParts.push(readerBadge);
    }
    if (o.user.paid_sub) {
      const patreonSrc = webAssetUrl('patreon.svg');
      if (patreon.getAttribute('src') !== patreonSrc) {
        patreon.src = patreonSrc;
      }
      patreon.alt = say('paidPatreon');
      userParts.push(patreon);
    }
    placeChildren(user, userParts);
    infoParts.push(user);

    const when = new Date(o.time);
    time.href = link;
    if (Number.isNaN(when.getTime())) {
      timeText.removeAttribute('datetime');
    } else {
      timeText.dateTime = when.toISOString();
    }
    timeText.textContent = localDatetime(deps.intl, when);
    infoParts.push(time);

    if (Boolean(props.level) && /** @type {number} */ (props.level) > 0 && props.view === 'main') {
      parentLink.href = `${o.locator.url}#${COMMENT_NODE_CLASSNAME_PREFIX}${o.pid}`;
      parentLink.title = say('goToParent');
      infoParts.push(parentLink);
    }
    if (props.isUserBanned && props.view !== 'user') {
      blocked.textContent = say('blockedUser');
      infoParts.push(blocked);
    }
    if (admin && !props.isUserBanned && o.delete) {
      deleted.textContent = say('deletedBadge');
      infoParts.push(deleted);
    }
    placeVotes();
    if (votes) {
      infoParts.push(votes.root);
    }
    placeChildren(info, infoParts);
    top.push(info);

    /** @type {Node[]} */
    const bodyParts = [];
    if (!props.collapsed || props.view === 'pinned') {
      const html = o.delete ? say('deletedComment') : o.text;
      // only when it differs: setting it again would replace the nodes, and reload the images
      if (html !== textHtml) {
        text.innerHTML = html;
        textHtml = html;
      }
      // a deleted comment shows the widget's own sentence in place of its text
      if (o.delete) {
        text.setAttribute('data-nosnippet', 'true');
      } else {
        text.removeAttribute('data-nosnippet');
      }
      bodyParts.push(text);
    }
    placeActions();
    if (actions) {
      bodyParts.push(actions.root);
    }
    placeChildren(body, bodyParts);
    top.push(body);

    placeForm();
    if (form) {
      top.push(form.root);
    }

    placeChildren(root, top);
    mountPlacedForm();
  }

  /** @param {string | undefined} id */
  function setId(id) {
    if (id === undefined) {
      root.removeAttribute('id');
    } else {
      root.id = id;
    }
  }

  function destroyParts() {
    votes?.votes.destroy();
    votes = undefined;
    actions?.actions.destroy();
    actions = undefined;
    form?.form?.destroy();
    form = undefined;
  }

  render();
  // the preact component drew once more right after it appeared, which sized a placeholder from
  // what was on the page
  mounted = true;
  render();

  return {
    /**
     * @param {CommentProps} next
     * @param {CommentDeps} nextDeps
     */
    update(next, nextDeps) {
      props = next;
      deps = nextDeps;
      if (props.inView) {
        renderDummy = false;
      }
      editDeadline = deadlineFor(props);
      render();
    },
    destroy() {
      destroyed = true;
      clearTimeout(deadlineTimer);
      clearTimeout(copiedTimer);
      clearTimeout(blockTimer);
      destroyParts();
    },
  };
}

/**
 * @param {IntlShape} intl
 * @param {Date} date
 */
function localDatetime(intl, date) {
  return /** @type {string} */ (
    intl.formatMessage(messages.commentTime, { day: intl.formatDate(date), time: intl.formatTime(date) })
  );
}

const messages = defineMessages({
  deleteMessage: { id: 'comment.delete-message', defaultMessage: 'Do you really want to delete this comment?' },
  hideUserComments: {
    id: 'comment.hide-user-comment',
    defaultMessage: 'Do you want to hide all comments by {userName}?',
  },
  pinComment: { id: 'comment.pin-comment', defaultMessage: 'Do you want to pin this comment?' },
  unpinComment: { id: 'comment.unpin-comment', defaultMessage: 'Do you want to unpin this comment?' },
  verifyUser: { id: 'comment.verify-user', defaultMessage: 'Do you want to verify {userName}?' },
  unverifyUser: { id: 'comment.unverify-user', defaultMessage: 'Do you want to unverify {userName}?' },
  blockUser: { id: 'comment.block-user', defaultMessage: 'Do you want to block {userName} {duration}?' },
  unblockUser: { id: 'comment.unblock-user', defaultMessage: 'Do you want to unblock {userName}?' },
  deletedComment: { id: 'comment.deleted-comment', defaultMessage: 'This comment was deleted' },
  toggleVerification: { id: 'comment.toggle-verification', defaultMessage: 'Toggle verification' },
  verifiedUser: { id: 'comment.verified-user', defaultMessage: 'Verified user' },
  unverifiedUser: { id: 'comment.unverified-user', defaultMessage: 'Unverified user' },
  goToParent: { id: 'comment.go-to-parent', defaultMessage: 'Go to parent comment' },
  expiredTime: { id: 'comment.expired-time', defaultMessage: 'Editing time has expired.' },
  commentTime: { id: 'comment.time', defaultMessage: '{day} at {time}' },
  paidPatreon: { id: 'comment.paid-patreon', defaultMessage: 'Patreon Paid Subscriber' },
  blockedUser: { id: 'comment.blocked-user', defaultMessage: 'Blocked' },
  deletedBadge: { id: 'comment.deleted-badge', defaultMessage: 'Deleted' },
});
