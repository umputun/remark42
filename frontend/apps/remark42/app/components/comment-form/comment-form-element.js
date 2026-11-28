// The comment form in plain JavaScript: the markdown toolbar, the field with its emoji completion,
// the errors, the buttons, the sign-in form for a guest, the subscriptions and the preview. It draws
// the markup the preact class component drew and updates its nodes in place, so the field keeps
// its focus, caret and undo history while the form redraws.

import { classes, placeChildren } from '../../common/dom.js';
import { isEmailSubscription, isRssSubscription, isTelegramSubscription, pageTitle } from '../../common/settings.ts';
import { StaticStore } from '../../common/static-store.ts';
import { errorMessages, extractErrorMessageFromResponse } from '../../utils/errorUtils.ts';
import { isUserAnonymous } from '../../utils/isUserAnonymous.ts';
import { replaceSelection } from '../../utils/replaceSelection.ts';
import { mountSignIn } from '../auth/sign-in.js';
import { buttonClassName, buttonElement } from '../button/button-element.js';
import { textareaElement } from '../textarea-element.js';
import { mountSubscribeByEmail } from './__subscribe-by-email/subscribe-by-email-element.js';
import { mountSubscribeByRss } from './__subscribe-by-rss/subscribe-by-rss-element.js';
import { mountSubscribeByTelegram } from './__subscribe-by-telegram/subscribe-by-telegram-element.js';
import { messages } from './comment-form.messages.ts';
import { getPersistedComment, removePersistedComment, updatePersistedComments } from './comment-form.persist.ts';
import { emojiCompletion } from './emoji-completion.js';
import { markdownToolbar } from './markdown-toolbar-element.js';
import styles from './comment-form.module.css';

const IMAGE_MIME = /image\//i;
/** between the uploads of several images, so the server's rate limiter lets them all through */
const UPLOAD_DELAY = 5000;
/** a wait for the field to show new text before its caret is placed in it */
const SETTLE_DELAY = 100;

/** @typedef {import('../../common/types.ts').User} User */
/** @typedef {import('../../common/types.ts').Theme} Theme */
/** @typedef {import('../../common/types.ts').Image} Image */
/** @typedef {import('../../common/intl-message.ts').IntlShape} IntlShape */

/**
 * @typedef {object} CommentFormProps
 * @property {string} id  the comment, reply or page the text is kept under while it is a draft
 * @property {string} textareaId  the field's id
 * @property {User | null} user
 * @property {string} [errorMessage]
 * @property {string} [value]  the text to start from, as when editing
 * @property {string | string[]} [mix]
 * @property {'main' | 'edit' | 'reply'} [mode]
 * @property {boolean} [isReply]  in an edit, whether the comment is a reply, which sets the placeholder
 * @property {Theme} theme
 * @property {boolean} [autofocus]
 */

/**
 * @typedef {object} CommentFormDeps
 * @property {IntlShape} intl
 * @property {(text: string, pageTitle: string) => Promise<void>} onSubmit
 * @property {(text: string) => Promise<string>} getPreview
 * @property {() => void} [onCancel]
 * @property {(image: File) => Promise<Image>} [uploadImage]
 * @property {() => Promise<Record<string, string>>} loadEmoji
 * @property {Theme} theme  the widget's theme, which the parts that read it from the store draw with
 * @property {import('../auth/sign-in.js').SignInDeps} signIn
 * @property {{ props: import('./__subscribe-by-rss/subscribe-by-rss-element.js').RssProps, intl: IntlShape }} rss
 * @property {{
 *   props: import('./__subscribe-by-email/subscribe-by-email-element.js').EmailSubscriptionProps,
 *   deps: import('./__subscribe-by-email/subscribe-by-email-element.js').EmailSubscriptionDeps,
 * }} email
 * @property {{
 *   props: import('./__subscribe-by-telegram/subscribe-by-telegram-element.js').TelegramSubscriptionProps,
 *   deps: import('./__subscribe-by-telegram/subscribe-by-telegram-element.js').TelegramSubscriptionDeps,
 * }} telegram
 */

