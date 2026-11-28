// The Telegram subscription in plain JavaScript: a dropdown that asks the server for a link to the
// bot, shows it with a Check button, and then the subscription with a way to end it and start again.
//
// The step and the link are kept in sessionStorage, per site and user, so closing and opening the
// panel, or reloading the page, does not ask for a new link while the server still keeps the old one.

import { readStored, writeStored } from '../../../hooks/session-storage.ts';
import { placeChildren } from '../../../common/dom.js';
import { defineMessages } from '../../../common/intl-message.ts';
import { siteId } from '../../../common/settings.ts';
import { extractErrorMessageFromResponse } from '../../../utils/errorUtils.ts';
import { buttonElement } from '../../button/button-element.js';
import { dropdownElement } from '../../dropdown/dropdown-element.js';
import { preloaderElement } from '../../preloader-element.js';
import { telegramLink } from '../../telegram/telegram-link-element.js';
import styles from './comment-form__subscribe-by-telegram.module.css';

/** how long the server keeps a subscription link, see telegramSubscribeCtrl in the backend */
const LINK_LIFETIME = 10 * 60 * 1000;

/**
 * the longest an open panel goes without comparing the link's deadline with the clock. One timer
 * set for the whole lifetime would not count the time a computer spends asleep
 */
const EXPIRY_CHECK_INTERVAL = 10 * 1000;

/** @typedef {'initial' | 'subscribed' | 'unsubscribed'} Step */
/** @typedef {{ token: string, bot: string, expires: number }} TelegramLinkState a link and the time the server forgets it */

/**
 * @typedef {object} TelegramSubscriptionProps
 * @property {string} userId
 * @property {boolean} anonymous  only registered users can subscribe
 * @property {'light' | 'dark'} theme
 */

/**
 * @typedef {object} TelegramSubscriptionDeps
 * @property {import('../../../common/intl-message.ts').IntlShape} intl
 * @property {() => Promise<{ token: string, bot: string }>} subscribe  asks for a link to the bot
 * @property {(token: string) => Promise<{ updated: boolean }>} check  whether the reader started the bot
 * @property {() => Promise<unknown>} unsubscribe
 * @property {Pick<Storage, 'getItem' | 'setItem'>} storage
 */

/**
 * @param {HTMLElement} root  an empty div; its attributes and children are this component's
 * @param {TelegramSubscriptionProps} initial
 * @param {TelegramSubscriptionDeps} initialDeps
 */
export function mountSubscribeByTelegram(root, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  /** @type {ReturnType<typeof telegramForm> | null} */
  let form = null;

  const dropdown = dropdownElement(root, dropdownProps(), (items) => {
    form = telegramForm(items, props, deps);
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
      title: intl.formatMessage(messages.telegram),
      theme: props.theme,
      disabled: props.anonymous,
      buttonTitle: intl.formatMessage(props.anonymous ? messages.onlyRegisteredUsers : messages.subscribeByTelegram),
    };
  }

  return {
    /**
     * @param {TelegramSubscriptionProps} next
     * @param {TelegramSubscriptionDeps} [nextDeps]
     */
    update(next, nextDeps = deps) {
      props = next;
      deps = nextDeps;
      dropdown.update(dropdownProps());
      form?.update(deps);
    },
    destroy() {
      dropdown.destroy();
    },
  };
}

/**
 * The content of the open dropdown.
 *
 * @param {HTMLElement} slot  the element whose children the form sets
 * @param {TelegramSubscriptionProps} props
 * @param {TelegramSubscriptionDeps} initialDeps
 */
