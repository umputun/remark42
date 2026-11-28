/**
 * The OAuth sign-in of a frame on another site. The popup is a page of its own, and the session its
 * callback sets is remark42's own site's, which a browser keeping third-party cookies apart never
 * shows to the frame. The popup therefore opens on the backend's start page, which tells this frame
 * the id of a sign-in bound to a verifier only this frame holds. Once the provider has brought the
 * reader back, the frame either sees the new account already, or redeems the sign-in for a session
 * of its own. See backend/app/rest/api/handoff.go.
 */

/** What the start page posts to the frame, and what the frame answers it with */
export const HANDOFF_MESSAGE = 'remark42-handoff';
export const HANDOFF_ACK = 'remark42-handoff-ack';

/** The code of the error a blocked sign-in window rejects with */
export const POPUP_BLOCKED = 'popup-blocked';

export type HandoffPair = { verifier: string; challenge: string };

let ready: HandoffPair | null = null;
let making = false;

function base64url(bytes: Uint8Array): string {
  let s = '';
  bytes.forEach((b) => {
    s += String.fromCharCode(b);
  });
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '');
}

/**
 * Makes the verifier for the next sign-in ahead of the click. Its hash is computed asynchronously, and
 * a popup opened after waiting for it is no longer opened by the reader's click, so a browser blocks
 * it. Without WebCrypto, which needs https, nothing is made and the sign-in goes as it always did
 */
export function prepareHandoff(): void {
  if (ready || making || !window.crypto?.subtle || typeof TextEncoder === 'undefined') {
    return;
  }
  making = true;
  const verifier = base64url(window.crypto.getRandomValues(new Uint8Array(32)));
  window.crypto.subtle
    .digest('SHA-256', new TextEncoder().encode(verifier))
    .then(
      (digest) => {
        ready = { verifier, challenge: base64url(new Uint8Array(digest)) };
      },
      () => undefined
    )
    .finally(() => {
      making = false;
    });
}

/** Takes the prepared verifier, one per sign-in, and starts making the next */
export function takeHandoff(): HandoffPair | null {
  const pair = ready;
  ready = null;
  prepareHandoff();
  return pair;
}

/**
 * The start page for the provider login the widget links to, or null for a link that is not one.
 * It takes the same `from` and `site`, and goes on to exactly that login when it cannot hand over
 */
export function handoffStartURL(loginURL: string, challenge: string): string | null {
  let u: URL;
  try {
    u = new URL(loginURL, window.location.href);
  } catch (e) {
    return null;
  }
  const m = u.pathname.match(/^(.*)\/auth\/([^/]+)\/login$/);
  if (!m) {
    return null;
  }
  u.pathname = `${m[1]}/auth/handoff/start`;
  u.searchParams.set('provider', decodeURIComponent(m[2]));
  u.searchParams.set('challenge', challenge);
  return u.toString();
}

/** The transaction id a message from the start page carries, or null for any other message */
export function handoffTxn(data: unknown): string | null {
  if (typeof data !== 'object' || data === null) {
    return null;
  }
  const { type, txn } = data as { type?: unknown; txn?: unknown };
  return type === HANDOFF_MESSAGE && typeof txn === 'string' && /^[0-9a-f]{32}$/.test(txn) ? txn : null;
}
