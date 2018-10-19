/* eslint-disable no-console */
import { NODE_ID, BASE_URL } from 'common/constants';
import { approveDeleteMe, getUser } from 'common/api';
import { token } from 'common/settings';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

function init() {
  const node = document.getElementById(NODE_ID);

  if (!node) {
    console.error("Remark42: Can't find root node.");
    return;
  }

  getUser()
    .then(user => {
      if (!user.admin) {
        handleNotAuthorizedError(node);
        return;
      }

      approveDeleteMe(token).then(
        data =>
          (node.innerHTML = `
			<h3>User deleted successfully</h3>
			<pre>${JSON.stringify(data, null, 4)}</pre>`),
        err =>
          (node.innerHTML = `
              <h3>Something went wrong</h3>
              <pre>${err}</pre>`)
      );
    })
    .catch(() => handleNotAuthorizedError(node));
}

function handleNotAuthorizedError(node) {
  node.innerHTML = `<h3>You are not logged in</h3>
        			  <p><a href='${BASE_URL}' target='_blank'>Sign in</a> as admin to delete user information</p>`;
}
