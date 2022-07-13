import { validToken, invalidToken } from '__stubs__/jwt';

import { parseJwt, isJwtExpired } from './jwt';

describe('JWT', () => {
  describe('parseJWT', () => {
    it('should parse token', () => {
      expect(parseJwt(validToken)).toEqual({
        aud: 'remark',
        exp: 1579986982,
        handshake: {
          id: 'dev_user::asd@x101.pw',
        },
        iss: 'remark42',
        nbf: 1579985122,
      });
    });

    it('should throw error', () => {
      expect(() => parseJwt(invalidToken)).toThrowError('The string to be decoded contains invalid characters.');
    });
  });

  describe('isJwtExpired', () => {
    const now = jest
      .fn()
      .mockImplementationOnce(() => 1579986981 * 1000)
      .mockImplementationOnce(() => 1579986982 * 1000)
      .mockImplementationOnce(() => 1579986983 * 1000);

    Object.defineProperty(window, 'Data', { value: { now } });

    it('should be not expired', () => {
      expect(isJwtExpired(validToken)).toBe(true);
      expect(isJwtExpired(validToken)).toBe(true);
    });

    it('should be expired', () => {
      expect(isJwtExpired(validToken)).toBe(true);
    });
  });
});
