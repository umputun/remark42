/** @jsx createElement */
// Must be the first import
if (process.env.NODE_ENV === 'development') {
  // Must use require here as import statements are only allowed
  // to exist at the top of a file.
  require('preact/debug');
}
import loadPolyfills from '@app/common/polyfills';

import { IntlProvider } from 'react-intl';
import { loadLocale } from './utils/loadLocale';
import { getLocale } from './utils/getLocale';

import { createElement, render } from 'preact';
import { bindActionCreators } from 'redux';
import { Provider } from 'react-redux';

import { ConnectedRoot } from '@app/components/root';
import { UserInfo } from '@app/components/user-info';
import reduxStore from '@app/store';

// importing css
import '@app/components/list-comments';

import { NODE_ID, BASE_URL } from '@app/common/constants';
import { StaticStore } from '@app/common/static_store';
import api from '@app/common/api';
import { fetchHiddenUsers } from './store/user/actions';
import { restoreProvider } from './store/provider/actions';
import { restoreCollapsedThreads } from './store/thread/actions';
import parseQuery from './utils/parseQuery';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = BASE_URL + '/web/';

  await loadPolyfills();

  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error("Remark42: Can't find root node."); // eslint-disable-line no-console
    return;
  }

  const boundActions = bindActionCreators(
    { fetchHiddenUsers, restoreProvider, restoreCollapsedThreads },
    reduxStore.dispatch
  );
  boundActions.fetchHiddenUsers();
  boundActions.restoreProvider();
  boundActions.restoreCollapsedThreads();

  const params = parseQuery();
  const locale = getLocale(params);
  const messages = await loadLocale(locale).catch(() => ({}));
  StaticStore.config = await api.getConfig();

  if (params.page === 'user-info') {
    return render(
      <IntlProvider locale={locale} messages={messages}>
        <div id={NODE_ID}>
          <div className="root root_user-info">
            <Provider store={reduxStore}>
              <UserInfo />
            </Provider>
          </div>
        </div>
      </IntlProvider>,
      node
    );
  }

  render(
    <IntlProvider locale={locale} messages={messages}>
      <Provider store={reduxStore}>
        <ConnectedRoot />
      </Provider>
    </IntlProvider>,
    node
  );
}
