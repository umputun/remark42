import { h, render } from 'preact';
import { bindActionCreators } from 'redux';
import { Provider } from 'react-redux';
import { IntlProvider } from 'react-intl';

import { loadLocale } from 'utils/loadLocale';
import { getLocale } from 'utils/getLocale';
import { ConnectedRoot } from 'components/root';
import { UserInfo } from 'components/user-info';
import { store } from 'store';
import { NODE_ID, BASE_URL } from 'common/constants';
import { StaticStore } from 'common/static-store';
import { getConfig } from 'common/api';
import { fetchHiddenUsers } from 'store/user/actions';
import { restoreCollapsedThreads } from 'store/thread/actions';
import { parseQuery } from 'utils/parseQuery';
import { parseBooleansFromDictionary } from 'utils/parse-booleans-from-dictionary';

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

  const params = parseQuery<{ page?: string; locale?: string; simple_view?: boolean }>();
  const locale = getLocale(params);
  const messages = await loadLocale(locale).catch(() => ({}));
  const boundActions = bindActionCreators({ fetchHiddenUsers, restoreCollapsedThreads }, store.dispatch);

  boundActions.fetchHiddenUsers();
  boundActions.restoreCollapsedThreads();

  const config = await getConfig();
  const optionsParams = parseBooleansFromDictionary(params, 'simple_view');
  StaticStore.config = { ...config, ...optionsParams };

  render(
    <IntlProvider locale={locale} messages={messages}>
      <Provider store={store}>{params.page === 'user-info' ? <UserInfo /> : <ConnectedRoot />}</Provider>
    </IntlProvider>,
    node
  );
}
