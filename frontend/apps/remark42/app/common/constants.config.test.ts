/**
 * The validation rules live in `base-url.ts` and are covered there against arbitrary input. What
 * only a browser can show is the wiring: that the host an integrator set and the protocol of the
 * page the widget is on both reach the validator.
 *
 * jsdom seals window.location, so the protocol comes from the environment URL.
 *
 * @jest-environment-options {"url": "https://test.com"}
 */
import { getBaseUrl } from './constants.config';

describe('getBaseUrl', () => {
  let host: string;

  beforeAll(() => {
    host = window.remark_config.host!;
  });
  afterEach(() => {
    window.remark_config.host = host;
  });

  it('reads the host out of remark_config', () => {
    window.remark_config.host = 'https://configured.example.com';

    expect(getBaseUrl()).toBe('https://configured.example.com');
  });

  it('judges the host against the protocol of the page it is on', () => {
    const consoleErrorSpy = jest.spyOn(console, 'error').mockImplementation();

    window.remark_config.host = 'http://configured.example.com';
    getBaseUrl();

    // the page is https here, so an http host is a mismatch. a hard-coded protocol would not see it
    expect(consoleErrorSpy).toHaveBeenCalledWith('Remark42: Protocol mismatch.');
    consoleErrorSpy.mockRestore();
  });
});
