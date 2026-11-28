// The email subscription in plain JavaScript: a dropdown whose form asks for an address, then for
// the token the email carries, and shows the subscription with a way to end it.
//
// The link in the email confirms the subscription from another tab or another device, so while the
// form waits for the token, or has just unsubscribed, it asks whether that has happened: each time
// the page becomes visible again, and every few seconds until the token expires.

import { classes, placeChildren } from '../../../common/dom.js';
import { defineMessages } from '../../../common/intl-message.ts';
import { extractErrorMessageFromResponse } from '../../../utils/errorUtils.ts';
import { isJwtExpired } from '../../../utils/jwt.ts';
import { getPersistedEmail, isCompleteToken } from '../../auth/auth.utils.ts';
import { buttonClassName, buttonElement } from '../../button/button-element.js';
import { dropdownElement } from '../../dropdown/dropdown-element.js';
import { inputElement } from '../../input/input-element.js';
import { preloaderElement } from '../../preloader-element.js';
import { textareaElement } from '../../textarea-element.js';
import styles from './subscribe-by-email.module.css';

// how often the token step asks whether the link in the email has confirmed the subscription
export const CONFIRMATION_POLL_INTERVAL = 10 * 1000;
// how long it keeps asking: the confirmation token the email carries expires after 30 minutes
export const CONFIRMATION_POLL_DURATION = 30 * 60 * 1000;

const EMAIL = /[^@]+@[^.]+\..+/;

/** @param {string} a @param {string} b */
function sameAddress(a, b) {
  return a.trim().toLowerCase() === b.trim().toLowerCase();
}

/** @typedef {'email' | 'token' | 'subscribed' | 'unsubscribed'} Step */

/**
 * @typedef {object} EmailSubscriptionProps
 * @property {boolean} subscribed  the reader is subscribed already
 * @property {string | undefined} userId
 * @property {boolean} anonymous  only registered users can subscribe
 * @property {'light' | 'dark'} theme
 */

/**
 * @typedef {object} EmailSubscriptionDeps
 * @property {import('../../../common/intl-message.ts').IntlShape} intl
 * @property {(address: string) => Promise<{ address: string, updated: boolean }>} verify  asks for the email
 * @property {(token: string) => Promise<unknown>} confirm  sends the token from the email
 * @property {() => Promise<unknown>} unsubscribe
 * @property {() => Promise<{ user: { id: string }, address: string } | null | undefined>} subscription
 *   the subscription the server holds for the signed-in reader
 * @property {(on: boolean) => void} setSubscribed  stores whether the reader is subscribed
 */

/**
 * @param {HTMLElement} root  an empty div; its attributes and children are this component's
 * @param {EmailSubscriptionProps} initial
 * @param {EmailSubscriptionDeps} initialDeps
 */
export function mountSubscribeByEmail(root, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  /** @type {ReturnType<typeof emailForm> | null} */
  let form = null;

  const dropdown = dropdownElement(root, dropdownProps(), (items, control) => {
    form = emailForm(items, props, deps, control);
    return {
      destroy() {
        form?.destroy();
        form = null;
      },
    };
  });

  function dropdownProps() {
    const { intl } = deps;
    return {
      title: intl.formatMessage(messages.email),
      theme: props.theme,
      disabled: props.anonymous,
      buttonTitle: intl.formatMessage(props.anonymous ? messages.onlyRegisteredUsers : messages.subscribeByEmail),
    };
  }

  return {
    /**
     * @param {EmailSubscriptionProps} next
     * @param {EmailSubscriptionDeps} [nextDeps]
     */
    update(next, nextDeps = deps) {
      props = next;
      deps = nextDeps;
      dropdown.update(dropdownProps());
      form?.update(props, deps);
    },
    destroy() {
      dropdown.destroy();
    },
  };
}

/**
 * The form inside the open dropdown. It starts on the subscription when the reader has one.
 *
 * @param {HTMLElement} slot  the element whose children the form sets
 * @param {EmailSubscriptionProps} initial
 * @param {EmailSubscriptionDeps} initialDeps
 * @param {import('../../dropdown/dropdown-element.js').DropdownControl} control
 */
