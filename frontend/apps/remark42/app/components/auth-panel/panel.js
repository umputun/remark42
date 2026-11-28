// The panel above the comments, in plain JavaScript: the signed-in user with their sign-out
// button, the sign-in form on a read-only page, the warnings about blocked cookies, and the
// settings and read-only switches. mountAuthPanel fills an element the caller owns; update()
// redraws it in place and destroy() takes the sign-in form down with it.

import { classes, placeChildren } from '../../common/dom.js';
import { IS_STORAGE_AVAILABLE, IS_THIRD_PARTY } from '../../common/constants.ts';
import { defineMessages } from '../../common/intl-message.ts';
import { postMessageToParent } from '../../utils/post-message.ts';
import { issueReaderToken } from '../../utils/reader.ts';
import { isUserAnonymous } from '../../utils/isUserAnonymous.ts';
import { avatarElement } from '../avatar/avatar-element.js';
import { buttonElement } from '../button/button-element.js';
import { iconButtonElement } from '../icon-button/icon-button-element.js';
import { signOutIcon } from '../icons/signout-icon.js';
import { messages as authMessages } from '../auth/auth.messages.ts';
import { mountSignIn } from '../auth/sign-in.js';
import styles from './auth-panel.module.css';

/**
 * @typedef {import('../../common/types.ts').User} User
 */

/**
 * @typedef {object} AuthPanelProps
 * @property {User | null} user
 * @property {Record<string, unknown>} hiddenUsers  the users this reader hid, by id
 * @property {boolean} isCommentsDisabled
 * @property {import('../../common/types.ts').PostInfo} postInfo
 * @property {(evt?: Event) => unknown} signout
 * @property {(readOnly: boolean) => unknown} onCommentsChangeReadOnlyMode
 * @property {() => void} [onBlockedUsersShow]
 * @property {() => void} [onBlockedUsersHide]
 */

/**
 * @param {HTMLElement} root  an empty element; its attributes and children are this component's
 * @param {AuthPanelProps} initial
 * @param {import('../auth/sign-in.js').SignInDeps} initialDeps  for the sign-in form, and the intl
 *   of the panel itself
 */
