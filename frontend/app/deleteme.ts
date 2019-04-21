/* eslint-disable no-console */
import loadPolyfills from '@app/common/polyfills';
import { NODE_ID, BASE_URL } from '@app/common/constants';
import { approveDeleteMe, getUser } from '@app/common/api';
import { token } from '@app/common/settings';

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
    console.error("Remark42: Can't find root node.");
    return;
  }

  getUser()
    .then(user => {
      if (user && !user.admin) {
        handleNotAuthorizedError(node);
        return;
      }

      approveDeleteMe(token!).then(
        data => {
          node.innerHTML = `
            <h3>User deleted successfully</h3>
            <pre>${JSON.stringify(data, null, 4)}</pre>
          `;
        },
        err => {
          node.innerHTML = `
            <h3>Something went wrong</h3>
            <pre>${err}</pre>
          `;
        }
      );
    })
    .catch(() => handleNotAuthorizedError(node));
}

function handleNotAuthorizedError(node: HTMLElement): void {
  node.innerHTML = `
    <h3>You are not logged in</h3>
    <p><a href='${BASE_URL}' target='_blank'>Sign in</a> as admin to delete user information</p>
  `;
}
