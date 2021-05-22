import { h, render } from 'preact';
import { bindActionCreators } from 'redux';
import { Provider } from 'react-redux';
import { IntlProvider } from 'react-intl';

import { loadLocale } from 'utils/loadLocale';
import { getLocale } from 'utils/getLocale';
import { parseQuery } from 'utils/parse-query';
import { parseMessage } from 'utils/post-message';
import { parseBooleansFromDictionary } from 'utils/parse-booleans-from-dictionary';
import { ConnectedRoot } from 'components/root';
import { Profile } from 'components/profile';
import { store } from 'store';
import { NODE_ID, BASE_URL } from 'common/constants';
import { StaticStore } from 'common/static-store';
import { getConfig } from 'common/api';
import { fetchHiddenUsers } from 'store/user/actions';
import { restoreCollapsedThreads } from 'store/thread/actions';

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

  const params = parseQuery();
  const locale = getLocale(params);
  const messages = await loadLocale(locale).catch(() => ({}));
  const boundActions = bindActionCreators({ fetchHiddenUsers, restoreCollapsedThreads }, store.dispatch);

  node.innerHTML = '';

  window.addEventListener('message', (evt) => {
    const data = parseMessage(evt);

    if (data.theme === 'light') {
      document.body.classList.remove('dark');
    }

    if (data.theme === 'dark') {
      document.body.classList.add('dark');
    }
  });

  if (params.theme === 'dark') {
    document.body.classList.add('dark');
  }

  boundActions.fetchHiddenUsers();
  boundActions.restoreCollapsedThreads();

  const config = await getConfig();
  const optionsParams = parseBooleansFromDictionary(params, 'simple_view');
  StaticStore.config = { ...config, ...optionsParams };

  render(
    <IntlProvider locale={locale} messages={messages}>
      <Provider store={store}>{params.page === 'profile' ? <Profile /> : <ConnectedRoot />}</Provider>
    </IntlProvider>,
    node
  );
}
