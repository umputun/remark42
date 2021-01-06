import jestFetchMock from 'jest-fetch-mock';

import { emailVerificationForSubscribe } from './api';

jest.mock('common/constants', () => ({
  BASE_URL: 'https://example.com',
  API_BASE: '/api',
}));

jest.mock('common/settings', () => ({
  siteId: 'remark42',
}));

describe('api', () => {
  beforeAll(() => {
    jestFetchMock.enableMocks();
  });

  afterAll(() => {
    jestFetchMock.disableMocks();
  });

  beforeEach(() => {
    jestFetchMock.resetMocks();
  });
  it('should send request with encoded email', async () => {
    await emailVerificationForSubscribe("address.!#$%&'*+-/=?^_`{|}~(),:;<>[\\]@example.com");

    expect(jestFetchMock.mock.calls.length).toEqual(1);

    const url = jestFetchMock.mock.calls[0][0] as string;
    const match = url.match(/address=(\S+)$/) as string[];

    expect(Array.isArray(match)).toBe(true);
    expect(match.length).toBeGreaterThan(1);
    expect(match[1]).toBe(
      "address.!%23%24%25%26'*%2B-%2F%3D%3F%5E_%60%7B%7C%7D~()%2C%3A%3B%3C%3E%5B%5C%5D%40example.com"
    );
  });
});
