import { themeStylingFromUrlSearchParams, themeStylingToUrlSearchParams } from './theme';

describe('themeStylingToUrlSearchParams', () => {
  test('returns an empty object when styling is undefined', () => {
    expect(themeStylingToUrlSearchParams(undefined)).toEqual({});
  });

  test('converts primary color string to "colorsPrimary" parameter', () => {
    const styling = { colors: { primary: '#ff0000' } };
    const params = themeStylingToUrlSearchParams(styling);
    expect(params).toEqual({ colorsPrimary: '#ff0000' });
  });

  test('converts primary color object with light/dark properties to "colorsPrimary", "colorsPrimaryLight", and "colorsPrimaryDark" parameters', () => {
    const styling = { colors: { primary: { light: '#ffffff', dark: '#000000' } } };
    const params = themeStylingToUrlSearchParams(styling);
    expect(params).toEqual({
      colorsPrimaryLight: '#ffffff',
      colorsPrimaryDark: '#000000',
    });
  });
});

describe('themeStylingFromUrlSearchParams', () => {
  it('returns undefined when params is empty', () => {
    const result = themeStylingFromUrlSearchParams({});
    expect(result).toBeUndefined();
  });

  it('returns theme styling object with colors when params contains colorsPrimaryLight and colorsPrimaryDark', () => {
    const params = {
      colorsPrimaryLight: '#fff',
      colorsPrimaryDark: '#000',
    };
    const result = themeStylingFromUrlSearchParams(params);
    expect(result).toEqual({
      colors: {
        primary: {
          light: '#fff',
          dark: '#000',
        },
      },
    });
  });

  it('returns theme styling object with colors when params contains colorsPrimary', () => {
    const params = {
      colorsPrimary: '#fff',
    };
    const result = themeStylingFromUrlSearchParams(params);
    expect(result).toEqual({
      colors: {
        primary: '#fff',
      },
    });
  });

  it('returns undefined when params does not contain colorsPrimary or colorsPrimaryLight and colorsPrimaryDark', () => {
    const params = {
      someOtherParam: 'value',
    };
    const result = themeStylingFromUrlSearchParams(params);
    expect(result).toBeUndefined();
  });
});
