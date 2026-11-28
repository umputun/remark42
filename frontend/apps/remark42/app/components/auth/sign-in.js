// The sign-in button and its dropdown in plain JavaScript: the OAuth buttons, the anonymous and
// email forms with their tabs, the token step and the Telegram step. mountSignIn fills an element
// the caller owns; update() redraws it in place and destroy() stops every timer and listener.
//
// The dropdown and the form inside it are built when the dropdown opens and dropped when it
// closes, and the token and Telegram steps each time they are entered, as the preact component
// mounted them; within one of them every node stays, so a field keeps its focus and caret.

import { classes, placeChildren, svgElement, SVG_NS } from '../../common/dom.js';
import { element } from '../../common/element.js';
import { getCookie } from '../../common/cookies.ts';
import { XSRF_COOKIE } from '../../common/fetcher.ts';
import { errorMessages, RequestError } from '../../utils/errorUtils.ts';
import { isObject } from '../../utils/is-object.ts';
import { parseMessage, updateIframeHeight } from '../../utils/post-message.ts';
import { arrowIcon } from '../icons/arrow-icon.js';
import { crossIcon } from '../icons/cross-icon.js';
import { inputElement } from '../input/input-element.js';
import { spinnerElement } from '../spinner/spinner-element.js';
import { telegramLink } from '../telegram/telegram-link-element.js';
import { textareaElement } from '../textarea-element.js';
import { buttonClassName, buttonElement, suffixElement } from './components/button-element.js';
import { oauthList } from './components/oauth-list.js';
import { messages } from './auth.messages.ts';
import { POPUP_BLOCKED, prepareHandoff } from './auth.handoff.ts';
import { getProviders, getTokenInvalidReason, isCompleteToken, persistEmail } from './auth.utils.ts';
import inputStyles from '../input/input.module.css';
import styles from './auth.module.css';

const USERNAME_PATTERN = String.raw`\s*[\p{L}\d_.\-][\p{L}\d _.\-]{1,62}[\p{L}\d_.\-]\s*`;
const EMAIL_USERNAME_PATTERN = String.raw`\s*[\p{L}\d_\-][\p{L}\d _\-]{1,62}[\p{L}\d_\-]\s*`;

/**
 * @typedef {import('../../common/types.ts').User} User
 */

/**
 * The server calls the form makes, passed in so the component carries no fetch of its own.
 *
 * @typedef {object} SignInApi
 * @property {(username: string) => Promise<User>} anonymousSignin
 * @property {(email: string, username: string) => Promise<unknown>} emailSignin
 * @property {(token: string) => Promise<User>} verifyEmailSignin
 * @property {(url: string) => Promise<User | null>} oauthSignin
 * @property {() => Promise<{ bot: string, token: string }>} getTelegramSigninParams
 * @property {(token: string) => Promise<User>} verifyTelegramSignin
 * @property {() => Promise<User>} getSignedInUser  the user the current cookies belong to
 * @property {() => Promise<User | null>} getUser  the session's user, or null
 */

/**
 * @typedef {object} SignInDeps
 * @property {import('../../common/intl-message.ts').IntlShape} intl
 * @property {import('../../common/types.ts').Theme} theme
 * @property {(user: User) => void} signin  stores the user who signed in
 * @property {SignInApi} api
 */

/**
 * @param {HTMLElement} root  an empty element; its class and children are this component's
 * @param {SignInDeps} initialDeps
 */
