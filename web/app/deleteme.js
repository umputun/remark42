/* eslint-disable no-console */
import { NODE_ID } from 'common/constants';
import { approveDeleteMe } from 'common/api';
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
}
