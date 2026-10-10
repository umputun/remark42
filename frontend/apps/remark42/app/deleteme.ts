import { NODE_ID } from 'common/constants';
import { approveDeleteMe, getUser } from 'common/api';
import { token } from 'common/settings';
import type { ApiError } from 'common/types';

if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', init);
} else {
  init();
}

async function init(): Promise<void> {
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

    // everything below reaches the page as text. the token is supplied by whoever sent the link,
    // and both the response and the error echo values it carries, its site id among them
    approveDeleteMe(token).then(
      (data) => {
        render(node, 'User deleted successfully', JSON.stringify(data, null, 4));
      },
      (err: Error | ApiError | string) => {
        const message =
          err instanceof Error ? err.message : typeof err === 'object' && err !== null && err.error ? err.error : err;
        console.error(err);
        render(node, 'Something went wrong', String(message));
      }
    );
  });
}

function render(node: HTMLElement, heading: string, details: string): void {
  const title = document.createElement('h3');
  title.textContent = heading;

  const body = document.createElement('pre');
  body.textContent = details;

  node.replaceChildren(title, body);
}

function handleNotAuthorizedError(node: HTMLElement): void {
  node.innerHTML = `
    <h3>You are not logged in</h3>
    <p><a href='/web' target='_blank'>Sign in</a> as admin to delete user information</p>
  `;
}
