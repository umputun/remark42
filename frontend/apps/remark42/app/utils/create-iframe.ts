import { BASE_URL } from 'common/constants.config';
import { parseMessage } from 'utils/post-message';
import { setStyles, setAttributes, StylesDeclaration } from 'utils/set-dom-props';

type Params = {
  [key: string]: unknown;
  __colors__?: Record<string, string>;
  __font__?: string;
  styles?: StylesDeclaration;
  onReveal?: () => void;
};

/**
 * How long to wait for the iframe document to report itself as inited before
 * showing it anyway. Only reached when the document fails to bootstrap.
 */
const REVEAL_TIMEOUT = 5000;

/**
 * Creates the remark42 iframe for the given params.
 *
 * The returned iframe starts hidden and becomes visible once its document posts
 * `inited`, or after REVEAL_TIMEOUT if that never arrives. Until then it holds a
 * global `message` listener and a timer, both dropped on the first reveal.
 *
 * `onReveal` runs right after the iframe becomes visible. Anything that needs a
 * visible iframe, such as focus(), belongs there rather than on a timer.
 *
 * `styles` is applied last, so a caller passing `visibility` overrides the hiding
 * and brings the white flash back.
 */
export function createIframe({ __colors__, __font__, styles, onReveal, ...params }: Params) {
  const iframe = document.createElement('iframe');
  const query = new URLSearchParams(params as Record<string, string>).toString();

  setAttributes(iframe, {
    src: `${BASE_URL}/web/iframe.html?${query}`,
    name: JSON.stringify({ __colors__, __font__ }),
    tabindex: '0',
    title: 'Comments | Remark42',
  });
  setStyles(iframe, {
    height: '100%',
    width: '100%',
    border: 'none',
    padding: 0,
    margin: 0,
    overflow: 'hidden',
    colorScheme: params.theme === 'dark' ? 'dark' : 'light',
    // an iframe whose color-scheme differs from its document's gets an opaque canvas
    // painted in the document's scheme, and browsers paint a default surface before
    // the document is parsed at all, which shows as a white flash on dark host pages.
    // reveal only on `inited`, which the document posts from its body script; the head
    // script has applied the matching scheme by then, so the two always agree on reveal.
    visibility: 'hidden',
    ...styles,
  });

  hideUntilInited(iframe, onReveal);

  return iframe;
}

function hideUntilInited(iframe: HTMLIFrameElement, onReveal?: () => void) {
  function reveal() {
    window.removeEventListener('message', handleMessage);
    window.clearTimeout(timeout);
    iframe.style.visibility = 'visible';
    onReveal?.();
  }

  function handleMessage(event: MessageEvent) {
    if (event.source === iframe.contentWindow && parseMessage(event).inited === true) {
      reveal();
    }
  }

  const timeout = window.setTimeout(reveal, REVEAL_TIMEOUT);

  window.addEventListener('message', handleMessage);
}
