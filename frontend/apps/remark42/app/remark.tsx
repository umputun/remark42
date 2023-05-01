import { getConfig } from 'common/api';
import { BASE_URL, NODE_ID } from 'common/constants';
import { locale, rawParams, theme } from 'common/settings';
import { StaticStore } from 'common/static-store';
import { isThemeStyles, setThemeStyles } from 'common/theme';
import { Profile } from 'components/profile';
import { ConnectedRoot } from 'components/root';
import { render } from 'preact';
import { IntlProvider } from 'react-intl';
import { Provider } from 'react-redux';
import { bindActionCreators } from 'redux';
import { store } from 'store';
import { restoreCollapsedThreads } from 'store/thread/actions';
import { fetchHiddenUsers } from 'store/user/actions';
import { loadLocale } from 'utils/loadLocale';
import { parseMessage } from 'utils/post-message';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = `${BASE_URL}/web/`;

  const node = document.getElementById(NODE_ID);

  if (!node) {
    throw new Error("Remark42: Can't find root node.");
  }

  const messages = await loadLocale(locale).catch(() => ({}));
  const boundActions = bindActionCreators({ fetchHiddenUsers, restoreCollapsedThreads }, store.dispatch);

  node.innerHTML = '';

  window.addEventListener('message', (evt) => {
    const data = parseMessage(evt);

    if (data.theme) {
      setTheme(data.theme);
    }

    if (isThemeStyles(data.styles)) {
      setThemeStyles(data.styles);
    }
  });

  if (theme) {
    setTheme(theme);
  }

  boundActions.fetchHiddenUsers();
  boundActions.restoreCollapsedThreads();

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

const setTheme = (theme: string) => {
  if (theme === 'light') {
    document.body.classList.remove('dark');
  }

  if (theme === 'dark') {
    document.body.classList.add('dark');
  }
};
