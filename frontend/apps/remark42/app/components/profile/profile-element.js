// The profile page in plain JavaScript: the user, their comments a page at a time, and for the
// reader's own profile the sign-out and the request to delete their data. mountProfile places the
// profile after a node the caller owns once there is somebody to show, keeps it current, and
// destroy() takes it away with its listeners and comments.

import { classes, placeChildren } from '../../common/dom.js';
import { getDirection } from '../../common/direction.ts';
import { defineMessages } from '../../common/intl-message.ts';
import { parseQuery } from '../../utils/parse-query.ts';
import { setStyles } from '../../utils/set-dom-props.ts';
import { isUserAnonymous } from '../../utils/isUserAnonymous.ts';
import { postMessageToParent } from '../../utils/post-message.ts';
import { askReader } from '../../utils/reader.ts';
import { avatarElement } from '../avatar/avatar-element.js';
import { mountComment } from '../comment/comment-element.js';
import { preloaderElement } from '../preloader-element.js';
import { signOutIcon } from '../icons/signout-icon.js';
import { spinnerElement } from '../spinner/spinner-element.js';
import { crossIcon } from '../icons/cross-icon.js';
import { iconButtonElement } from '../icon-button/icon-button-element.js';
import { buttonElement } from '../auth/components/button-element.js';
import { messages as authMessages } from '../auth/auth.messages.ts';
import { counterElement } from './components/counter/counter-element.js';
import styles from './profile.module.css';

const COMMENTS_LIMIT = 10;
// the page Gravatar's own documentation names for uploading and choosing an avatar; a signed-out
// visitor is taken through the Gravatar login and returned to it
const GRAVATAR_AVATARS_URL = 'https://gravatar.com/profile/avatars';

/** @typedef {import('../../common/types.ts').Comment} CommentData */
/** @typedef {import('../../common/types.ts').Profile} ProfileUser */
/** @typedef {import('../../common/types.ts').User} User */
/** @typedef {import('../../common/types.ts').Theme} Theme */

/**
 * @typedef {object} ProfileDeps
 * @property {import('../../common/intl-message.ts').IntlShape} intl
 * @property {(id: string, page: { skip: number, limit: number }) => Promise<{ comments: CommentData[], count: number }>} getUserComments
 * @property {() => Promise<void>} requestDeletion
 * @property {import('../comment/votes.js').VotesDeps} votes  for the comments, whose votes are shown and not taken
 */

/** Email sign-in is the only provider whose avatar is taken from Gravatar */
function isEmailUser(/** @type {string} */ id) {
  return id.startsWith('email_');
}

/**
 * Asks the widget to sign the reader out, and the page to close the profile. The page removes this
 * frame on the message, and a logout request sent from here would go with it, so the request is
 * the widget's to make: its document stays
 */
function signout() {
  postMessageToParent({ profile: null, signout: true });
}

/**
 * @param {Node} anchor  the profile goes right after it, in the same parent
 * @param {ProfileDeps} initialDeps
 */