function emailForm(slot, initial, initialDeps, control) {
  let props = initial;
  let deps = initialDeps;
  /** @type {Step} */
  let step = props.subscribed ? 'subscribed' : 'email';
  let token = '';
  let address = getPersistedEmail();
  let loading = false;
  /** @type {string | null} */
  let error = null;
  let justSubscribed = false;
  let destroyed = false;
  /** @type {(() => void) | null} */
  let stopWatching = null;

  const form = document.createElement('form');
  form.addEventListener('submit', (evt) => {
    evt.preventDefault();
    void send();
  });

  const title = document.createElement('div');
  title.className = styles.title;
  const description = document.createElement('div');
  description.className = styles.description;
  const addressField = inputElement({}, { autofocus: true, placeholder: '' });
  addressField.addEventListener('input', (evt) => {
    evt.preventDefault();
    error = null;
    address = addressField.value;
    render();
  });

  const back = buttonElement({ kind: 'link' });
  back.setAttribute('role', 'button');
  back.tabIndex = 0;
  back.addEventListener('click', toEmailStep);
  back.addEventListener('keypress', (evt) => {
    if (evt.key === ' ' || evt.key === 'Enter') {
      evt.preventDefault();
      toEmailStep();
    }
  });
  const tokenField = textareaElement(
    { class: styles.tokenInput, placeholder: '', autofocus: true },
    { onInput: onToken, controlled: true }
  );

  const errorBox = document.createElement('div');
  errorBox.className = styles.error;
  errorBox.setAttribute('role', 'alert');
  const submit = buttonElement({ kind: 'primary', size: 'large', className: styles.button, type: 'submit' });
  const preloader = preloaderElement(styles.preloader);
  const submitLabel = document.createTextNode('');

  const subscribedBox = document.createElement('div');
  subscribedBox.className = styles.subscribed;
  const subscribedText = document.createTextNode('');
  const unsubscribe = buttonElement({ kind: 'primary', size: 'middle', className: styles.button });
  unsubscribe.addEventListener('click', () => void leave());
  subscribedBox.append(subscribedText, unsubscribe);

  const unsubscribedBox = document.createElement('div');
  unsubscribedBox.className = styles.unsubscribed;
  const unsubscribedText = document.createTextNode('');
  const close = buttonElement({ kind: 'primary', size: 'middle', className: styles.button });
  close.addEventListener('click', () => control.close());
  unsubscribedBox.append(unsubscribedText, close);

  /** @type {Step | undefined} */
  let watchedStep;
  /** @type {string | null} the text last passed to the token field, null while it is off the page */
  let drawnToken = null;
  render();

  function render() {
    if (destroyed) {
      return;
    }
    const { intl } = deps;
    const say = (/** @type {keyof typeof messages} */ name) => intl.formatMessage(messages[name]);
    const theme = props.theme;

    // a click elsewhere on the page while the reader is in the mailbox would otherwise drop the token step
    control.keepOpen(step === 'token');
    if (step !== watchedStep) {
      watchedStep = step;
      watch();
    }

    if (step === 'subscribed') {
      subscribedText.data = say(justSubscribed ? 'haveSubscribed' : 'subscribed');
      unsubscribe.className = buttonClassName({ kind: 'primary', size: 'middle', theme, className: styles.button });
      unsubscribe.textContent = say('unsubscribe');
      placeChildren(slot, [subscribedBox]);
      drawnToken = null;
      return;
    }
    if (step === 'unsubscribed') {
      unsubscribedText.data = say('haveUnsubscribed');
      close.className = buttonClassName({ kind: 'primary', size: 'middle', theme, className: styles.button });
      close.textContent = say('close');
      placeChildren(slot, [unsubscribedBox]);
      drawnToken = null;
      return;
    }

    form.className = classes(styles.root, theme === 'dark' ? styles.themeDark : styles.themeLight);
    /** @type {Node[]} */
    const children = [];
    if (step === 'email') {
      title.textContent = say('subscribeToReplies');
      description.textContent = say('subscribeToRepliesDescription');
      addressField.placeholder = say('email');
      if (addressField.value !== address) {
        addressField.value = address;
      }
      addressField.disabled = loading;
      children.push(title, description, addressField);
    } else {
      back.textContent = say('back');
      tokenField.textarea.placeholder = say('token');
      tokenField.textarea.disabled = loading;
      children.push(back, tokenField.textarea);
    }
    if (error !== null) {
      errorBox.textContent = error;
      children.push(errorBox);
    }
    submitLabel.data = say(step === 'email' ? 'submit' : 'subscribe');
    placeChildren(submit, [loading ? preloader : submitLabel]);
    submit.disabled = !EMAIL.test(address) || loading;
    children.push(submit);
    placeChildren(form, children);
    placeChildren(slot, [form]);
    // measured only when its text changes, as the preact field was, so a height the reader dragged
    // survives a redraw of the comment form
    if (step !== 'token') {
      drawnToken = null;
    } else if (token !== drawnToken) {
      drawnToken = token;
      tokenField.setValue(token);
    }
  }

  // asks the server whether the link in the email has confirmed the subscription: once on the
  // address step, and while the form waits for the token or has just unsubscribed, on each return
  // to the page and every few seconds until the token in the email expires
  function watch() {
    stopWatching?.();
    stopWatching = null;
    if (step === 'subscribed') {
      return;
    }
    const watching = step;
    let active = true;

    async function check() {
      if (document.hidden) {
        return;
      }
      const subscription = await deps.subscription().catch(() => null);
      // another tab may have signed in as someone else since the form opened
      if (!active || destroyed || !subscription?.address || subscription.user.id !== props.userId) {
        return;
      }
      // the token step waits for one address; a link to an earlier, perhaps mistyped one does not end
      // it, and pasting this step's token still replaces that address
      if (watching === 'token' && !sameAddress(subscription.address, address)) {
        return;
      }
      justSubscribed = watching !== 'email';
      deps.setSubscribed(true);
      step = 'subscribed';
      render();
    }

    if (watching === 'email') {
      void check();
      stopWatching = () => {
        active = false;
      };
      return;
    }
    // the reader may confirm in webmail in this browser and come back later than that, so the check on
    // returning to the page has no deadline; only the timer stops
    const deadline = Date.now() + CONFIRMATION_POLL_DURATION;
    const timer = setInterval(() => {
      if (Date.now() > deadline) {
        clearInterval(timer);
        return;
      }
      void check();
    }, CONFIRMATION_POLL_INTERVAL);
    document.addEventListener('visibilitychange', check);
    stopWatching = () => {
      active = false;
      clearInterval(timer);
      document.removeEventListener('visibilitychange', check);
    };
  }

  /** @param {string} [currentToken] */
  async function send(currentToken = token) {
    const sending = step;
    loading = true;
    error = null;
    render();
    try {
      if (sending === 'email') {
        let verification;
        try {
          verification = await deps.verify(address);
          justSubscribed = true;
        } catch (err) {
          // the status says the address is subscribed already; the code only picks the message
          if (/** @type {{ status?: number }} */ (err).status !== 409) {
            throw err;
          }
          verification = { address, updated: true };
        }
        if (verification.updated) {
          deps.setSubscribed(true);
          step = 'subscribed';
        } else {
          token = '';
          step = 'token';
        }
      } else if (sending === 'token') {
        await deps.confirm(currentToken);
        deps.setSubscribed(true);
        justSubscribed = true;
        step = 'subscribed';
      }
    } catch (err) {
      error = extractErrorMessageFromResponse(/** @type {any} */ (err), deps.intl);
    } finally {
      loading = false;
      render();
    }
  }

  /** @param {Event} evt */
  function onToken(evt) {
    evt.preventDefault();
    const value = tokenField.textarea.value;
    error = null;
    // a token still being pasted or typed is not sent, the server would refuse it
    const pasted = value.trim();
    try {
      if (pasted.length > 0 && isJwtExpired(pasted)) {
        error = deps.intl.formatMessage(messages.expiredToken);
      } else if (isCompleteToken(pasted)) {
        void send(pasted);
      }
    } catch {
      // a text that is no token at all waits for the reader to finish it
    }
    token = value;
    render();
  }

  function toEmailStep() {
    error = null;
    step = 'email';
    render();
  }

  async function leave() {
    loading = true;
    render();
    try {
      await deps.unsubscribe();
      deps.setSubscribed(false);
      step = 'unsubscribed';
    } catch (err) {
      error = extractErrorMessageFromResponse(/** @type {any} */ (err), deps.intl);
    } finally {
      loading = false;
      render();
    }
  }

  return {
    /**
     * @param {EmailSubscriptionProps} next
     * @param {EmailSubscriptionDeps} [nextDeps]
     */
    update(next, nextDeps = deps) {
      props = next;
      deps = nextDeps;
      render();
    },
    destroy() {
      destroyed = true;
      stopWatching?.();
      stopWatching = null;
      control.keepOpen(false);
    },
  };
}

