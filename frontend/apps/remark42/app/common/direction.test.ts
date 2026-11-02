import { getDirection } from './direction';

describe('getDirection', () => {
  it.each(['ar', 'fa', 'he'])('treats %s as rtl', (locale) => {
    expect(getDirection(locale)).toBe('rtl');
  });

  it.each(['en', 'ru', 'de', 'zh-tw'])('treats %s as ltr', (locale) => {
    expect(getDirection(locale)).toBe('ltr');
  });

  it('defaults an unknown locale to ltr', () => {
    expect(getDirection('xx')).toBe('ltr');
  });
});
