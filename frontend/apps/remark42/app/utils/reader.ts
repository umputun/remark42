import type { Profile, User } from 'common/types';

// The profile frame learns from the widget frame who the reader is. Both are frames of this origin
// on the page embedding them, and a message one posts to the other addressed to this origin is
// delivered to no other origin, so the page never sees the answer. It also works where the
// profile frame has no session of its own: with AUTH_SEND_JWT_HEADER the session is a token in the
// widget frame's memory, and a browser blocking third-party cookies sends the profile frame none.

/** How long the profile frame waits for the widget frame to answer before it takes no answer */
export const READER_TIMEOUT = 3000;

type ReaderMessage = {
  readerAsk?: string;
  readerAnswer?: string;
  reader?: Profile | null;
};

// tokens of the profiles this widget frame opened. The profile frame quotes its token when it
// asks, so with several widgets of one instance on a page only the one that opened it answers
const issued = new Set<string>();

function readMessage(evt: MessageEvent): ReaderMessage {
  if (evt.origin !== window.location.origin || typeof evt.data !== 'object' || evt.data === null) {
    return {};
  }
  return evt.data as ReaderMessage;
}

/** Issues the token a profile this widget frame opens is sent with */
export function issueReaderToken(): string {
  const token = Array.from(window.crypto.getRandomValues(new Uint32Array(4)), (n) => n.toString(36)).join('');
  issued.add(token);
  return token;
}

/**
 * Answers the profile frames this widget frame opened with the reader it has signed in, or null.
 * Returns the function that stops answering.
 */
export function answerReaderAsks(getUser: () => User | null): () => void {
  function handleMessage(evt: MessageEvent) {
    const { readerAsk } = readMessage(evt);

    if (typeof readerAsk !== 'string' || !issued.has(readerAsk) || !evt.source) {
      return;
    }

    const user = getUser();
    const reader: Profile | null = user ? { id: user.id, name: user.name, picture: user.picture } : null;
    const answer: ReaderMessage = { readerAnswer: readerAsk, reader };

    (evt.source as Window).postMessage(answer, window.location.origin);
  }

  window.addEventListener('message', handleMessage);
  return () => window.removeEventListener('message', handleMessage);
}

/**
 * Asks the widget frame that issued the token who the reader is. Resolves with null when nobody is
 * signed in there or no answer comes within READER_TIMEOUT.
 */
export function askReader(token: string): Promise<Profile | null> {
  return new Promise((resolve) => {
    function finish(reader: Profile | null) {
      window.clearTimeout(timer);
      window.removeEventListener('message', handleMessage);
      resolve(reader);
    }

    function handleMessage(evt: MessageEvent) {
      const { readerAnswer, reader } = readMessage(evt);

      if (readerAnswer === token) {
        finish(reader ?? null);
      }
    }

    window.addEventListener('message', handleMessage);
    const timer = window.setTimeout(() => finish(null), READER_TIMEOUT);

    // the widget frame is a sibling of this one. Frames of other origins on the page are asked
    // too, since a frame's origin cannot be read from here, and the browser drops the message to
    // each of them: it is addressed to this origin only
    const ask: ReaderMessage = { readerAsk: token };
    const { frames } = window.parent;
    for (let i = 0; i < frames.length; i++) {
      if (frames[i] !== window) {
        frames[i].postMessage(ask, window.location.origin);
      }
    }
  });
}
