import {
  Color,
  color,
  colorToHexStr,
  colorToRgbBodyStr,
  colorToRgbStr,
  darkenColor,
  lightenColor,
  parseColorStr,
} from './colors';

describe('color function', () => {
  it('should return object with properties', () => {
    const val = color({ r: 100, g: 150, b: 200, a: 1 });
    expect(val).toHaveProperty('alpha');
    expect(val).toHaveProperty('lighten');
    expect(val).toHaveProperty('darken');
    expect(val).toHaveProperty('hex');
    expect(val).toHaveProperty('rgb');
    expect(val).toHaveProperty('rgbBody');
    expect(val).toHaveProperty('object');
  });

  it('should throw an error if the color object is invalid', () => {
    expect(() => color({ r: 100, g: 150, b: 200, a: 1 })).not.toThrow();
    expect(() => color('invalid value')).toThrow();
  });

  it('should lighten a color', () => {
    expect(color('#000').lighten(0.5).hex()).toEqual('#808080');
  });

  it('should darken a color', () => {
    expect(color('#fff').darken(0.5).hex()).toEqual('#808080');
  });

  it('should set the alpha value of a color', () => {
    expect(color('#000').alpha(0.5).object()).toEqual({ r: 0, g: 0, b: 0, a: 0.5 });
  });

  it('should convert a color to a hex string', () => {
    expect(color({ r: 100, g: 150, b: 200, a: 1 }).hex()).toEqual('#6496c8');
  });

  it('should convert a color to an RGB string', () => {
    expect(color({ r: 100, g: 150, b: 200, a: 1 }).rgb()).toEqual('rgb(100,150,200)');
  });

  it('should convert a color to an RGB body string', () => {
    expect(color({ r: 100, g: 150, b: 200, a: 1 }).rgbBody()).toEqual('100,150,200');
  });
});

describe('parseColorStr', () => {
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
    const val = { r: 100, g: 150, b: 200, a: 1 }; // rgb(100, 150, 200)
    const amount = 0.5;
    const lightenedColor = lightenColor(val, amount); // rgb(178, 203, 228)
    expect(lightenedColor.r).toBe(178);
    expect(lightenedColor.g).toBe(203);
    expect(lightenedColor.b).toBe(228);
    expect(lightenedColor.a).toBe(1);
  });

  test('should not change a color if amount is 0', () => {
    const val = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0;
    const lightenedColor = lightenColor(val, amount);
    expect(lightenedColor).toEqual(val);
  });

  test('should not exceed a color value of 255', () => {
    const val = { r: 200, g: 240, b: 255, a: 1 }; // rgb(200, 240, 255)
    const amount = 0.5;
    expect(lightenColor(val, amount)).toEqual({ r: 228, g: 248, b: 255, a: 1 }); // rgb(228, 248, 255)
  });

  test('should return a new color object', () => {
    const val = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0.5;
    const lightenedColor = lightenColor(val, amount);
    expect(lightenedColor).not.toBe(val);
  });
});

describe('darkenColor', () => {
  test('should darken a color by the given amount', () => {
    const val = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0.5;
    const darkenedColor = darkenColor(val, amount);
    expect(darkenedColor.r).toBe(50);
    expect(darkenedColor.g).toBe(75);
    expect(darkenedColor.b).toBe(100);
    expect(darkenedColor.a).toBe(1);
  });

  test('should not change a color if amount is 0', () => {
    const val = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0;
    const darkenedColor = darkenColor(val, amount);
    expect(darkenedColor).toEqual(val);
  });

  test('should not exceed a color value of 0', () => {
    const val = { r: 0, g: 20, b: 30, a: 1 };
    const amount = 0.5;
    expect(darkenColor(val, amount)).toEqual({ r: 0, g: 10, b: 15, a: 1 });
  });

  test('should return a new color object', () => {
    const val = { r: 100, g: 150, b: 200, a: 1 };
    const amount = 0.5;
    const darkenedColor = darkenColor(val, amount);
    expect(darkenedColor).not.toBe(val);
  });
});

describe('colorToHexStr', () => {
  it('should convert a color object to a 6-digit hex color code', () => {
    const val: Color = { r: 255, g: 128, b: 0, a: 1 };
    const expectedHexCode = '#ff8000';
    const hexCode = colorToHexStr(val);
    expect(hexCode).toBe(expectedHexCode);
  });

  it('should convert a color object to a 3-digit hex color code', () => {
    const val: Color = { r: 34, g: 68, b: 102, a: 1 };
    const expectedHexCode = '#224466';
    const hexCode = colorToHexStr(val);
    expect(hexCode).toBe(expectedHexCode);
  });

  it('should convert a color object with alpha to an 8-digit hex color code', () => {
    const val: Color = { r: 0, g: 128, b: 255, a: 0.5 };
    const expectedHexCode = '#0080ff80';
    const hexCode = colorToHexStr(val);
    expect(hexCode).toBe(expectedHexCode);
  });
});

describe('colorToRgbStr', () => {
  it('should convert color object to RGB string', () => {
    const result = colorToRgbStr({ r: 255, g: 255, b: 255, a: 1 });
    expect(result).toBe('rgb(255,255,255)');
  });

  it('should convert color object with alpha to RGBA string', () => {
    const colorWithAlpha: Color = { r: 255, g: 255, b: 255, a: 0.5 };
    const result = colorToRgbStr(colorWithAlpha);
    expect(result).toBe('rgba(255,255,255,0.5)');
  });

  it('should convert color object with alpha equal to 1 to RGB string', () => {
    const colorWithAlpha: Color = { r: 255, g: 255, b: 255, a: 1 };
    const result = colorToRgbStr(colorWithAlpha);
    expect(result).toBe('rgb(255,255,255)');
  });
});

describe('colorToRgbBodyStr', () => {
  it('should return a string in RGB(A) format', () => {
    const val: Color = { r: 255, g: 255, b: 255, a: 1 };
    const result = colorToRgbBodyStr(val);
    expect(result).toEqual('255,255,255');
  });

  it('should include alpha channel when it is not 1', () => {
    const val: Color = { r: 255, g: 255, b: 255, a: 0.5 };
    const result = colorToRgbBodyStr(val);
    expect(result).toEqual('255,255,255,0.5');
  });
});
