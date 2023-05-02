import { Color, colorToHexStr, darkenColor, lightenColor, parseColorStr } from './colors';

describe('parseColorStr function', () => {
  it('should parse a 3-digit hex color code', () => {
    expect(parseColorStr('#abc')).toEqual({ r: 170, g: 187, b: 204, a: 1 });
  });

  it('should parse a 6-digit hex color code', () => {
    expect(parseColorStr('#abcdef')).toEqual({ r: 171, g: 205, b: 239, a: 1 });
  });

  it('should parse an RGB color code', () => {
    expect(parseColorStr('rgb(255, 0, 128)')).toEqual({ r: 255, g: 0, b: 128, a: 1 });
  });

  it('should parse an RGBA color code', () => {
    expect(parseColorStr('rgba(100, 200, 150, 0.5)')).toEqual({ r: 100, g: 200, b: 150, a: 0.5 });
  });

  it('should return default color for an invalid color code', () => {
    const colorCode = 'invalid color';
    expect(parseColorStr(colorCode)).toEqual(undefined);
  });
});

describe('lightenColor', () => {
  test('should lighten a color by the given amount', () => {
    const color = { r: 100, g: 150, b: 200, a: 1 }; // rgb(100, 150, 200)
    const amount = 0.5;
    const lightenedColor = lightenColor(color, amount); // rgb(178, 203, 228)
    expect(lightenedColor.r).toBe(178);
    expect(lightenedColor.g).toBe(203);
    expect(lightenedColor.b).toBe(228);
    expect(lightenedColor.a).toBe(1);
  });

  test('should not change a color if amount is 0', () => {
    const color = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0;
    const lightenedColor = lightenColor(color, amount);
    expect(lightenedColor).toEqual(color);
  });

  test('should not exceed a color value of 255', () => {
    const color = { r: 200, g: 240, b: 255, a: 1 }; // rgb(200, 240, 255)
    const amount = 0.5;
    expect(lightenColor(color, amount)).toEqual({ r: 228, g: 248, b: 255, a: 1 }); // rgb(228, 248, 255)
  });

  test('should return a new color object', () => {
    const color = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0.5;
    const lightenedColor = lightenColor(color, amount);
    expect(lightenedColor).not.toBe(color);
  });
});

describe('darkenColor', () => {
  test('should darken a color by the given amount', () => {
    const color = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0.5;
    const darkenedColor = darkenColor(color, amount);
    expect(darkenedColor.r).toBe(50);
    expect(darkenedColor.g).toBe(75);
    expect(darkenedColor.b).toBe(100);
    expect(darkenedColor.a).toBe(1);
  });

  test('should not change a color if amount is 0', () => {
    const color = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0;
    const darkenedColor = darkenColor(color, amount);
    expect(darkenedColor).toEqual(color);
  });

  test('should not exceed a color value of 255', () => {
    const color = { r: 200, g: 240, b: 255, a: 1 }; // rgb(200, 240, 255)
    const amount = 0.5;
    const darkenedColor = darkenColor(color, amount); // rgb(100, 120, 128)
    expect(darkenedColor.r).toBe(100);
    expect(darkenedColor.g).toBe(120);
    expect(darkenedColor.b).toBe(128);
    expect(darkenedColor.a).toBe(1);
  });

  test('should return a new color object', () => {
    const color = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0.5;
    const darkenedColor = darkenColor(color, amount);
    expect(darkenedColor).not.toBe(color);
  });
});

describe('colorToHexStr function', () => {
  it('should convert a color object to a 6-digit hex color code', () => {
    const color: Color = { r: 255, g: 128, b: 0, a: 1 };
    const expectedHexCode = '#ff8000';
    const hexCode = colorToHexStr(color);
    expect(hexCode).toBe(expectedHexCode);
  });

  it('should convert a color object to a 3-digit hex color code', () => {
    const color: Color = { r: 34, g: 68, b: 102, a: 1 };
    const expectedHexCode = '#224466';
    const hexCode = colorToHexStr(color);
    expect(hexCode).toBe(expectedHexCode);
  });

  it('should convert a color object with alpha to an 8-digit hex color code', () => {
    const color: Color = { r: 0, g: 128, b: 255, a: 0.5 };
    const expectedHexCode = '#0080ff80';
    const hexCode = colorToHexStr(color);
    expect(hexCode).toBe(expectedHexCode);
  });
});
