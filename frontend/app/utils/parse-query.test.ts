import { parseQuery } from './parse-query';

describe('parseQuery', () => {
  it('should return empty object', () => {
    expect(parseQuery('')).toEqual({});
    expect(parseQuery('?')).toEqual({});
  });

  it('should add empty field to object', () => {
    expect(parseQuery('?a')).toEqual({ a: '' });
  });

  it('should add empty field and field with param to object', () => {
    expect(parseQuery('?a&b=1')).toEqual({ a: '', b: '1' });
  });

  it('should add all params to object', () => {
    expect(parseQuery('?a=1&b=1')).toEqual({ a: '1', b: '1' });
  });

  it('should convert urlencoded param', () => {
    expect(parseQuery('?x=%D1%8B%D1%84%D0%B2%D0%B0%D1%84%D1%8B%D0%B2%D1%84%D1%8B')).toEqual({ x: 'ыфвафывфы' });
  });
});
