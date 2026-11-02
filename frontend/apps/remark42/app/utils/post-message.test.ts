import { isFromParent } from './post-message';

/**
 * The widget document acts on `signout` and `theme` straight out of a message, so the sender has to
 * be the page embedding it. The origin cannot stand in for that check: the host is whatever site
 * embeds the widget, and `ALLOWED_HOSTS` is enforced server side through `frame-ancestors`.
 */
describe('isFromParent', () => {
  it('accepts the embedding page', () => {
    expect(isFromParent(new MessageEvent('message', { source: window.parent }))).toBe(true);
  });

  it('rejects another frame on the page', () => {
    const foreign = document.createElement('iframe');
    document.body.appendChild(foreign);

    expect(isFromParent(new MessageEvent('message', { source: foreign.contentWindow }))).toBe(false);

    foreign.remove();
  });

  it('rejects a message carrying no sender', () => {
    // an extension posting into the frame is the common case
    expect(isFromParent(new MessageEvent('message', { data: { signout: true } }))).toBe(false);
  });
});
