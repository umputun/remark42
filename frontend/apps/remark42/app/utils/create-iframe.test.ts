import { createIframe } from './create-iframe';

const REVEAL_TIMEOUT = 5000;

function postInited(source: MessageEventSource | null) {
  window.dispatchEvent(new MessageEvent('message', { data: { inited: true }, source }));
}

describe('createIframe', () => {
  beforeEach(() => {
    jest.useFakeTimers();
    document.body.innerHTML = '';
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  it('starts hidden', () => {
    const iframe = createIframe({ site_id: 'remark' });
    expect(iframe.style.visibility).toBe('hidden');
  });

  it('lets caller styles override visibility', () => {
    const iframe = createIframe({ site_id: 'remark', styles: { visibility: 'visible' } });
    expect(iframe.style.visibility).toBe('visible');
  });

  it('reveals when its own document reports inited', () => {
    const iframe = createIframe({ site_id: 'remark' });
    document.body.appendChild(iframe);

    postInited(iframe.contentWindow);
    expect(iframe.style.visibility).toBe('visible');
  });

  it('ignores inited from a foreign source', () => {
    const iframe = createIframe({ site_id: 'remark' });
    const other = document.createElement('iframe');
    document.body.append(iframe, other);

    postInited(other.contentWindow);
    expect(iframe.style.visibility).toBe('hidden');

    postInited(window);
    expect(iframe.style.visibility).toBe('hidden');
  });

  it('reveals on timeout when inited never arrives', () => {
    const iframe = createIframe({ site_id: 'remark' });
    document.body.appendChild(iframe);

    jest.advanceTimersByTime(REVEAL_TIMEOUT - 1);
    expect(iframe.style.visibility).toBe('hidden');

    jest.advanceTimersByTime(1);
    expect(iframe.style.visibility).toBe('visible');
  });

  it('calls onReveal after the iframe becomes visible', () => {
    const onReveal = jest.fn(() => {
      expect(iframe.style.visibility).toBe('visible');
    });
    const iframe = createIframe({ site_id: 'remark', onReveal });
    document.body.appendChild(iframe);

    postInited(iframe.contentWindow);
    expect(onReveal).toHaveBeenCalledTimes(1);
  });

  it('calls onReveal on the timeout path too', () => {
    const onReveal = jest.fn();
    createIframe({ site_id: 'remark', onReveal });

    jest.advanceTimersByTime(REVEAL_TIMEOUT);
    expect(onReveal).toHaveBeenCalledTimes(1);
  });

  it('drops the listener and the timer on reveal', () => {
    const onReveal = jest.fn();
    const iframe = createIframe({ site_id: 'remark', onReveal });
    document.body.appendChild(iframe);

    postInited(iframe.contentWindow);
    postInited(iframe.contentWindow);
    jest.advanceTimersByTime(REVEAL_TIMEOUT);

    expect(onReveal).toHaveBeenCalledTimes(1);
  });

  it('does not put onReveal into the iframe query', () => {
    const iframe = createIframe({ site_id: 'remark', onReveal: () => undefined });
    expect(iframe.getAttribute('src')).not.toContain('onReveal');
  });
});