/**
 * @typedef {object} FormState
 * @property {string | null} preview
 * @property {boolean} isErrorShown
 * @property {string | null} errorMessage  several errors are separate lines
 * @property {boolean} errorLock  typing leaves the error in place
 * @property {boolean} isDisabled
 * @property {string} text
 * @property {string | null} buttonText  in place of the submit button's label
 */

/**
 * A part mounted while it is shown and destroyed once it is not.
 *
 * @template {{ update: (...args: any[]) => void, destroy: () => void }} T
 * @param {(root: HTMLDivElement) => T} mount
 */
function mounted(mount) {
  /** @type {{ root: HTMLDivElement, part: T } | null} */
  let current = null;
  return {
    /**
     * The part's root when it is shown, after it is mounted or brought up to date.
     *
     * @param {boolean} shown
     * @param {(part: T) => void} update
     */
    place(shown, update) {
      if (!shown) {
        current?.part.destroy();
        current = null;
        return null;
      }
      if (current) {
        update(current.part);
      } else {
        const root = document.createElement('div');
        current = { root, part: mount(root) };
      }
      return current.root;
    },
    destroy() {
      current?.part.destroy();
      current = null;
    },
  };
}

/**
 * @param {HTMLElement} root  an empty div; its attributes and children are this component's
 * @param {CommentFormProps} initial
 * @param {CommentFormDeps} initialDeps
 */
