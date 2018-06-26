import {
  h,
  render
} from 'preact';

import {
  BASE_URL,
  USER_INFO_NODE_CLASSNAME
} from './common/constants';
import store from 'common/store';
import UserInfo from 'components/user-info';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const nodes = document.getElementsByClassName(USER_INFO_NODE_CLASSNAME);

  if (!nodes) {
    console.error('Remark42: Can\'t find last comments nodes.');
    return;
  }

  try {
    remark_config = remark_config || {};
  } catch (e) {
    console.error('Remark42: Config object is undefined.');
    return;
  }

  if (!remark_config.site_id) {
    console.error('Remark42: Site ID is undefined.');
    return;
  }

  const styles = document.createElement('link');
  styles.href = `${BASE_URL}/web/remark.css`;
  styles.rel = 'stylesheet';
  (document.head || document.body).appendChild(styles);
  const params = window.location.search.replace(/^\?/, '')
    .split('&').reduce((memo, value) => {
      const vals = value.split('=');
      if (vals.length === 2) {
        memo[vals[0]] = vals[1];
      }
      return memo;
    }, {});
  const user = {
    id: params.id,
    name: params.name || '',
    isDefaultPicture: params.isDefaultPicture,
    picture: params.picture
  };

  function onClose() {
    if (window.parent) {
      window.parent.postMessage(JSON.stringify({ isUserInfoShown: false }), '*');
    }
  }
  store.set('user', user);
  [].slice.call(nodes).forEach(node => {
    render(<UserInfo user={user} onClose={onClose}/>, node);
  });
}
