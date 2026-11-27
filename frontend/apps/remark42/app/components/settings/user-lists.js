// The settings page of the widget, with the hidden and blocked users, in plain JavaScript.
// mountSettings fills an element the caller owns; update() redraws it in place and destroy()
// makes its buttons inert. Each button calls the handler of the latest props.
//
// The lists are the ones the page opened with. A user shown again or unblocked stays in the list,
// dimmed, with the action that takes the change back, until the page is opened again.

import { classes, placeChildren } from '../../common/dom.js';
import { defineMessages } from '../../common/intl-message.ts';
import { keyedList } from '../../common/list.js';
import styles from './settings.module.css';

/** @typedef {import('../../common/types.ts').User} User */
/** @typedef {import('../../common/types.ts').BlockedUser} BlockedUser */

/**
 * @typedef {object} SettingsProps
 * @property {import('../../common/types.ts').Theme} theme
 * @property {import('../../common/intl-message.ts').IntlShape} intl
 * @property {User | null} user  the reader
 * @property {BlockedUser[]} blockedUsers
 * @property {Record<string, User>} hiddenUsers
 * @property {(id: string, name: string, ttl: import('../../common/types.ts').BlockTTL) => Promise<void>} blockUser
 * @property {(id: string) => Promise<void>} unblockUser
 * @property {(user: User) => void} hideUser
 * @property {(id: string) => void} unhideUser
 * @property {() => void} onUnblockSomeone
 */

/**
 * @param {HTMLElement} root  an empty element; its attributes and children are this component's
 * @param {SettingsProps} initial
 */
export function mountSettings(root, initial) {
  let props = initial;
  let destroyed = false;
  // each row keyed by its place among the users with the same id, so a list that names one user
  // twice, as a record written by something else may, still draws a row for each
  const hiddenUsers = withKeys(Object.values({ ...initial.hiddenUsers }));
  const blockedUsers = withKeys(initial.blockedUsers.slice());
  /** @type {Set<string>} */
  const unblocked = new Set();
  /** @type {Set<string>} */
  const unhidden = new Set();

  root.className = '';
  root.setAttribute('data-nosnippet', 'true');

  /** @type {Map<string, RowParts>} */
  const hiddenParts = new Map();
  /** @type {Map<string, RowParts>} */
  const blockedParts = new Map();

  const hiddenSection = section();
  const hiddenEmpty = dimmed();
  const hiddenList = list();
  const hiddenRows = keyedList(hiddenList, (/** @type {Keyed<User>} */ row) => row.key, hiddenRow);

  const blockedSection = section();
  const blockedEmpty = dimmed();
  const blockedList = list();
  const blockedRows = keyedList(blockedList, (/** @type {Keyed<BlockedUser>} */ row) => row.key, blockedRow);

  render();

  /** @param {keyof typeof messages} name @param {Record<string, string>} [values] */
  function say(name, values) {
    return /** @type {string} */ (props.intl.formatMessage(messages[name], values));
  }

  /** @param {User} user */
  function toggleHidden(user) {
    if (unhidden.has(user.id)) {
      unhidden.delete(user.id);
      props.hideUser(user);
    } else {
      unhidden.add(user.id);
      props.unhideUser(user.id);
      props.onUnblockSomeone();
    }
    render();
  }

  /** @param {BlockedUser} user */
  function toggleBlocked(user) {
    if (unblocked.has(user.id)) {
      if (!window.confirm(say('blockUser', { userName: user.name }))) {
        return;
      }
      unblocked.delete(user.id);
      render();
      props.blockUser(user.id, user.name, 'permanently');
    } else {
      if (!window.confirm(say('unblockUser', { userName: user.name }))) {
        return;
      }
      unblocked.add(user.id);
      render();
      props.unblockUser(user.id);
      props.onUnblockSomeone();
    }
  }

  /** @param {Keyed<User>} entry */
  function hiddenRow({ key, user }) {
    const row = userRow(user);
    const action = actionButton(() => toggleHidden(user), 'action-first');
    row.node.append(row.username, action, row.idLine);
    hiddenParts.set(key, { username: row.username, action });
    return row.node;
  }

  /** @param {Keyed<BlockedUser>} entry */
  function blockedRow({ key, user }) {
    const row = userRow(user);
    const period = document.createElement('span');
    const periodText = document.createTextNode('');
    period.append(' ', periodText);
    const action = actionButton(() => toggleBlocked(user), 'role-first');
    row.node.append(row.username, period, action, row.idLine);
    blockedParts.set(key, { username: row.username, action, periodText });
    return row.node;
  }

  /** @param {Date} time */
  function blockPeriod(time) {
    // a block more than 50 years away is taken to be permanent
    if (time.getFullYear() - new Date().getFullYear() >= 50) {
      return say('permanently');
    }
    return say('blockTime', { day: props.intl.formatDate(time), time: props.intl.formatTime(time) });
  }

  /**
   * A clickable span that works from the keyboard as well. Its attributes come in the order the
   * preact component wrote them, which differed between the two lists.
   *
   * @param {() => void} onPress
   * @param {'action-first' | 'role-first'} order
   */
  function actionButton(onPress, order) {
    const button = document.createElement('span');
    if (order === 'action-first') {
      button.className = styles.action;
    }
    button.setAttribute('role', 'button');
    button.tabIndex = 0;
    if (order === 'role-first') {
      button.className = styles.action;
    }
    const press = () => {
      if (!destroyed) {
        onPress();
      }
    };
    button.addEventListener('click', press);
    button.addEventListener('keypress', (event) => {
      if (event.key === ' ' || event.key === 'Enter') {
        event.preventDefault();
        press();
      }
    });
    return button;
  }

  function render() {
    const { theme, user } = props;
    root.className = classes(styles.root, theme === 'dark' ? styles.themeDark : styles.themeLight);

    hiddenSection.node.setAttribute('aria-label', say('hiddenUsers'));
    setText(hiddenSection.title, say('hiddenUsersHeader'));
    setText(hiddenEmpty, say('noHiddenUsers'));
    hiddenRows.update(hiddenUsers);
    for (const { key, user: hidden } of hiddenUsers) {
      const parts = /** @type {RowParts} */ (hiddenParts.get(key));
      const shown = unhidden.has(hidden.id);
      parts.username.className = classes(styles.username, shown && styles.invisible);
      setText(parts.username, hidden.name ? hidden.name : say('unknown'));
      setText(parts.action, say(shown ? 'hide' : 'show'));
    }
    placeChildren(hiddenSection.node, [hiddenSection.title, hiddenUsers.length ? hiddenList : hiddenEmpty]);

    /** @type {Node[]} */
    const sections = [hiddenSection.node];
    if (user && user.admin) {
      blockedSection.node.setAttribute('aria-label', say('blockedUsers'));
      setText(blockedSection.title, say('blockedUsersHeader'));
      setText(blockedEmpty, say('noBlockedUsers'));
      blockedRows.update(blockedUsers);
      for (const { key, user: blocked } of blockedUsers) {
        const parts = /** @type {RowParts} */ (blockedParts.get(key));
        const lifted = unblocked.has(blocked.id);
        parts.username.className = classes(styles.username, lifted && styles.invisible);
        setText(parts.username, blocked.name ? blocked.name : say('unknown'));
        const period = /** @type {Text} */ (parts.periodText);
        const text = blockPeriod(new Date(blocked.time));
        if (period.data !== text) {
          period.data = text;
        }
        setText(parts.action, say(lifted ? 'block' : 'unblock'));
      }
      placeChildren(blockedSection.node, [blockedSection.title, blockedUsers.length ? blockedList : blockedEmpty]);
      sections.push(blockedSection.node);
    }
    placeChildren(root, sections);
  }

  return {
    /** @param {SettingsProps} next */
    update(next) {
      props = next;
      if (!destroyed) {
        render();
      }
    },
    destroy() {
      destroyed = true;
    },
  };
}

