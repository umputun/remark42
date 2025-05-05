/**
 * Represents a color with RGBA components.
 *
 * @typedef {Object} Color
 * @property {number} r - The red component of the color (0-255).
 * @property {number} g - The green component of the color (0-255).
 * @property {number} b - The blue component of the color (0-255).
 * @property {number} a - The alpha component of the color (0-1).
 */
export interface Color {
  r: number;
  g: number;
  b: number;
  a: number;
}

export interface DualColors {
  light: Color;
  dark: Color;
}

/**
 * Creates a color object that provides utility methods for manipulating and converting color values.
 *
 * @function
 * @param {string | Color} inputValue - A string representing a CSS color value, or an object representing a color with RGBA components.
 * @returns {object} - An object containing utility methods for manipulating and converting color values.
 * @throws {Error} - If the provided color value is invalid.
 *
 * @example
 * const myColor = color('#ff0000');
 * const lighterColor = myColor.lighten(0.2).hex(); // returns a hex string representation of the original color, lightened by 20%
 * const rgbaValue = myColor.rgb(); // returns an RGBA string representation of the original color
 * const alphaValue = myColor.alpha(0.5).object(); // returns an object representing the original color with its alpha value set to 0.5
 */
export const color = (inputValue: string | Color) => {
  const parts = typeof inputValue === 'string' ? parseColorStr(inputValue) : inputValue;
  if (!parts) {
    throw new Error(`Invalid color value: ${inputValue}`);
  }

  const lighten = (amount: number) => color(lightenColor(parts, amount));
  const darken = (amount: number) => color(darkenColor(parts, amount));
  const alpha = (amount: number) => color({ ...parts, a: amount });
  const hex = () => colorToHexStr(parts);
  const rgb = () => colorToRgbStr(parts);
  const rgbBody = () => colorToRgbBodyStr(parts);
  const object = () => ({ ...parts });

  return { alpha, lighten, darken, hex, rgb, rgbBody, object };
};

/**
 * Parses a color string in the format of a 3-digit or 6-digit hexadecimal color code
 * or an RGB(A) color code and returns an object representing the color's RGB values.
 * @param {string} color - The color string to parse.
 * @returns {Color|undefined} An object representing the color's RGB values, or `undefined` if the color string is invalid.
 */
export const parseColorStr = (rawColor: string): Color | undefined => {
  const color = rawColor.toLowerCase().trim();
  if (color.charAt(0) === '#') {
    if (color.length === 4) {
      const r = parseInt(color.charAt(1) + color.charAt(1), 16);
      const g = parseInt(color.charAt(2) + color.charAt(2), 16);
      const b = parseInt(color.charAt(3) + color.charAt(3), 16);
      return { r, g, b, a: 1 };
    } else if (color.length === 7) {
      const r = parseInt(color.substr(1, 2), 16);
      const g = parseInt(color.substr(3, 2), 16);
      const b = parseInt(color.substr(5, 2), 16);
      return { r, g, b, a: 1 };
    }
  } else if (color.startsWith('rgb(') && color.endsWith(')')) {
    const rgbValues = color.substring(4, color.length - 1).split(',');
    if (rgbValues.length === 3) {
      const r = parseInt(rgbValues[0], 10);
      const g = parseInt(rgbValues[1], 10);
      const b = parseInt(rgbValues[2], 10);
      return { r, g, b, a: 1 };
    }
  } else if (color.startsWith('rgba(') && color.endsWith(')')) {
    const rgbaValues = color.substring(5, color.length - 1).split(',');
    if (rgbaValues.length === 4) {
      const r = parseInt(rgbaValues[0], 10);
      const g = parseInt(rgbaValues[1], 10);
      const b = parseInt(rgbaValues[2], 10);
      const a = parseFloat(rgbaValues[3]);
      return { r, g, b, a };
    }
  }
  return undefined;
};

/**
 * Lightens a given color by a specified amount.
 *
 * @param {Color} color - The color to be lightened.
 * @param {number} amount - The amount by which the color should be lightened.
 * @returns {Color} A new color object representing the lightened color.
 */
export const lightenColor = (color: Color, amount: number): Color => {
  const { r, g, b, a } = color;
  const amt = Math.abs(amount);
  const red = Math.round(Math.min(r + (255 - r) * amt, 255));
  const green = Math.round(Math.min(g + (255 - g) * amt, 255));
  const blue = Math.round(Math.min(b + (255 - b) * amt, 255));
  return { r: red, g: green, b: blue, a };
};

/**
 * Darkens a given color by a specified amount.
 *
 * @param {Color} color - The color to be darkened.
 * @param {number} amount - The amount by which the color should be darkened.
 *   A value of 0 will not change the color, while positive values will darken the color
 *   and negative values will lighten the color.
 * @returns {Color} A new color object representing the darkened color.
 */
export const darkenColor = (color: Color, amount: number): Color => {
  const { r, g, b, a } = color;
  const amt = Math.abs(amount);
  const red = Math.round(Math.max(r - r * amt, 0));
  const green = Math.round(Math.max(g - g * amt, 0));
  const blue = Math.round(Math.max(b - b * amt, 0));
  return { r: red, g: green, b: blue, a };
};

/**
 * Converts a given color object to a hexadecimal color code.
 *
 * @param {Color} color - The color object to convert to a hexadecimal color code.
 * @returns {string} A hexadecimal color code representation of the input color.
 */
export const colorToHexStr = (color: Color): string => {
  const { r, g, b, a } = color;
  let str: string = `#${pad(r.toString(16), 2, '0')}${pad(g.toString(16), 2, '0')}${pad(b.toString(16), 2, '0')}`;
  if (a !== 1) {
    str += `${pad(Math.round(a * 255).toString(16), 2, '0')}`;
  }
  return str;
};

/**
 * Converts a color object into its equivalent RGB string representation.
 * @param {Color} color - The color object to convert.
 * @returns {string} The RGB string representation of the input color object.
 */
export const colorToRgbStr = (color: Color): string => {
  const { r, g, b, a } = color;
  return `rgb${a !== 1 ? 'a' : ''}(${r},${g},${b}${a !== 1 ? `,${a}` : ''})`;
};

/**
 * Converts a color object into its equivalent RGB(A) body string representation.
 * @param {Color} color - The color object to convert.
 * @returns {string} The RGB(A) body string representation of the input color object.
 */
export const colorToRgbBodyStr = (color: Color): string => {
  const { r, g, b, a } = color;
  let str = `${r},${g},${b}`;
  if (a !== 1) {
    str += `,${a}`;
  }
  return str;
};

const pad = (str: string, len: number, char: string): string => {
  while (str.length < len) {
    str = char + str;
  }
  return str;
};
