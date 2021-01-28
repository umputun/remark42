/* eslint-disable no-console */
import { NODE_ID } from 'common/constants';
import { approveDeleteMe, getUser } from 'common/api';
import { token } from 'common/settings';
import { ApiError } from 'common/types';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
  __webpack_public_path__ = `${window.location.origin}/web/`;

  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error("Remark42: Can't find root node.");
    return;
  }

  getUser().then((user) => {
    if (!user || !user.admin) {
      handleNotAuthorizedError(node);
      return;
    }

    approveDeleteMe(token).then(
      (data) => {
        node.innerHTML = `
            <h3>User deleted successfully</h3>
            <pre>${JSON.stringify(data, null, 4)}</pre>
          `;
      },
      (err: Error | ApiError | string) => {
        const message =
          err instanceof Error ? err.message : typeof err === 'object' && err !== null && err.error ? err.error : err;
        console.error(err);
        node.innerHTML = `
          <h3>Something went wrong</h3>
          <pre>${message}</pre>
        `;
      }
    );
  });
}

function handleNotAuthorizedError(node: HTMLElement): void {
  node.innerHTML = `
    <h3>You are not logged in</h3>
    <p><a href='/web' target='_blank'>Sign in</a> as admin to delete user information</p>
  `;
}
