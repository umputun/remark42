/**
 * The widget document of an instance mounted under a path, so a request built from the origin
 * alone, or from the configured host, would not match.
 *
 * @jest-environment-options {"url": "https://remark.example.com/comments/web/iframe.html?site_id=remark"}
 */
import fetchMock from 'jest-fetch-mock';

/** a fresh module each time, as the table is kept for the life of the document */
async function freshLoader() {
  jest.resetModules();

  return (await import('./emoji-data')).loadEmoji;
}

describe('loadEmoji', () => {
  beforeEach(() => {
    fetchMock.resetMocks();
  });

  it('asks for the table next to the document it runs in', async () => {
    fetchMock.mockResponseOnce(JSON.stringify({ smile: '😄' }));
    const loadEmoji = await freshLoader();

    await expect(loadEmoji()).resolves.toEqual({ smile: '😄' });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe('https://remark.example.com/comments/web/emoji.json');
  });

  it('fetches the table once however often it is asked for', async () => {
    fetchMock.mockResponseOnce(JSON.stringify({ smile: '😄' }));
    const loadEmoji = await freshLoader();

    const [first, second] = await Promise.all([loadEmoji(), loadEmoji()]);
    await loadEmoji();

    expect(second).toBe(first);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it('rejects when the server does not have the table', async () => {
    fetchMock.mockResponseOnce('not found', { status: 404 });
    const loadEmoji = await freshLoader();

    await expect(loadEmoji()).rejects.toThrow('404');
  });

  it('asks again after a failed load', async () => {
    fetchMock.mockRejectOnce(new Error('offline'));
    fetchMock.mockResponseOnce(JSON.stringify({ smile: '😄' }));
    const loadEmoji = await freshLoader();

    await expect(loadEmoji()).rejects.toThrow('offline');
    await expect(loadEmoji()).resolves.toEqual({ smile: '😄' });
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});