export function mountSignIn(root, initialDeps) {
  let deps = initialDeps;
  const [oauthProviders, formProviders] = getProviders();

  let loading = false;
  /** @type {string | undefined} the step shown: a form provider, 'token' or 'telegram' */
  let view = formProviders[0];
  // what the reader has typed and the tab it is on stay while the dropdown is closed and opened
  // again, and the address while the email field is away on another tab or step
  /** @type {string | undefined} */
  let formView = formProviders[0];
  let username = '';
  let email = '';
  /** @type {string | number | null} why the last attempt failed */
  let reason = null;
  let shown = false;
  /** @type {{ bot: string, token: string } | null} */
  let telegramParams = null;
  let sentTo = '';
  let verifying = false;
  let signedIn = false;
  let destroyed = false;

  root.className = classes('auth', styles.root);
  const toggle = buttonElement({ className: 'auth-button' });
  const toggleLabel = document.createTextNode('');
  toggle.append(toggleLabel, suffixElement(arrowIcon()));
  toggle.addEventListener('click', (evt) => {
    evt.preventDefault();
    setShown(!shown, evt);
  });
  root.append(toggle);

  /** @type {ReturnType<typeof openDropdown> | null} */
  let dropdown = null;
  /** @type {(() => void) | null} stops watching the cookie on the token step */
  let stopWatching = null;

  render();

  /**
   * @param {boolean} next
   * @param {Event} [cause]  the click that opened it, which must not also count as a click outside
   */
  function setShown(next, cause) {
    shown = next;
    render(cause);
  }

  function resetView() {
    view = formView;
    reason = null;
  }

  /** @param {unknown} err */
  function setError(err) {
    if (err === null) {
      reason = null;
    } else if (typeof err === 'string') {
      reason = err;
    } else if (err instanceof RequestError || isObject(err)) {
      reason = /** @type {{ error: string }} */ (err).error;
    } else if (err instanceof Error) {
      reason = err.message;
    } else {
      reason = 0;
    }
  }

  /** the text of the error to show, if any */
  function errorText() {
    const { intl } = deps;
    /** @type {string | number | null} */
    let text = reason;
    if (typeof reason === 'string' && messages[reason]) {
      text = intl.formatMessage(messages[reason]);
    }
    if (reason !== null && errorMessages[reason]) {
      text = intl.formatMessage(errorMessages[reason]);
    }
    if (typeof text === 'number') {
      console.error('Wrong error message', text);
      text = null;
    }
    return text;
  }

  /**
   * The dropdown with the parts that stay while it is open; the steps come and go inside it.
   *
   * @param {Event} [cause]
   */
  function openDropdown(cause) {
    const { intl } = deps;
    const box = element('div', { class: classes('auth-dropdown', styles.dropdown) });
    const noProviders = element('div', { class: classes('auth-error', styles.error) });
    const form = element('form', { class: classes('auth-form', styles.form) });
    form.addEventListener('submit', handleSubmit);

    const oauthTitle = element('h5', { class: classes('auth-form-title', styles.title) });
    const oauth = oauthList(oauthProviders, { intl, theme: deps.theme, onOauthClick: handleOauthClick });
    if (oauthProviders.length > 0) {
      // ready before the click, which has to open the sign-in window at once
      prepareHandoff();
    }
    const divider = element('div', { class: classes('auth-divider', styles.divider), title: '' });
    const formTitle = element('h5', { class: classes('auth-form-title', styles.title) });
    const tabs = element('div', { class: classes('auth-tabs', styles.tabs) });
    const radios = formProviders.map((provider) => {
      const radio = element('input', {
        class: styles.radio,
        type: 'radio',
        id: `form-provider-${provider}`,
        name: 'form-provider',
        value: provider,
      });
      radio.addEventListener('change', handleProviderChange);
      const label = element('label', {
        class: classes('auth-tabs-item', styles.provider),
        for: `form-provider-${provider}`,
      });
      label.textContent = provider.slice(0, 6);
      tabs.append(radio, label);
      return { provider, radio };
    });

    const nameField = inputElement(
      { className: 'auth-input-username' },
      { required: true, name: 'username', pattern: '', title: '', placeholder: '', dir: 'auto' }
    );
    nameField.addEventListener('input', () => {
      username = nameField.value;
    });
    nameField.addEventListener('blur', () => {
      nameField.value = nameField.value.trim();
      username = nameField.value;
    });
    const nameRow = element('div', { class: classes('auth-row', styles.row) }, [nameField]);

    const emailField = inputElement(
      { className: 'auth-input-email', type: 'email' },
      { required: true, name: 'email', placeholder: '', dir: 'auto' }
    );
    emailField.addEventListener('input', () => {
      email = emailField.value;
    });
    const emailRow = element('div', { class: classes('auth-row', styles.row) }, [emailField]);

    const error = element('div', { class: classes('auth-error', styles.error) });
    const submit = buttonElement({ className: 'auth-submit' });
    submit.type = 'submit';
    const submitLabel = document.createTextNode('');
    const spinner = spinnerElement(intl);

    // only the embedding page may close the dropdown, and only by reporting a click outside the
    // widget. The parent also posts hash, title and theme messages, and extensions post their
    // own, none of which should interrupt an active sign-in
    /** @param {MessageEvent} evt */
    function onMessage(evt) {
      if (evt.source !== window.parent) {
        return;
      }
      if (!parseMessage(evt).clickOutside || closingDisabled()) {
        return;
      }
      setShown(false);
    }

    let clickInside = false;
    function onClickInside() {
      clickInside = true;
    }
    /** @param {Event} evt */
    function onClickOutside(evt) {
      const inside = clickInside;
      clickInside = false;
      if (evt === cause || closingDisabled() || inside) {
        return;
      }
      setShown(false);
    }

    box.addEventListener('click', onClickInside, { capture: true });
    document.addEventListener('click', onClickOutside);
    window.addEventListener('message', onMessage);
    const observer = new ResizeObserver(() => updateIframeHeight(box));

    /** @type {{ row: HTMLElement, field: ReturnType<typeof textareaElement>, measured: boolean } | null} */
    let token = null;
    /** @type {ReturnType<typeof telegramLink> | null} */
    let telegram = null;
    let measured = false;

    function makeBackRow() {
      const back = buttonElement({ className: 'auth-back-button', size: 'xs', kind: 'transparent' });
      back.append(
        svgElement(
          'svg',
          {
            class: styles.backButtonArrow,
            width: '14',
            height: '14',
            viewBox: '0 0 14 14',
            fill: 'none',
            xmlns: SVG_NS,
          },
          [
            svgElement('path', {
              d: 'M8.75 3L5 7.25L9 11',
              stroke: 'currentColor',
              'stroke-width': '2',
              'stroke-linecap': 'round',
              'stroke-linejoin': 'round',
            }),
          ]
        ),
        document.createTextNode('')
      );
      back.addEventListener('click', (evt) => {
        evt.preventDefault();
        if (view === 'token') {
          // the email step, whichever tab the reader came from
          formView = 'email';
          view = 'email';
          reason = null;
        } else {
          resetView();
        }
        render();
      });
      const close = element('button', { class: classes('auth-close-button', styles.closeButton), title: '' }, [
        crossIcon(),
      ]);
      close.addEventListener('click', (evt) => {
        evt.preventDefault();
        resetView();
        setShown(false);
      });
      const row = element('div', { class: classes('auth-row', styles.row) }, [
        element('div', { class: styles.backButton }, [back]),
        close,
      ]);
      return { row, back, close };
    }

    /** @type {ReturnType<typeof makeBackRow> | null} the back and close row of the token or Telegram step */
    let backParts = null;

    function draw() {
      const { intl, theme } = deps;
      const message = errorText();
      const hasOAuth = oauthProviders.length > 0;
      const hasForm = formProviders.length > 0;

      noProviders.textContent = intl.formatMessage(messages.noProviders);
      placeChildren(box, !hasOAuth && !hasForm ? [noProviders, form] : [form]);

      const stepWithBack = (view === 'telegram' && telegramParams !== null) || view === 'token';
      if (stepWithBack && !backParts) {
        backParts = makeBackRow();
      } else if (!stepWithBack) {
        backParts = null;
      }
      if (backParts) {
        /** @type {Text} */ (backParts.back.lastChild).data = intl.formatMessage(messages.back);
        backParts.close.title = 'Close sign-in dropdown';
      }

      if (view === 'token' && !token) {
        const field = textareaElement(
          {
            name: 'token',
            class: classes('auth-token-textarea', inputStyles.input, styles.textarea),
            placeholder: '',
            disabled: false,
          },
          { onInput: handleTokenInput }
        );
        token = {
          row: element('div', { class: classes('auth-row', styles.row) }, [field.textarea]),
          field,
          measured: false,
        };
      } else if (view !== 'token') {
        token = null;
      }

      if (view === 'telegram' && telegramParams !== null) {
        const props = { ...telegramParams, errorMessage: message, onSubmit: handleTelegramSubmit };
        if (telegram) {
          telegram.update(props, intl);
        } else {
          telegram = telegramLink(props, intl);
        }
      } else {
        telegram = null;
      }

      error.textContent = message ?? '';
      submit.disabled = loading;
      submitLabel.data = intl.formatMessage(messages.submit);
      spinner.setAttribute('aria-label', intl.formatMessage(messages.loading));
      placeChildren(submit, [loading ? spinner : submitLabel]);
      const footer = message ? [error, submit] : [submit];

      /** @type {Node[]} */
      let children;
      if (telegram && backParts) {
        children = [backParts.row, ...telegram.nodes()];
      } else if (token && backParts) {
        token.field.textarea.placeholder = intl.formatMessage(messages.token);
        token.field.textarea.disabled = loading;
        children = [backParts.row, token.row, ...footer];
      } else {
        children = [];
        if (hasOAuth) {
          oauthTitle.textContent = intl.formatMessage(messages.oauthSource);
          oauth.update({ intl, theme, onOauthClick: handleOauthClick });
          children.push(oauthTitle, oauth.root);
          // with a form below, its footer shows the error
          if (message && !hasForm) {
            children.push(error);
          }
        }
        if (hasOAuth && hasForm) {
          divider.title = intl.formatMessage(messages.or);
          children.push(divider);
        }
        if (hasForm) {
          if (formProviders.length === 1) {
            formTitle.textContent = formProviders[0];
            children.push(formTitle);
          } else {
            for (const { provider, radio } of radios) {
              radio.checked = provider === view;
            }
            children.push(tabs);
          }
          const emailStep = view === 'email';
          nameField.pattern = emailStep ? EMAIL_USERNAME_PATTERN : USERNAME_PATTERN;
          nameField.title = intl.formatMessage(
            emailStep ? messages.emailUsernameRestriction : messages.usernameRestriction
          );
          nameField.placeholder = intl.formatMessage(messages.username);
          nameField.disabled = loading;
          if (nameField.value !== username) {
            nameField.value = username;
          }
          children.push(nameRow);
          if (emailStep) {
            emailField.placeholder = intl.formatMessage(messages.emailAddress);
            emailField.disabled = loading;
            if (emailField.value !== email) {
              emailField.value = email;
            }
            children.push(emailRow);
          }
          children.push(...footer);
        }
      }
      placeChildren(form, children);

      // once, when the field is new: its input listener measures it after that, and a redraw for
      // anything else leaves the height the reader dragged it to
      if (token && !token.measured) {
        token.measured = true;
        token.field.resize();
      }
      if (!measured) {
        measured = true;
        updateIframeHeight(box);
        observer.observe(box);
      }
    }

    return {
      box,
      draw,
      close() {
        box.removeEventListener('click', onClickInside, { capture: true });
        document.removeEventListener('click', onClickOutside);
        window.removeEventListener('message', onMessage);
        observer.disconnect();
        document.body.style.removeProperty('min-height');
        box.remove();
        // the panel is positioned absolutely, so closing it resizes no box an observer watches:
        // without a report of its own the parent keeps the open panel's height, a hole under the
        // widget for as long as the reader stays on the page
        updateIframeHeight();
      },
    };
  }

  /** the token and Telegram steps stay open whatever is clicked outside them */
  function closingDisabled() {
    return view === 'token' || view === 'telegram';
  }

  /** @param {Event} [cause] */
  function render(cause) {
    if (destroyed) {
      return;
    }
    toggle.className = buttonClassName({ className: 'auth-button', selected: shown });
    toggleLabel.data = deps.intl.formatMessage(messages.signin);

    if (shown && !dropdown) {
      dropdown = openDropdown(cause);
      root.append(dropdown.box);
    } else if (!shown && dropdown) {
      dropdown.close();
      dropdown = null;
    }
    dropdown?.draw();

    watchToken(view === 'token');
  }

  /**
   * The link in the email signs the browser in on a page of its own, and this frame shares that
   * page's cookies wherever the browser lets it. A new XSRF cookie is the sign it happened:
   * reading the cookie costs no request, so the user is loaded only once it has changed. A browser
   * keeps a hidden tab's timers slow, so the check also runs when the reader comes back to it.
   *
   * @param {boolean} on
   */
  function watchToken(on) {
    if (!on) {
      stopWatching?.();
      stopWatching = null;
      return;
    }
    if (stopWatching) {
      return;
    }

    let seen = getCookie(XSRF_COOKIE);

    async function check() {
      const current = getCookie(XSRF_COOKIE);

      if (!current || current === seen || verifying || signedIn) {
        return;
      }

      const previous = seen;
      // a cookie that does not lead to a user is not looked at again
      seen = current;
      verifying = true;

      try {
        const user = await deps.api.getSignedInUser();
        signedIn = true;
        deps.signin(user);
      } catch (e) {
        // a refusal is the answer for this cookie, and the token from the email still works. A
        // request that did not get through, on the network or at the rate limit, is made again on
        // the next check
        if (!(e instanceof RequestError && (e.code === 401 || e.code === 403))) {
          seen = previous;
        }
      } finally {
        verifying = false;
      }
    }

    const timer = setInterval(check, 1000);
    document.addEventListener('visibilitychange', check);
    window.addEventListener('focus', check);
    stopWatching = () => {
      clearInterval(timer);
      document.removeEventListener('visibilitychange', check);
      window.removeEventListener('focus', check);
    };
  }

  /** @param {MouseEvent} evt */
  async function handleOauthClick(evt) {
    evt.preventDefault();
    const { href, dataset } = /** @type {HTMLAnchorElement} */ (evt.currentTarget);

    if (dataset.providerName?.toLowerCase() === 'telegram') {
      try {
        telegramParams = await deps.api.getTelegramSigninParams();
      } catch (e) {
        setError(e);
        render();
        return;
      }
      view = 'telegram';
      reason = null;
      render();
      return;
    }

    try {
      const user = await deps.api.oauthSignin(href);
      if (user === null) {
        return;
      }
      deps.signin(user);
    } catch (e) {
      // by its code, which has a message of its own in every language, and not by the English text
      // it was thrown with
      const code = e instanceof RequestError ? e.code : undefined;
      setError(code === POPUP_BLOCKED || code === 0 || code === 'fetch-error' ? code : e);
      render();
    }
  }

  /** @param {Event} evt */
  function handleProviderChange(evt) {
    const value = /** @type {HTMLInputElement} */ (evt.currentTarget).value;
    formView = value;
    view = value;
    reason = null;
    render();
  }

  /**
   * A confirmation token signs in once, so a second request with the same one is refused, and
   * the events that can ask twice come before the form is disabled.
   *
   * @param {string} value
   */
  async function verifyToken(value) {
    if (verifying || signedIn) {
      return;
    }
    verifying = true;

    try {
      /** @type {User | null} */
      let user;
      try {
        user = await deps.api.verifyEmailSignin(value);
        persistEmail(sentTo);
      } catch (e) {
        // a token refused as used may be the one that signed this browser in: its first request
        // went through and the user was not loaded after it. The session is what tells. It can
        // also belong to another account signed in from another tab, so the address typed here
        // is not kept for it
        user = await deps.api.getUser();
        if (user === null) {
          throw e;
        }
      }
      signedIn = true;
      deps.signin(user);
    } finally {
      verifying = false;
    }
  }

  /**
   * Signs in as soon as the field holds a whole token, the way the subscription form does. Text
   * that is not one yet is left alone, since it may still be typed, and the submit button
   * reports it.
   *
   * @param {Event} evt
   */
  async function handleTokenInput(evt) {
    const value = /** @type {HTMLTextAreaElement} */ (evt.currentTarget).value.trim();

    if (verifying) {
      return;
    }
    reason = null;

    if (!isCompleteToken(value)) {
      render();
      return;
    }

    const invalid = getTokenInvalidReason(value);
    if (invalid) {
      reason = invalid;
      render();
      return;
    }

    loading = true;
    render();
    try {
      await verifyToken(value);
    } catch (e) {
      setError(e);
    }
    loading = false;
    render();
  }

  /** @param {SubmitEvent} evt */
  async function handleSubmit(evt) {
    const data = new FormData(/** @type {HTMLFormElement} */ (evt.target));

    evt.preventDefault();
    loading = true;
    reason = null;
    render();

    try {
      switch (view) {
        case 'anonymous': {
          // the field is trimmed on blur, which pressing Enter skips, and the server refuses a
          // space at either end
          const name = /** @type {string} */ (data.get('username')).trim();
          const user = await deps.api.anonymousSignin(name);
          deps.signin(user);
          break;
        }
        case 'email': {
          const address = /** @type {string} */ (data.get('email'));
          const name = /** @type {string} */ (data.get('username')).trim();
          sentTo = address;
          await deps.api.emailSignin(address, name);
          view = 'token';
          break;
        }
        case 'token': {
          const value = /** @type {string} */ (data.get('token')).trim();
          const invalid = getTokenInvalidReason(value);
          if (invalid) {
            reason = invalid;
            break;
          }
          await verifyToken(value);
          break;
        }
        default:
          // the Telegram step submits through its own button, which keeps the form from sending
          break;
      }
    } catch (e) {
      setError(e);
    }

    loading = false;
    render();
  }

  /** @param {Event} evt */
  async function handleTelegramSubmit(evt) {
    evt.preventDefault();
    loading = true;
    reason = null;
    render();

    if (telegramParams === null) {
      try {
        telegramParams = await deps.api.getTelegramSigninParams();
      } catch (e) {
        setError(e);
        render();
        return;
      }
    }

    try {
      const user = await deps.api.verifyTelegramSignin(telegramParams.token);
      deps.signin(user);
    } catch (e) {
      setError(e);
    }
    loading = false;
    render();
  }

  return {
    /** @param {SignInDeps} next */
    update(next) {
      deps = next;
      render();
    },
    destroy() {
      if (destroyed) {
        return;
      }
      watchToken(false);
      dropdown?.close();
      dropdown = null;
      destroyed = true;
    },
  };
}
