// The OAuth sign-in buttons in plain JavaScript, drawn as the preact OAuth component draws them.

import { classes } from '../../../common/dom.js';
import { BASE_URL } from '../../../common/constants.config.ts';
import { siteId } from '../../../common/settings.ts';
import { messages } from '../auth.messages.ts';
import { getButtonVariant, getProviderData } from './oauth.utils.ts';
import styles from './oauth.module.css';

const location = encodeURIComponent(`${window.location.origin}${window.location.pathname}?selfClose`);

/**
 * @typedef {object} OAuthListDeps
 * @property {import('../../../common/intl-message.ts').IntlShape} intl
 * @property {import('../../../common/types.ts').Theme} theme  which icon a provider with two shows
 * @property {(evt: MouseEvent) => void} onOauthClick  a provider's button was clicked
 */

/**
 * @param {import('../../../common/types.ts').OAuthProvider[]} providers
 * @param {OAuthListDeps} initialDeps
 */
export function oauthList(providers, initialDeps) {
  let deps = initialDeps;
  const variant = getButtonVariant(providers.length);
  const root = document.createElement('ul');
  root.className = classes('oauth', styles.root);

  const items = providers.map((provider) => {
    const item = document.createElement('li');
    item.className = classes('oauth-item', styles.item);
    const button = document.createElement('a');
    button.target = '_blank';
    button.rel = 'noopener noreferrer';
    button.href = `${BASE_URL}/auth/${provider}/login?from=${location}&site=${siteId}`;
    button.className = classes('oauth-button', styles.button, styles[variant], styles[provider]);
    button.addEventListener('click', (evt) => deps.onOauthClick(evt));
    const icon = document.createElement('img');
    icon.className = 'oauth-icon';
    icon.src = '';
    icon.width = 20;
    icon.height = 20;
    icon.alt = '';
    icon.setAttribute('aria-hidden', 'true');
    button.append(icon);
    item.append(button);
    root.append(item);
    return { provider, button, icon };
  });

  function render() {
    for (const { provider, button, icon } of items) {
      const { name, icon: src } = getProviderData(provider, deps.theme);
      button.dataset.providerName = name;
      button.title = deps.intl.formatMessage(messages.oauthTitle, { provider: name });
      // only when it differs: setting a source again restarts the image's load
      if (icon.getAttribute('src') !== src) {
        icon.src = src;
      }
    }
  }

  render();
  return {
    root,
    /** @param {OAuthListDeps} next */
    update(next) {
      deps = next;
      render();
    },
  };
}