function telegramForm(slot, props, initialDeps) {
  let deps = initialDeps;
  // sessionStorage outlives a sign-out, and the server issues a link to one user of one site, so
  // the state is kept per site and user and the next user of the tab gets a link of their own
  const owner = `${encodeURIComponent(siteId)}/${encodeURIComponent(props.userId)}`;
  const stepKey = `telegram-subscription-step/${owner}`;
  const linkKey = `telegram-subscription-telegram/${owner}`;
  /** @type {Step} */
  let step = /** @type {Step} */ (readStored(deps.storage, stepKey, 'initial'));
  /** @type {TelegramLinkState | null} */
  let telegram = /** @type {TelegramLinkState | null} */ (readStored(deps.storage, linkKey, null));
  let loading = false;
  /** @type {string | null} */
  let error = null;
  let destroyed = false;
  /** @type {ReturnType<typeof setTimeout> | undefined} */
  let expiryTimer;
  /** @type {string | undefined} the link the expiry timer watches */
  let watchedToken;

  const root = document.createElement('div');
  root.className = styles.root;
  const preloader = preloaderElement(styles.preloader);
  const errorBox = document.createElement('div');
  errorBox.className = 'auth-error';
  /** @type {ReturnType<typeof telegramLink> | null} */
  let link = null;
  const note = document.createElement('p');
  const action = buttonElement({ kind: 'primary', size: 'large' });
  action.addEventListener('click', () => void (step === 'subscribed' ? leave() : rejoin()));
  placeChildren(slot, [root]);

  render();
  // a stored link that has expired was dropped by the first render, which asked for a new one
  if (!telegram && !loading) {
    void fetchLink();
  }

  /** @param {Step} next */
  function setStep(next) {
    step = next;
    writeStored(deps.storage, stepKey, step);
  }

  /** @param {TelegramLinkState | null} next */
  function setLink(next) {
    telegram = next;
    writeStored(deps.storage, linkKey, telegram);
  }

  function render() {
    if (destroyed) {
      return;
    }
    const { intl } = deps;
    /** @type {Node[]} */
    const children = [];
    if (loading) {
      children.push(preloader);
    } else {
      if (error && (!telegram || step !== 'initial')) {
        errorBox.textContent = error;
        children.push(errorBox);
      }
      if (telegram && step === 'initial') {
        const linkProps = { bot: telegram.bot, token: telegram.token, errorMessage: error, onSubmit: checkLink };
        if (link) {
          link.update(linkProps, intl);
        } else {
          link = telegramLink(linkProps, intl);
        }
        children.push(...link.nodes());
      } else {
        link = null;
      }
      if (step === 'subscribed' || step === 'unsubscribed') {
        note.textContent = intl.formatMessage(
          step === 'subscribed' ? messages.haveSubscribed : messages.haveUnsubscribed
        );
        action.textContent = intl.formatMessage(step === 'subscribed' ? messages.unsubscribe : messages.resubscribe);
        children.push(note, action);
      }
    }
    placeChildren(root, children);
    watchExpiry();
  }

  // a link the server has forgotten is dropped, at once when it is already that old and by the
  // timer when it gets there with the panel open, and a new one is asked for. An error on screen
  // came from checking the dropped link, so it goes with it
  function watchExpiry() {
    const shown = telegram && step === 'initial' ? telegram : null;
    if (shown?.token === watchedToken && (shown !== null) === (expiryTimer !== undefined)) {
      return;
    }
    clearTimeout(expiryTimer);
    expiryTimer = undefined;
    watchedToken = shown?.token;
    if (!shown) {
      return;
    }
    const dropIfExpired = () => {
      if (shown.expires > Date.now()) {
        return false;
      }
      expiryTimer = undefined;
      watchedToken = undefined;
      error = null;
      setLink(null);
      render();
      void fetchLink();
      return true;
    };
    if (dropIfExpired()) {
      return;
    }
    const checkLater = () => {
      expiryTimer = setTimeout(
        () => {
          if (!dropIfExpired()) {
            checkLater();
          }
        },
        Math.min(EXPIRY_CHECK_INTERVAL, shown.expires - Date.now())
      );
    };
    checkLater();
  }

  /**
   * @param {boolean} [showLoading]
   * @returns {Promise<boolean>} whether a link came
   */
  async function fetchLink(showLoading = true) {
    if (showLoading) {
      loading = true;
      render();
    }
    try {
      // counted from before the request, so the link is never shown past the server's own deadline
      const expires = Date.now() + LINK_LIFETIME;
      const { token, bot } = await deps.subscribe();
      if (destroyed) {
        return false;
      }
      setLink({ token, bot, expires });
      return true;
    } catch (err) {
      if (destroyed) {
        return false;
      }
      // a 409 means the reader is subscribed already, whatever code picks its message
      const refusal = /** @type {{ error?: string, status?: number }} */ (err);
      if (refusal.error === 'already subscribed' || refusal.status === 409) {
        setStep('subscribed');
        return false;
      }
      error = extractErrorMessageFromResponse(/** @type {any} */ (err), deps.intl);
      return false;
    } finally {
      if (showLoading) {
        loading = false;
      }
      render();
    }
  }

  function checkLink() {
    if (!telegram) {
      return;
    }
    const { token } = telegram;
    loading = true;
    render();
    deps
      .check(token)
      .then(({ updated }) => {
        error = null;
        if (updated) {
          setStep('subscribed');
        }
      })
      .catch((err) => {
        // a refusal of a link that has been replaced meanwhile says nothing about the one on screen
        if (telegram?.token === token) {
          error = extractErrorMessageFromResponse(err, deps.intl);
        }
      })
      .finally(() => {
        loading = false;
        render();
      });
  }

  async function leave() {
    loading = true;
    render();
    try {
      await deps.unsubscribe();
      error = null;
      setStep('unsubscribed');
    } catch (err) {
      error = extractErrorMessageFromResponse(/** @type {any} */ (err), deps.intl);
    } finally {
      loading = false;
      render();
    }
  }

  async function rejoin() {
    error = null;
    render();
    if (await fetchLink(false)) {
      setStep('initial');
      render();
    }
  }

  return {
    /** @param {TelegramSubscriptionDeps} next */
    update(next) {
      deps = next;
      render();
    },
    destroy() {
      destroyed = true;
      clearTimeout(expiryTimer);
      expiryTimer = undefined;
    },
  };
}

const messages = defineMessages({
  haveSubscribed: {
    id: 'subscribeByTelegram.have-been-subscribed',
    defaultMessage: 'You have subscribed to Telegram updates',
  },
  haveUnsubscribed: {
    id: 'subscribeByTelegram.have-been-unsubscribed',
    defaultMessage: 'You have unsubscribed from Telegram updates',
  },
  unsubscribe: {
    id: 'subscribeByTelegram.unsubscribe',
    defaultMessage: 'Unsubscribe',
  },
  resubscribe: {
    id: 'subscribeByTelegram.resubscribe',
    defaultMessage: 'Resubscribe',
  },
  subscribeByTelegram: {
    id: 'subscribeByTelegram.subscribe-by-telegram',
    defaultMessage: 'Subscribe by Telegram',
  },
  onlyRegisteredUsers: {
    id: 'subscribeByTelegram.only-registered-users',
    defaultMessage: 'Only available to registered users',
  },
  telegram: {
    id: 'subscribeByTelegram.telegram',
    defaultMessage: 'Telegram',
  },
});
