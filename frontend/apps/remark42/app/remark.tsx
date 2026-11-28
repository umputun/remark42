import { h, render } from 'preact';
import { bindActionCreators } from 'store/create-store';
import { Provider } from 'store/context';

import { IntlProvider } from 'common/intl';
import { getDirection } from 'common/direction';
import { getDocumentLang } from 'common/document-lang';
import { loadLocale } from 'utils/loadLocale';
import { parseMessage, isFromParent } from 'utils/post-message';
import { ConnectedRoot } from 'components/root';
import { Profile } from 'components/profile';
import { store } from 'store';
import { NODE_ID, THEMES } from 'common/constants';
import type { Theme } from 'common/types';
import { StaticStore } from 'common/static-store';
import { getConfig } from 'common/api';
import { fetchHiddenUsers, followReaderAcrossFrames } from 'store/user/actions';
import { restoreCollapsedThreads } from 'store/thread/actions';
import { setTheme } from 'store/theme/actions';
import { locale, theme, rawParams } from 'common/settings';

// above the call below: init can run right away, and a const declared after it is not yet initialized
const boundActions = bindActionCreators(
  { fetchHiddenUsers, followReaderAcrossFrames, restoreCollapsedThreads, setTheme },
  store.dispatch
);

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

// the embed script resolves 'auto' before posting, so the document only ever receives light or dark
function isTheme(value: unknown): value is Theme {
  return THEMES.includes(value as Theme);
}

function applyTheme(next: Theme) {
  document.body.classList.toggle('dark', next === 'dark');
  // global.css keys the light theme's own colors on this class
  document.documentElement.classList.toggle('light', next === 'light');
  document.documentElement.style.colorScheme = next;
  // Root renders from the store, whose initial theme is the URL's
  boundActions.setTheme(next);
}

async function init(): Promise<void> {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    throw new Error("Remark42: Can't find root node.");
  }

  // before the await below: a theme the parent posts while the locale loads would otherwise reach
  // no listener. The parent can flip it at any time after that too
  window.addEventListener('message', (evt) => {
    if (!isFromParent(evt)) {
      return;
    }

    const data = parseMessage(evt);

    if (isTheme(data.theme)) {
      applyTheme(data.theme);
    }
  });

  // the inline listener in iframe.html keeps the latest theme the parent posted before this bundle
  // ran; the URL only has the one the iframe was built with
  const current = window.remark_config?.theme;
  applyTheme(isTheme(current) ? current : theme);

  const messages = await loadLocale(locale).catch(() => ({}));

  node.innerHTML = '';

  // locale is fixed for the life of the page (no message updates it after init), so this only
  // needs to run once, unlike the theme above which a parent page can flip at any time
  document.documentElement.dir = getDirection(locale);
  // an empty catalogue means the widget shows English, either chosen or because the catalogue failed to load
  document.documentElement.lang = Object.keys(messages).length > 0 ? getDocumentLang(locale) : 'en';

  boundActions.fetchHiddenUsers();
  boundActions.restoreCollapsedThreads();
  if (rawParams.page !== 'profile') {
    boundActions.followReaderAcrossFrames();
  }

  const config = await getConfig();
  StaticStore.config = {
    ...config,
    simple_view: config.simple_view || rawParams.simple_view === 'true',
  };

  render(
    <IntlProvider locale={locale} messages={messages}>
      <Provider store={store}>{rawParams.page === 'profile' ? <Profile /> : <ConnectedRoot />}</Provider>
    </IntlProvider>,
    node
  );
}