export const messages = defineMessages({
  token: {
    id: 'subscribeByEmail.token',
    defaultMessage: 'Paste the token from the email, or click the link in it',
  },
  expiredToken: {
    id: 'token.expired',
    defaultMessage: 'Token is expired',
  },
  haveSubscribed: {
    id: 'subscribeByEmail.have-been-subscribed',
    defaultMessage: 'You have subscribed to email updates',
  },
  subscribed: {
    id: 'subscribeByEmail.subscribed',
    defaultMessage: 'You are subscribed to email updates',
  },
  submit: {
    id: 'subscribeByEmail.submit',
    defaultMessage: 'Submit',
  },
  subscribe: {
    id: 'subscribeByEmail.subscribe',
    defaultMessage: 'Subscribe',
  },
  subscribeByEmail: {
    id: 'subscribeByEmail.subscribe-by-email',
    defaultMessage: 'Subscribe by email',
  },
  onlyRegisteredUsers: {
    id: 'subscribeByEmail.only-registered-users',
    defaultMessage: 'Only available to registered users',
  },
  email: {
    id: 'subscribeByEmail.email',
    defaultMessage: 'Email',
  },
  subscribeToReplies: {
    id: 'subscribeByEmail.subscribe-to-replies',
    defaultMessage: 'Email subscription to replies',
  },
  subscribeToRepliesDescription: {
    id: 'subscribeByEmail.subscribe-to-replies-description',
    defaultMessage: 'Get an email about every reply to your comments and to the discussions under them.',
  },
  back: {
    id: 'subscribeByEmail.back',
    defaultMessage: 'Back',
  },
  unsubscribe: {
    id: 'subscribeByEmail.unsubscribe',
    defaultMessage: 'Unsubscribe',
  },
  haveUnsubscribed: {
    id: 'subscribeByEmail.have-been-unsubscribed',
    defaultMessage: 'You have unsubscribed from email updates',
  },
  close: {
    id: 'subscribeByEmail.close',
    defaultMessage: 'Close',
  },
});