export function mountAuthPanel(root, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  let settingsShown = false;

  root.className = styles.root;
  root.setAttribute('data-nosnippet', 'true');

  const userColumn = document.createElement('div');
  userColumn.className = styles.column;

  const userBox = document.createElement('div');
  userBox.className = styles.user;
  const profileButton = document.createElement('button');
  profileButton.className = styles.userButton;
  profileButton.title = '';
  const avatarBox = document.createElement('div');
  avatarBox.className = styles.userAvatar;
  const avatar = avatarElement({ size: 20 });
  avatarBox.append(avatar.img);
  const userName = document.createTextNode('');
  profileButton.append(avatarBox, userName);
  profileButton.addEventListener('click', () => {
    // the page embedding the widget receives this message, so it names nobody: the profile frame
    // asks this frame who the reader is
    postMessageToParent({ profile: { token: issueReaderToken() } });
  });
  const signOutBox = document.createElement('div');
  signOutBox.className = styles.userLogoutButton;
  const signOut = iconButtonElement();
  signOut.title = '';
  signOut.append(signOutIcon('14'));
  signOut.addEventListener('click', (evt) => props.signout(evt));
  signOutBox.append(signOut);
  userBox.append(profileButton, document.createTextNode(' '), signOutBox);

  /** @type {{ host: HTMLElement, form: ReturnType<typeof mountSignIn> } | null} */
  let signIn = null;

  const thirdParty = document.createElement('div');
  thirdParty.className = styles.column;
  const thirdPartyText = document.createTextNode('');
  const newPage = document.createElement('a');
  newPage.className = styles.textLink;
  newPage.href = '';
  newPage.target = '_blank';
  newPage.rel = 'noreferrer';
  thirdParty.append(thirdPartyText, document.createTextNode(' '), newPage);

  const cookies = document.createElement('div');
  cookies.className = styles.column;

  const switches = document.createElement('div');
  switches.className = classes(styles.column, styles.columnSeparated);
  const settings = linkButton(() => {
    if (settingsShown) {
      props.onBlockedUsersHide?.();
    } else {
      props.onBlockedUsersShow?.();
    }
    settingsShown = !settingsShown;
    render();
  });
  const settingsBox = document.createElement('span');
  settingsBox.append(settings);
  const readOnly = linkButton(() => props.onCommentsChangeReadOnlyMode(!props.isCommentsDisabled));
  const readOnlyBox = document.createElement('span');
  readOnlyBox.append(readOnly);
  const disabledNote = document.createElement('span');

  render();

  function render() {
    const { user, postInfo, hiddenUsers, isCommentsDisabled } = props;
    const say = (/** @type {keyof typeof messages} */ name) => deps.intl.formatMessage(messages[name]);
    const admin = Boolean(user?.admin);

    if (user) {
      profileButton.title = deps.intl.formatMessage(authMessages.openProfile);
      avatar.update({ url: user.picture, size: 20, identicon: isUserAnonymous(user) });
      userName.data = user.name;
      signOut.title = deps.intl.formatMessage(authMessages.signout);
    }

    const wantsSignIn = !user && Boolean(postInfo.read_only);
    if (wantsSignIn && !signIn) {
      const host = document.createElement('div');
      signIn = { host, form: mountSignIn(host, deps) };
    } else if (!wantsSignIn && signIn) {
      signIn.form.destroy();
      signIn = null;
    } else if (signIn) {
      signIn.form.update(deps);
    }
    placeChildren(userColumn, user ? [userBox] : signIn ? [signIn.host] : []);

    /** @type {Node[]} */
    const columns = [userColumn];
    if (!IS_STORAGE_AVAILABLE && IS_THIRD_PARTY) {
      thirdPartyText.data = say('thirdPartyCookies');
      newPage.href = `${window.location.origin}/web/comments.html${window.location.search}`;
      newPage.textContent = say('newPage');
      columns.push(thirdParty);
    }
    if (!IS_STORAGE_AVAILABLE && !IS_THIRD_PARTY) {
      cookies.textContent = say('enableCookies');
      columns.push(cookies);
    }
    columns.push(switches);
    placeChildren(root, columns);

    settings.textContent = say(settingsShown ? 'hideSettings' : 'showSettings');
    readOnly.textContent = say(isCommentsDisabled ? 'enableComments' : 'disableComments');
    disabledNote.textContent = say('readOnly');

    /** @type {Node[]} */
    const shown = [];
    if (Object.keys(hiddenUsers).length > 0 || admin || settingsShown) {
      shown.push(settingsBox);
    }
    if (admin) {
      shown.push(readOnlyBox);
    }
    if (!admin && postInfo.read_only) {
      shown.push(disabledNote);
    }
    placeChildren(switches, shown);
  }

  return {
    /**
     * @param {AuthPanelProps} next
     * @param {import('../auth/sign-in.js').SignInDeps} [nextDeps]
     */
    update(next, nextDeps = deps) {
      props = next;
      deps = nextDeps;
      render();
    },
    destroy() {
      signIn?.form.destroy();
      signIn = null;
    },
  };
}

/**
 * A button drawn as a link, working from the keyboard as a link does, as the panel's preact
 * buttons were.
 *
 * @param {() => void} onPress
 */
function linkButton(onPress) {
  const button = buttonElement({ kind: 'link', className: styles.adminAction });
  button.setAttribute('role', 'link');
  button.tabIndex = 0;
  button.addEventListener('click', () => onPress());
  button.addEventListener('keypress', (evt) => {
    if (evt.key === ' ' || evt.key === 'Enter') {
      evt.preventDefault();
      onPress();
    }
  });
  return button;
}

const messages = defineMessages({
  thirdPartyCookies: {
    id: 'authPanel.disabled-cookies',
    defaultMessage: 'Enable third-party cookies to log in, or open comments in',
  },
  newPage: { id: 'authPanel.new-page', defaultMessage: 'a new page' },
  enableCookies: { id: 'authPanel.enable-cookies', defaultMessage: 'Allow cookies to log in and comment' },
  hideSettings: { id: 'authPanel.hide-settings', defaultMessage: 'Hide settings' },
  showSettings: { id: 'authPanel.show-settings', defaultMessage: 'Show settings' },
  enableComments: { id: 'authPanel.enable-comments', defaultMessage: 'Enable comments' },
  disableComments: { id: 'authPanel.disable-comments', defaultMessage: 'Disable comments' },
  readOnly: { id: 'authPanel.read-only', defaultMessage: 'Comments disabled' },
});