/**
 * @template T
 * @typedef {{ key: string, user: T }} Keyed
 */

/**
 * The users with a key each: the id, followed by the count of earlier users with the same id.
 *
 * @template {{ id: string }} T
 * @param {T[]} users
 * @returns {Keyed<T>[]}
 */
function withKeys(users) {
  /** @type {Map<string, number>} */
  const seen = new Map();
  return users.map((user) => {
    // the id as text, so a stored number and the same digits as a string count as one
    const id = String(user.id);
    const count = seen.get(id) ?? 0;
    seen.set(id, count + 1);
    return { key: `${id}#${count}`, user };
  });
}

/**
 * Sets the text of an element only when it changes, so a selection inside it survives a render
 * that leaves it as it was.
 *
 * @param {HTMLElement} element
 * @param {string} text
 */
function setText(element, text) {
  if (element.textContent !== text) {
    element.textContent = text;
  }
}

/**
 * @typedef {object} RowParts
 * @property {HTMLSpanElement} username
 * @property {HTMLSpanElement} action
 * @property {Text} [periodText]  the block period, in the blocked list only
 */

/**
 * A list item with the user's name and id, the parts both lists share.
 *
 * @param {{ id: string }} user
 */
function userRow(user) {
  const node = document.createElement('li');
  node.className = styles.listItem;
  const username = document.createElement('span');
  username.className = styles.username;
  if (user.id !== undefined) {
    username.title = user.id;
  }
  const idLine = document.createElement('div');
  const label = document.createElement('span');
  label.className = styles.userId;
  const value = document.createElement('span');
  value.textContent = user.id;
  label.append('id: ', value);
  idLine.append(label);
  return { node, username, idLine };
}

function section() {
  const node = document.createElement('div');
  node.className = styles.section;
  node.setAttribute('role', 'region');
  node.setAttribute('aria-label', '');
  const title = document.createElement('h3');
  return { node, title };
}

function dimmed() {
  const element = document.createElement('h4');
  element.className = styles.dimmed;
  return element;
}

function list() {
  const element = document.createElement('ul');
  element.className = styles.list;
  return element;
}

const messages = defineMessages({
  blockUser: { id: 'settings.block-user', defaultMessage: 'Do you want to block {userName}?' },
  unblockUser: { id: 'settings.unblock-user', defaultMessage: 'Do you want to unblock {userName}?' },
  hiddenUsers: { id: 'settings.hidden-users-title', defaultMessage: 'Hidden users' },
  blockedUsers: { id: 'settings.blocked-users-title', defaultMessage: 'Blocked users' },
  hiddenUsersHeader: { id: 'settings.hidden-user-header', defaultMessage: 'Hidden users:' },
  noHiddenUsers: { id: 'settings.no-hidden-users', defaultMessage: 'There are no hidden users.' },
  unknown: { id: 'settings.unknown', defaultMessage: 'unknown' },
  show: { id: 'settings.show', defaultMessage: 'show' },
  hide: { id: 'settings.hide', defaultMessage: 'hide' },
  blockedUsersHeader: { id: 'settings.blocked-users-header', defaultMessage: 'Blocked users:' },
  noBlockedUsers: { id: 'settings.no-blocked-users', defaultMessage: 'There are no blocked users.' },
  block: { id: 'settings.block', defaultMessage: 'block' },
  unblock: { id: 'settings.unblock', defaultMessage: 'unblock' },
  permanently: { id: 'settings.permanently', defaultMessage: 'permanently' },
  blockTime: { id: 'settings.block-time', defaultMessage: 'until {day} at {time}' },
});