export function mountCommentForm(root, initial, initialDeps) {
  let props = initial;
  let deps = initialDeps;
  let destroyed = false;
  /** @type {Set<ReturnType<typeof setTimeout>>} the waits of an upload in progress, cleared when the form goes */
  const timers = new Set();

  /**
   * Resolves after ms, or never once the form is destroyed, which ends the upload waiting on it.
   *
   * @param {number} ms
   * @returns {Promise<void>}
   */
  function wait(ms) {
    return new Promise((resolve) => {
      const timer = setTimeout(() => {
        timers.delete(timer);
        resolve();
      }, ms);
      timers.add(timer);
    });
  }

  /** @type {FormState} */
  const state = {
    preview: null,
    isErrorShown: false,
    errorMessage: null,
    errorLock: false,
    isDisabled: false,
    text: props.value ?? getPersistedComment(props.id) ?? '',
    buttonText: null,
  };

  /** @param {Partial<FormState>} patch */
  function setState(patch) {
    if (destroyed) {
      return;
    }
    Object.assign(state, patch);
    render();
  }

  root.setAttribute('data-nosnippet', 'true');
  const form = document.createElement('form');
  form.addEventListener('submit', (evt) => send(evt));
  form.addEventListener('dragover', onDragOver);
  form.addEventListener('drop', onDrop);

  const controlPanel = document.createElement('div');
  controlPanel.className = styles.controlPanel;
  controlPanel.dataset.testid = 'markdown-toolbar';
  /** @type {ReturnType<typeof markdownToolbar> | null} */
  let toolbar = null;

  const fieldWrapper = document.createElement('div');
  fieldWrapper.className = styles.fieldWrapper;
  const area = textareaElement(
    {
      id: props.textareaId,
      class: styles.field,
      placeholder: '',
      autofocus: Boolean(props.autofocus),
      spellcheck: 'true',
      dir: 'auto',
    },
    { onInput, controlled: true }
  );
  const field = area.textarea;
  field.addEventListener('keydown', onKeyDown);
  field.addEventListener('paste', onPaste);
  // the text the field was last sized for
  let sizedFor = /** @type {string | null} */ (null);
  // set while text cut at the limit waits to go back into the field
  let cutPending = false;

  const expander = document.createElement('text-expander');
  expander.setAttribute('keys', ':');
  /** @type {ReturnType<typeof emojiCompletion> | null} */
  let completion = null;
  const counter = document.createElement('span');
  counter.className = styles.counter;

  /** @type {HTMLParagraphElement[]} */
  let errorNodes = [];
  let errorText = '';

  const actions = document.createElement('div');
  actions.className = styles.actions;
  const buttons = document.createElement('div');
  const previewButton = buttonElement({
    kind: 'secondary',
    theme: props.theme,
    size: 'large',
    className: styles.button,
  });
  previewButton.addEventListener('click', () => getPreview());
  const submitButton = buttonElement({ kind: 'primary', size: 'large', className: styles.button, type: 'submit' });
  buttons.append(submitButton);
  const rss = document.createElement('div');
  rss.className = styles.rss;

  const markdownTip = document.createElement('div');
  markdownTip.className = styles.markdown;
  let markdownTipLocale = '';

  const subscribeBy = document.createTextNode('');
  const orBeforeEmail = document.createTextNode('');
  const orBeforeTelegram = document.createTextNode('');

  const signIn = mounted((node) => mountSignIn(node, deps.signIn));
  const rssSubscription = mounted((node) => mountSubscribeByRss(node, deps.rss.props, deps.rss.intl));
  const emailSubscription = mounted((node) => mountSubscribeByEmail(node, deps.email.props, deps.email.deps));
  const telegramSubscription = mounted((node) =>
    mountSubscribeByTelegram(node, deps.telegram.props, deps.telegram.deps)
  );

  const previewWrapper = document.createElement('div');
  previewWrapper.className = styles.previewWrapper;
  const previewNode = document.createElement('div');
  previewNode.className = classes(styles.preview, 'raw-content');
  previewNode.setAttribute('dir', 'auto');
  previewWrapper.append(previewNode);
  let previewHtml = /** @type {string | null} */ (null);

  /** @param {KeyboardEvent} evt */
  function onKeyDown(evt) {
    // send on cmd+enter or ctrl+enter
    if (evt.keyCode === 13 && (evt.metaKey || evt.ctrlKey)) {
      send(evt);
    }
  }

  function onInput() {
    const { value } = field;
    const text = value.substring(0, StaticStore.config.max_comment_size);
    updatePersistedComments(props.id, value);
    if (text !== value) {
      // text cut at the limit goes back into the field once the event is over, as the preact form
      // wrote it after its render: an insert that fired the event checks the field's text when the
      // event returns, and the cut text would make it give up the browser's own insert for good
      cutPending = true;
      queueMicrotask(() => {
        cutPending = false;
        sizedFor = null;
        if (!destroyed) {
          // writing the value moves the selection to the end; it stays where it was, inside the text
          const { selectionStart, selectionEnd } = field;
          render();
          field.setSelectionRange(
            Math.min(selectionStart, field.value.length),
            Math.min(selectionEnd, field.value.length)
          );
        }
      });
    }
    if (state.errorLock) {
      setState({ preview: null, text });
      return;
    }
    setState({ isErrorShown: false, errorMessage: null, preview: null, text });
  }

  /** @param {ClipboardEvent} evt */
  async function onPaste(evt) {
    if (!(evt.clipboardData && evt.clipboardData.files.length > 0)) {
      return;
    }
    evt.preventDefault();
    await uploadImages(Array.from(evt.clipboardData.files));
  }

  /** @param {Event} [evt] */
  async function send(evt) {
    const { text } = state;
    evt?.preventDefault();
    if (!text || !text.trim()) {
      return;
    }
    // the Send button is drawn for a signed-in reader only, while the keyboard shortcut works in a
    // guest's form too. The widget can show a guest while the browser still holds a session, and
    // the server would then accept the comment
    if (!props.user) {
      setState({ isErrorShown: true, errorMessage: deps.intl.formatMessage(errorMessages[401]) });
      return;
    }
    if (text === props.value) {
      deps.onCancel?.();
      setState({ preview: null });
    }
    setState({ isDisabled: true, isErrorShown: false, text });
    try {
      await deps.onSubmit(text, pageTitle || window.remark_config?.host_page_title || '');
    } catch (err) {
      setState({
        isDisabled: false,
        isErrorShown: true,
        errorMessage: extractErrorMessageFromResponse(err, deps.intl),
      });
      return;
    }
    removePersistedComment(props.id);
    setState({ isDisabled: false, preview: null, text: '' });
  }

  function getPreview() {
    const text = field.value;
    if (!text || !text.trim()) {
      return;
    }
    setState({ isErrorShown: false, errorMessage: null, text });
    deps
      .getPreview(text)
      .then((preview) => setState({ preview }))
      .catch((err) => {
        setState({ isErrorShown: true, errorMessage: extractErrorMessageFromResponse(err, deps.intl) });
      });
  }

  /** @param {string[]} errors */
  function appendError(...errors) {
    setState({
      errorMessage: state.errorMessage ? `${state.errorMessage}\n${errors.join('\n')}` : errors.join('\n'),
      isErrorShown: true,
    });
  }

  /** @param {DragEvent} evt */
  function onDragOver(evt) {
    if (!props.user) {
      evt.preventDefault();
    }
    if (!deps.uploadImage || StaticStore.config.max_image_size === 0 || !evt.dataTransfer) {
      return;
    }
    const items = Array.from(evt.dataTransfer.items);
    if (items.filter((item) => item.kind === 'file' && IMAGE_MIME.test(item.type)).length === 0) {
      return;
    }
    evt.preventDefault();
    evt.dataTransfer.dropEffect = 'copy';
  }

  /** @param {DragEvent} evt */
  function onDrop(evt) {
    const anonymous = props.user && isUserAnonymous(props.user);
    if (!props.user || anonymous) {
      const message = anonymous ? messages.anonymousUploadingDisabled : messages.unauthorizedUploadingDisabled;
      setState({ isErrorShown: true, errorMessage: deps.intl.formatMessage(message) });
      evt.preventDefault();
      return;
    }
    if (!deps.uploadImage || StaticStore.config.max_image_size === 0 || !evt.dataTransfer) {
      return;
    }
    const files = Array.from(evt.dataTransfer.files).filter((file) => IMAGE_MIME.test(file.type));
    if (files.length === 0) {
      return;
    }
    evt.preventDefault();
    uploadImages(files);
  }

  /**
   * @param {File} file
   * @returns {Promise<Image | Error>}
   */
  function uploadImage(file) {
    const upload = /** @type {(image: File) => Promise<Image>} */ (deps.uploadImage);
    return upload(file).catch(
      (err) =>
        new Error(
          deps.intl.formatMessage(messages.uploadFileFail, {
            fileName: file.name,
            errorMessage: extractErrorMessageFromResponse(err, deps.intl),
          })
        )
    );
  }

  /** @param {[number, number]} selection */
  function setSelection(selection) {
    field.selectionStart = selection[0];
    field.selectionEnd = selection[1];
  }

  /** @param {File[]} files */
  async function uploadImages(files) {
    const { intl } = deps;
    if (!deps.uploadImage) {
      return;
    }
    // a human-readable size limit, such as 5.00MB
    const maxImageSize = `${(StaticStore.config.max_image_size / 1024 / 1024).toFixed(2)}MB`;

    setState({
      errorLock: true,
      errorMessage: null,
      isErrorShown: false,
      isDisabled: true,
      buttonText: intl.formatMessage(messages.uploading),
    });

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const placeholderStart = state.text.length === 0 ? '' : '\n';
      const placeholder = `${placeholderStart}![${intl.formatMessage(messages.uploadingFile, { fileName: file.name })}]()`;
      /** @type {[number, number]} */
      const selection = [field.selectionStart, field.selectionEnd];
      const originalText = state.text;

      if (file.size > StaticStore.config.max_image_size) {
        appendError(intl.formatMessage(messages.exceededSize, { fileName: file.name, maxImageSize }));
        continue;
      }

      setState({ text: replaceSelection(state.text, selection, placeholder) });
      updatePersistedComments(props.id, state.text);

      if (i !== 0) {
        await wait(UPLOAD_DELAY);
      }

      const result = await uploadImage(file);
      if (destroyed) {
        return;
      }

      if (result instanceof Error) {
        appendError(result.message);
        setState({ text: originalText });
        await wait(SETTLE_DELAY);
        setSelection(selection);
        continue;
      }

      const markdown = `${placeholderStart}![${result.name}](${result.url})`;
      setState({ text: replaceSelection(state.text, [selection[0], selection[0] + placeholder.length], markdown) });
      updatePersistedComments(props.id, state.text);
      await wait(SETTLE_DELAY);
      const caret = selection[0] + markdown.length;
      setSelection([caret, caret]);
    }

    setState({ errorLock: false, isDisabled: false, buttonText: null });
  }

  function renderMarkdownTip() {
    if (markdownTipLocale === deps.intl.locale && markdownTip.childNodes.length > 0) {
      return;
    }
    markdownTipLocale = deps.intl.locale;
    const parts = /** @type {unknown} */ (
      deps.intl.formatMessage(messages.markdownTip, {
        a: (title) => {
          const link = document.createElement('a');
          link.className = styles.markdownLink;
          link.target = '_blank';
          link.href = 'markdown-help.html';
          link.textContent = String(title);
          return /** @type {any} */ (link);
        },
      })
    );
    markdownTip.replaceChildren(
      ...(Array.isArray(parts) ? parts : [parts]).map((part) => (part instanceof Node ? part : String(part)))
    );
  }

  /**
   * The subscriptions after the markdown tip, as text and their dropdowns: none when the page
   * turned them all off, and null when they are not wanted, which takes down any that were shown.
   *
   * @param {boolean} wanted
   */
  function renderSubscriptions(wanted) {
    const { config } = StaticStore;
    const { intl } = deps;
    const byEmail = Boolean(config.email_notifications && isEmailSubscription);
    const byTelegram = Boolean(config.telegram_notifications && isTelegramSubscription);
    const byRss = isRssSubscription;
    const or = ` ${intl.formatMessage(messages.subscribeOr)} `;

    const rssRoot = rssSubscription.place(wanted && byRss, (part) => part.update(deps.rss.props, deps.rss.intl));
    const emailRoot = emailSubscription.place(wanted && byEmail, (part) =>
      part.update(deps.email.props, deps.email.deps)
    );
    const telegramRoot = telegramSubscription.place(wanted && byTelegram, (part) =>
      part.update(deps.telegram.props, deps.telegram.deps)
    );
    if (!wanted) {
      return null;
    }
    if (!byRss && !byEmail && !byTelegram) {
      return [];
    }
    subscribeBy.data = `${intl.formatMessage(messages.subscribeBy)} `;
    orBeforeEmail.data = or;
    orBeforeTelegram.data = or;
    return [
      subscribeBy,
      rssRoot,
      byRss && byEmail ? orBeforeEmail : null,
      emailRoot,
      (byRss && byTelegram) || (byEmail && byTelegram) ? orBeforeTelegram : null,
      telegramRoot,
    ].filter((node) => node !== null);
  }

  function renderErrors() {
    const message = props.errorMessage || state.errorMessage;
    const shown = state.isErrorShown || Boolean(message);
    const text = shown ? message || deps.intl.formatMessage(messages.unexpectedError) : '';
    if (text !== errorText) {
      errorText = text;
      errorNodes = text
        ? text.split('\n').map((line) => {
            const node = document.createElement('p');
            node.className = styles.error;
            node.setAttribute('role', 'alert');
            node.textContent = line;
            return node;
          })
        : [];
    }
    return errorNodes;
  }

  function render() {
    const { config } = StaticStore;
    const { intl } = deps;
    const { mode, user, theme } = props;
    const simple = config.simple_view;

    form.className = classes(
      styles.root,
      theme === 'dark' ? styles.themeDark : styles.themeLight,
      (!mode || mode === 'reply') && styles.typeReply,
      simple && styles.simple,
      ...(Array.isArray(props.mix) ? props.mix : [props.mix])
    );
    form.setAttribute('aria-label', intl.formatMessage(messages.newComment));
    form.dataset.testid = `commentform_${props.id}`;

    // the toolbar
    if (!simple) {
      const toolbarProps = { textareaId: props.textareaId, allowUpload: Boolean(deps.uploadImage) };
      const toolbarDeps = { intl, field: () => field, uploadImages };
      if (toolbar) {
        toolbar.update(toolbarProps, toolbarDeps);
      } else {
        toolbar = markdownToolbar(toolbarProps, toolbarDeps);
        controlPanel.append(toolbar.root);
      }
    } else if (toolbar) {
      toolbar.destroy();
      toolbar.root.remove();
      toolbar = null;
    }

    // the field
    // in edit mode there is no separate placeholder: the comment being edited is itself either a
    // top-level comment or a reply, so its placeholder matches whichever one it already is
    const replyPlaceholder = mode === 'reply' || (mode === 'edit' && props.isReply);
    field.placeholder = intl.formatMessage(replyPlaceholder ? messages.replyPlaceholder : messages.placeholder);
    if (!cutPending && field.value !== state.text) {
      field.value = state.text;
    }
    field.disabled = state.isDisabled;
    if (config.emoji_enabled) {
      if (field.parentNode !== expander) {
        expander.append(field);
      }
      const completionProps = { theme: deps.theme };
      const completionDeps = { loadEmoji: deps.loadEmoji };
      if (completion) {
        completion.update(completionProps, completionDeps);
      } else {
        completion = emojiCompletion(expander, field, completionProps, completionDeps);
      }
    } else if (completion) {
      completion.destroy();
      completion = null;
    }
    const charactersLeft = config.max_comment_size - state.text.length;
    counter.textContent = String(charactersLeft);
    placeChildren(fieldWrapper, [config.emoji_enabled ? expander : field, ...(charactersLeft < 100 ? [counter] : [])]);

    // the buttons, or the sign-in form for a guest
    renderMarkdownTip();
    const signInRoot = signIn.place(!user, (part) => part.update(deps.signIn));
    if (user) {
      previewButton.className = buttonClassName({ kind: 'secondary', theme, size: 'large', className: styles.button });
      previewButton.textContent = intl.formatMessage(messages.preview);
      previewButton.disabled = state.isDisabled;
      const labels = { main: messages.send, edit: messages.save, reply: messages.reply };
      submitButton.textContent = state.buttonText || intl.formatMessage(labels[mode || 'main']);
      submitButton.disabled = state.isDisabled;
      placeChildren(buttons, simple ? [submitButton] : [previewButton, submitButton]);
      const subscriptions = renderSubscriptions(mode === 'main');
      if (subscriptions) {
        placeChildren(rss, [markdownTip, ...subscriptions]);
      }
      placeChildren(actions, subscriptions ? [buttons, rss] : [buttons]);
    } else {
      renderSubscriptions(false);
      placeChildren(actions, [/** @type {HTMLDivElement} */ (signInRoot), markdownTip]);
    }

    // the preview
    if (state.preview && state.preview !== previewHtml) {
      previewNode.innerHTML = state.preview;
    }
    previewHtml = state.preview || null;

    placeChildren(form, [
      ...(simple ? [] : [controlPanel]),
      fieldWrapper,
      ...renderErrors(),
      actions,
      ...(state.preview ? [previewWrapper] : []),
    ]);

    if (sizedFor !== state.text && field.isConnected) {
      sizedFor = state.text;
      area.resize();
    }
  }

  render();
  root.append(form);
  sizedFor = state.text;
  area.resize();

  return {
    /**
     * The props of a new render of the form's parent: a new starting text replaces what is typed,
     * and a signed-in reader's error goes unless the form edits a comment.
     *
     * @param {CommentFormProps} next
     * @param {CommentFormDeps} nextDeps
     */
    update(next, nextDeps) {
      const previous = props;
      props = next;
      deps = nextDeps;
      if (next.value !== previous.value) {
        state.text = next.value || '';
      }
      if (next.user && !previous.value) {
        state.isErrorShown = false;
        state.errorMessage = null;
      }
      render();
    },
    destroy() {
      if (destroyed) {
        return;
      }
      destroyed = true;
      timers.forEach((timer) => clearTimeout(timer));
      timers.clear();
      toolbar?.destroy();
      completion?.destroy();
      signIn.destroy();
      rssSubscription.destroy();
      emailSubscription.destroy();
      telegramSubscription.destroy();
    },
  };
}
