/**
 * jsdom seals window.location, so the https protocol these tests need is set through the
 * environment URL rather than by redefining the property
 *
 * @jest-environment-options {"url": "https://test.com"}
 */
import { getBaseUrl } from './constants.config';

describe('constants.config', () => {
  let host: string;
  let consoleErrorSpy: jest.SpyInstance;

  beforeAll(() => {
    host = window.remark_config.host!;
  });
  beforeEach(() => {
    consoleErrorSpy = jest.spyOn(console, 'error').mockImplementationOnce(jest.fn());
  });
  afterEach(() => {
    consoleErrorSpy.mockClear();
    window.remark_config.host = host;
  });

  describe('BASE_URL validation', () => {
    it('should throw error if host is not defined', () => {
      window.remark_config.host = undefined;
      expect(() => getBaseUrl()).toThrow(`Remark42: remark_config.host wasn't configured.`);
    });
    it('should show mismatch error', () => {
      expect(getBaseUrl()).toBe('http://test.com');
      expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
      expect(consoleErrorSpy).toHaveBeenCalledWith('Remark42: Protocol mismatch.');
    });
    it('should throw error when BASE_URL has wrong protocol', () => {
      window.remark_config.host = 'data:application/json;base64';
      expect(() => getBaseUrl()).toThrow('Remark42: Invalid host URL.');
      expect(consoleErrorSpy).toHaveBeenCalledTimes(2);
      expect(consoleErrorSpy).toHaveBeenNthCalledWith(1, 'Remark42: Protocol mismatch.');
      expect(consoleErrorSpy).toHaveBeenNthCalledWith(2, 'Remark42: Wrong protocol in host URL.');
    });
    it('should throw error when BASE_URL is invalid', () => {
      window.remark_config.host = 'asfasdfa!asds';
      expect(() => getBaseUrl()).toThrow('Remark42: Invalid host URL.');
      expect(consoleErrorSpy).toHaveBeenCalledTimes(0);
    });
  });
});
