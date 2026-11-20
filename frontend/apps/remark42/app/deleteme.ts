import { NODE_ID } from 'common/constants';
import { approveDeleteMe, getUser } from 'common/api';
import { siteId, token } from 'common/settings';
import type { ApiError } from 'common/types';
import type { RequestError } from 'utils/errorUtils';

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
        // the backend's own text says what is wrong with the link, which is what an admin opening
        // it needs. a request error carries the message picked for a reader, and the body beside it
        const reason = (err instanceof Error && (err as RequestError).body) || err;
        const message =
          reason instanceof Error
            ? reason.message
            : typeof reason === 'object' && reason !== null && reason.error
              ? reason.error
              : reason;
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
  // the session has to belong to the site the request came from, so the admin signs in through the
  // comments on that site rather than the demo page, which is fixed to a site of its own
  render(
    node,
    'You are not logged in',
    `Sign in as admin to the comments on site "${siteId}", then open this link again to delete user information.`
  );
}
