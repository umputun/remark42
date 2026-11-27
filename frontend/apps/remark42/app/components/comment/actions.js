// The actions under a comment (reply, edit with its countdown, hide, copy, pin, block, delete), in
// plain JavaScript. mountActions fills an element the caller owns; update() redraws it in place
// and destroy() stops the countdown. Each button calls the handler of the latest props, so a
// caller may pass new functions on every update.

import { classes, placeChildren } from '../../common/dom.js';
import { defineMessages } from '../../common/intl-message.ts';
import { buttonElement } from '../auth/components/button-element.js';
import { countdown } from '../countdown.js';
import { selectElement } from '../select/select-element.js';
import { getBlockingDurations } from './getBlockingDurations.ts';
import styles from './comment-actions.module.css';

/**
 * @typedef {object} ActionsProps
 * @property {boolean | undefined} admin
 * @property {boolean | undefined} currentUser  the comment is the reader's own
 * @property {boolean | undefined} pinned
 * @property {boolean | undefined} copied
 * @property {boolean | undefined} bannedUser  its author is blocked
 * @property {boolean | undefined} readOnly
 * @property {boolean | undefined} editing
 * @property {boolean | undefined} replying
 * @property {boolean} editable
 * @property {number | undefined} editDeadline  when editing ends, in milliseconds; Infinity for never
 * @property {() => void} onCopy
 * @property {() => void} onToggleEditing
 * @property {() => void} onDelete
 * @property {() => void} onTogglePin
 * @property {() => void} onToggleReplying
 * @property {() => void} onHideUser
 * @property {(ttl: import('../../common/types.ts').BlockTTL) => void} onBlockUser
 * @property {() => void} onUnblockUser
 * @property {() => void} onDisableEditing  the edit deadline has passed
 */

/**
 * @param {HTMLElement} root  an empty element; its attributes and children are this component's
 * @param {ActionsProps} initial
 * @param {import('../../common/intl-message.ts').IntlShape} initialIntl
 */
export function mountActions(root, initial, initialIntl) {
  let props = initial;
  let intl = initialIntl;

  root.className = classes('comment-actions', styles.root);
  root.setAttribute('data-nosnippet', 'true');

  const link = () => buttonElement({ kind: 'link', size: 'sm' });
  const reply = link();
  const edit = link();
  const timer = document.createElement('span');
  timer.setAttribute('role', 'timer');
  timer.title = '';
  timer.className = classes('comment-actions-countdown', styles.countdown);
  const timeLeft = countdown(() => props.onDisableEditing());
  timer.append(timeLeft.text);

  const additional = document.createElement('div');
  additional.dataset.testid = 'comment-actions-additional';
  additional.className = classes('comment-actions-additional', styles.additionalActions);
  const hide = link();
  const copy = link();
  const pin = link();
  const unblock = link();
  const remove = link();
  const blockProps = () => ({
    items: getBlockingDurations(intl),
    size: /** @type {const} */ ('sm'),
    title: intl.formatMessage(messages.blockingPeriod),
    onChange: (/** @type {string} */ ttl) =>
      props.onBlockUser(/** @type {import('../../common/types.ts').BlockTTL} */ (ttl)),
  });
  /** @type {ReturnType<typeof selectElement> | undefined} */
  let block;

  reply.addEventListener('click', () => props.onToggleReplying());
  edit.addEventListener('click', () => props.onToggleEditing());
  hide.addEventListener('click', () => props.onHideUser());
  copy.addEventListener('click', () => props.onCopy());
  pin.addEventListener('click', () => props.onTogglePin());
  unblock.addEventListener('click', () => props.onUnblockUser());
  remove.addEventListener('click', () => props.onDelete());

  render();

  function render() {
    const { admin, currentUser, pinned, copied, bannedUser, readOnly, editing, replying, editable, editDeadline } =
      props;
    const say = (/** @type {keyof typeof messages} */ name) => intl.formatMessage(messages[name]);

    reply.textContent = say(replying ? 'cancel' : 'reply');
    edit.textContent = say(editing ? 'cancel' : 'edit');
    timer.title = say('editCountdown');
    hide.textContent = say('hide');
    copy.textContent = say(copied ? 'copied' : 'copy');
    copy.disabled = Boolean(copied);
    pin.textContent = say(pinned ? 'unpin' : 'pin');
    unblock.textContent = say('unblock');
    remove.textContent = say('delete');
    // a new select each time it comes back, so after a block and an unblock it offers the periods
    // afresh and picking the same one again blocks again
    if (admin && !bannedUser) {
      if (block?.root.parentNode === additional) {
        block.update(blockProps());
      } else {
        block = selectElement(blockProps());
      }
    }

    /** @type {Node[]} */
    const top = [];
    if (!readOnly) {
      top.push(reply);
    }
    const counting = Boolean(editable && editDeadline && Number.isFinite(editDeadline));
    if (editable && editDeadline) {
      top.push(edit);
      if (counting) {
        top.push(timer);
      }
    }
    top.push(additional);

    /** @type {Node[]} */
    const more = [];
    if (!currentUser) {
      more.push(hide);
    }
    if (admin) {
      more.push(copy, pin, bannedUser || !block ? unblock : block.root);
    }
    if ((currentUser && editDeadline !== undefined) || admin) {
      more.push(remove);
    }

    placeChildren(additional, more);
    placeChildren(root, top);

    // last, since reaching the deadline calls back into the caller
    if (counting) {
      timeLeft.countTo(/** @type {number} */ (editDeadline));
    } else {
      timeLeft.stop();
    }
  }

  return {
    /**
     * @param {ActionsProps} next
     * @param {import('../../common/intl-message.ts').IntlShape} [nextIntl]
     */
    update(next, nextIntl = intl) {
      props = next;
      intl = nextIntl;
      render();
    },
    destroy() {
      timeLeft.stop();
    },
  };
}

const messages = defineMessages({
  unblock: { id: 'comment.unblock', defaultMessage: 'Unblock' },
  pin: { id: 'comment.pin', defaultMessage: 'Pin' },
  unpin: { id: 'comment.unpin', defaultMessage: 'Unpin' },
  hide: { id: 'comment.hide', defaultMessage: 'Hide user' },
  cancel: { id: 'comment.cancel', defaultMessage: 'Cancel' },
  edit: { id: 'comment.edit', defaultMessage: 'Edit' },
  reply: { id: 'comment.reply', defaultMessage: 'Reply' },
  delete: { id: 'comment.delete', defaultMessage: 'Delete' },
  editCountdown: { id: 'comment.edit-countdown', defaultMessage: 'Time left to edit' },
  copied: { id: 'comment.copied', defaultMessage: 'Copied!' },
  copy: { id: 'comment.copy', defaultMessage: 'Copy' },
  blockingPeriod: { id: 'comment.blocking-period', defaultMessage: 'Blocked for' },
});
