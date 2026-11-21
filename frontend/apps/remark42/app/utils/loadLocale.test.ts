import fetchMock from 'jest-fetch-mock';

import { loadLocale } from './loadLocale';

describe('loadLocale', () => {
  beforeEach(() => {
    fetchMock.resetMocks();
  });

  it('fetches a supported catalogue from /web on the instance', async () => {
    fetchMock.mockResponseOnce(JSON.stringify({ 'auth.back': 'Назад' }));

    await expect(loadLocale('ru')).resolves.toEqual({ 'auth.back': 'Назад' });
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe('http://test.com/web/ru.json');
  });

  it('fetches a code that is not two letters, such as zh-tw', async () => {
    fetchMock.mockResponseOnce(JSON.stringify({ 'auth.back': '返回' }));

    await expect(loadLocale('zh-tw')).resolves.toEqual({ 'auth.back': '返回' });
    expect(fetchMock.mock.calls[0][0]).toBe('http://test.com/web/zh-tw.json');
  });

  // the locale comes from the host page, so anything off the list must never reach a URL
  it.each(['en', 'xx', 'RU', 'ru ', '../api/v1/config', 'emoji', ''])(
    'requests nothing for %j and leaves the English text',
    async (locale) => {
      await expect(loadLocale(locale)).resolves.toEqual({});
      expect(fetchMock).not.toHaveBeenCalled();
    }
  );

  // a JSON error body, as a proxy may send, so only the status tells it from a catalogue
  it('falls back to English when the catalogue is missing', async () => {
    fetchMock.mockResponseOnce(JSON.stringify({ error: 'not found' }), { status: 404 });

    await expect(loadLocale('de')).resolves.toEqual({});
  });

  it('falls back to English when the request fails', async () => {
    fetchMock.mockRejectOnce(new Error('blocked by CORS'));

    await expect(loadLocale('de')).resolves.toEqual({});
  });

  it('falls back to English when the body is not JSON', async () => {
    fetchMock.mockResponseOnce('<html>proxy error</html>');

    await expect(loadLocale('de')).resolves.toEqual({});
  });
});