export function mountProfile(anchor, initialDeps) {
  let deps = initialDeps;
  let destroyed = false;
  const query = parseQuery();
  const theme = /** @type {Theme} */ (query.theme || 'light');

  // the author whose name was clicked, or, when readers open their own profile, whoever the widget
  // frame answers is signed in
  /** @type {ProfileUser | null} */
  let user = query.id ? { id: query.id, name: query.name, picture: query.picture } : null;
  // only the widget frame knows whether the reader is the one shown; the page never says
  let isCurrent = false;
  let isCommentsLoading = false;
  let error = false;
  /** @type {CommentData[] | null} */
  let comments = null;
  let commentsAmount = 0;
  let skip = 0;
  let isSigningOut = false;

  /** @type {HTMLDivElement | undefined} */
  let root;
  /** @type {ReturnType<typeof parts> | undefined} */
  let view;
  /** @type {Array<{ article: HTMLElement, comment: ReturnType<typeof mountComment> }>} */
  const mounted = [];

  const say = (/** @type {keyof typeof messages} */ name) => deps.intl.formatMessage(messages[name]);

  function parts() {
    const closeWrapper = document.createElement('div');
    closeWrapper.className = classes('profile-close-button-wrapper', styles.closeButtonWrapper);
    closeWrapper.addEventListener('click', () => handleClickClose());
    const close = iconButtonElement();
    close.append(crossIcon('16'));
    closeWrapper.append(close);

    const sidebar = document.createElement('aside');
    const header = document.createElement('header');
    header.className = classes('profile-header', styles.header);
    const avatarBox = document.createElement('div');
    avatarBox.className = classes('profile-avatar', styles.avatar);
    const avatar = avatarElement(avatarProps());
    avatarBox.append(avatar.img);
    const info = document.createElement('div');
    info.className = classes('profile-content', styles.info);
    const name = document.createElement('div');
    name.className = classes('profile-title', styles.name);
    const id = document.createElement('div');
    id.className = classes('profile-id', styles.id);
    info.append(name, id);
    const signoutButton = document.createElement('button');
    signoutButton.className = classes('profile-signout', styles.signout);
    signoutButton.title = '';
    signoutButton.addEventListener('click', () => handleClickLogout());
    const signoutIcon = signOutIcon();
    /** @type {HTMLDivElement | undefined} */
    let signoutSpinner;

    const editAvatar = document.createElement('p');
    editAvatar.className = classes('profile-edit-avatar', styles.editAvatar);
    const editAvatarLink = document.createElement('a');
    editAvatarLink.href = GRAVATAR_AVATARS_URL;
    editAvatarLink.target = '_blank';
    editAvatarLink.rel = 'noopener noreferrer';
    const editAvatarHint = document.createElement('span');
    editAvatarHint.className = styles.editAvatarHint;
    editAvatar.append(editAvatarLink, editAvatarHint);

    const content = document.createElement('section');
    content.className = classes('profile-content', styles.content);
    const errorContent = document.createElement('div');
    errorContent.className = styles.errorContent;
    const errorText = document.createElement('p');
    errorText.className = classes('profile-error', styles.error);
    const retry = buttonElement({ kind: 'link', size: 'sm' });
    retry.addEventListener('click', () => fetchComments());
    errorContent.append(errorText, retry);
    const preloader = preloaderElement(styles.preloader);

    const titleWrapper = document.createElement('div');
    titleWrapper.className = styles.titleWrapper;
    const title = document.createElement('h3');
    title.className = classes('profile-title', styles.title);
    const counterWrapper = document.createElement('div');
    counterWrapper.className = styles.counterWrapper;
    /** @type {HTMLDivElement | undefined} */
    let counter;
    const loadMoreWrapper = document.createElement('div');
    loadMoreWrapper.className = styles.loadMoreWrapper;
    const loadMore = buttonElement({ kind: 'link', size: 'sm' });
    loadMore.addEventListener('click', () => fetchComments());
    /** @type {HTMLDivElement | undefined} */
    let loadMoreSpinner;
    const emptyState = document.createElement('p');
    emptyState.className = classes('profile-emptyState', styles.emptyState);

    const footer = document.createElement('footer');
    footer.className = classes('profile-footer', styles.footer);
    const removeData = buttonElement({ kind: 'hollow', size: 'sm' });
    removeData.addEventListener('click', () => handleClickRequestRemoveData());
    footer.append(removeData);

    return {
      closeWrapper,
      close,
      sidebar,
      header,
      avatarBox,
      avatar,
      info,
      name,
      id,
      signoutButton,
      signoutIcon,
      signoutSpinner: () => (signoutSpinner ??= spinnerElement(deps.intl)),
      editAvatar,
      editAvatarLink,
      editAvatarHint,
      content,
      errorContent,
      errorText,
      retry,
      preloader,
      titleWrapper,
      title,
      counterWrapper,
      /** @param {number} count */
      counter: (count) => {
        if (!counter) {
          counter = counterElement(count);
        } else {
          counter.textContent = String(count);
        }
        return counter;
      },
      loadMoreWrapper,
      loadMore,
      loadMoreSpinner: () => (loadMoreSpinner ??= spinnerElement(deps.intl, 'gray')),
      emptyState,
      footer,
      removeData,
    };
  }

  function avatarProps() {
    const shown = /** @type {ProfileUser} */ (user);
    return {
      url: shown.picture,
      size: 32,
      identicon: isUserAnonymous(/** @type {User} */ (/** @type {unknown} */ (shown))),
    };
  }

  async function fetchComments() {
    const userId = user?.id;
    if (!userId) {
      return;
    }
    isCommentsLoading = true;
    error = false;
    render();

    try {
      const page = await deps.getUserComments(userId, { skip, limit: COMMENTS_LIMIT });
      // the skip count moves on after a successful fetch, before the page is drawn
      skip += COMMENTS_LIMIT;
      comments = [...(comments || []), ...page.comments];
      commentsAmount = page.count;
    } catch {
      error = true;
    }
    isCommentsLoading = false;
    // drawn only while the profile is up, so an answer after destroy draws nothing
    render();
  }

  function handleClickClose() {
    if (!root) {
      return;
    }
    root.classList.remove(styles.rootAppear);
    root.classList.add(styles.rootDisappear);
    // the page removes the frame on the message, so the listener goes with it
    root.addEventListener('transitionend', () => {
      postMessageToParent({ profile: null });
    });
  }

  function handleClickLogout() {
    isSigningOut = true;
    render();
    signout();
  }

  async function handleClickRequestRemoveData() {
    await deps.requestDeletion();
    if (!destroyed) {
      signout();
    }
  }

  /** @param {KeyboardEvent} evt */
  function handleKeydown(evt) {
    if (evt.code !== 'Escape') {
      return;
    }
    postMessageToParent({ profile: null });
  }

  /** @param {CommentData} data */
  function commentProps(data) {
    return /** @type {import('../comment/comment-element.js').CommentProps} */ ({
      user: null,
      data,
      level: 0,
      view: 'user',
      theme,
    });
  }

  function commentDeps() {
    return { intl: deps.intl, partsIntl: deps.intl, votes: deps.votes, actions: {} };
  }

  function render() {
    if (destroyed || !user) {
      return;
    }
    if (!root) {
      root = document.createElement('div');
      root.dir = getDirection(deps.intl.locale);
      root.className = classes('profile', styles.root);
      view = parts();
      anchor.parentNode?.insertBefore(root, anchor.nextSibling);
    }
    const p = /** @type {ReturnType<typeof parts>} */ (view);

    p.close.title = say('close');

    p.sidebar.className = classes('profile-sidebar', isCurrent && 'profile_current', styles.sidebar);
    p.avatar.update(avatarProps());
    p.name.textContent = user.name;
    p.id.textContent = user.id;

    /** @type {Node[]} */
    const headerParts = [p.avatarBox, p.info];
    if (isCurrent) {
      p.signoutButton.title = /** @type {string} */ (deps.intl.formatMessage(authMessages.signout));
      p.signoutButton.disabled = isSigningOut;
      placeChildren(p.signoutButton, [isSigningOut ? p.signoutSpinner() : p.signoutIcon]);
      headerParts.push(p.signoutButton);
    }
    placeChildren(p.header, headerParts);

    /** @type {Node[]} */
    const sidebarParts = [p.header];
    if (isCurrent && isEmailUser(user.id)) {
      p.editAvatarLink.textContent = say('editAvatar');
      p.editAvatarHint.textContent = say('editAvatarHint');
      sidebarParts.push(p.editAvatar);
    }

    /** @type {Node[]} */
    const contentParts = [];
    if (error) {
      p.errorText.textContent = say('error');
      p.retry.textContent = say('retry');
      contentParts.push(p.errorContent);
    }
    if (comments === null && isCommentsLoading) {
      contentParts.push(p.preloader);
    }
    if (comments !== null) {
      if (comments.length) {
        p.title.textContent = say(isCurrent ? 'myComments' : 'comments');
        placeChildren(p.titleWrapper, commentsAmount > 0 ? [p.title, p.counterWrapper] : [p.title]);
        if (commentsAmount > 0) {
          placeChildren(p.counterWrapper, [p.counter(commentsAmount)]);
        }
        contentParts.push(p.titleWrapper);

        // the list only grows, a page at a time
        comments.forEach((data, i) => {
          if (mounted[i]) {
            mounted[i].comment.update(commentProps(data), commentDeps());
          } else {
            const article = document.createElement('article');
            mounted.push({ article, comment: mountComment(article, commentProps(data), commentDeps()) });
          }
          contentParts.push(mounted[i].article);
        });

        if (commentsAmount > skip) {
          if (isCommentsLoading) {
            placeChildren(p.loadMoreWrapper, [p.loadMoreSpinner()]);
          } else {
            p.loadMore.textContent = say('loadMore');
            placeChildren(p.loadMoreWrapper, [p.loadMore]);
          }
          contentParts.push(p.loadMoreWrapper);
        }
      } else {
        p.emptyState.textContent = say('empty');
        contentParts.push(p.emptyState);
      }
    }
    placeChildren(p.content, contentParts);
    sidebarParts.push(p.content);

    if (isCurrent && !isUserAnonymous(/** @type {User} */ (/** @type {unknown} */ (user)))) {
      p.removeData.textContent = say('requestDeletion');
      sidebarParts.push(p.footer);
    }
    placeChildren(p.sidebar, sidebarParts);
    placeChildren(root, [p.closeWrapper, p.sidebar]);
  }

  /** draws the profile for the first time, then lets it fade in */
  function appear() {
    render();
    if (root && !root.classList.contains(styles.rootAppear) && !root.classList.contains(styles.rootDisappear)) {
      // the class starts the transition, so the browser first takes the profile as it is without it
      void root.offsetWidth;
      root.classList.add(styles.rootAppear);
    }
  }

  const pageStyles = { height: '100%', padding: 0 };
  setStyles(document.documentElement, pageStyles);
  setStyles(document.body, pageStyles);
  document.addEventListener('keydown', handleKeydown);

  appear();
  fetchComments();

  if (query.token) {
    askReader(query.token).then((reader) => {
      if (destroyed) {
        return;
      }
      if (!query.id) {
        if (reader) {
          user = reader;
          isCurrent = true;
          appear();
          fetchComments();
        } else {
          // the reader signed out before the profile asked, so there is nobody to show
          postMessageToParent({ profile: null });
        }
        return;
      }
      isCurrent = reader?.id === query.id;
      render();
    });
  }

  return {
    /** @param {ProfileDeps} next */
    update(next) {
      deps = next;
      render();
    },
    destroy() {
      destroyed = true;
      document.removeEventListener('keydown', handleKeydown);
      mounted.forEach(({ comment }) => comment.destroy());
      root?.remove();
    },
  };
}

const messages = defineMessages({
  close: { id: 'profile.close', defaultMessage: 'Close profile' },
  myComments: { id: 'user.my-comments', defaultMessage: 'Your comments' },
  comments: { id: 'user.comments', defaultMessage: 'Comments' },
  loadMore: { id: 'user.load-more', defaultMessage: 'Load more' },
  empty: { id: 'empty-state', defaultMessage: 'No comments yet.' },
  error: { id: 'errors.0', defaultMessage: 'Something went wrong. Please try again a bit later.' },
  retry: { id: 'retry', defaultMessage: 'Retry' },
  editAvatar: { id: 'profile.edit-avatar', defaultMessage: 'Edit avatar' },
  editAvatarHint: {
    id: 'profile.edit-avatar-hint',
    defaultMessage:
      'The avatar comes from Gravatar for the email you sign in with. After changing it there, sign out and sign in again; a browser may still show the old one for up to a week.',
  },
  requestDeletion: { id: 'profile.request-to-delete-data', defaultMessage: 'Request deletion of my data' },
});
