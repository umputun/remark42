import { Color, color, parseColorStr } from 'utils/colors';
import { isStr, isUnknownDict } from 'utils/types';

export interface ThemeStyles {
  colors?: ThemeStyleColors;
}

export const isThemeStyles = (val: unknown): val is ThemeStyles => {
  if (!isUnknownDict(val)) return false;
  if (val.colors && !isThemeStylesColors(val.colors)) return false;
  return true;
};

export interface ThemeStyleColors {
  primary?: string;
}

const isThemeStylesColors = (val: unknown): val is ThemeStyleColors => {
  if (!isUnknownDict(val)) return false;
  if (val.primary && !isStr(val.primary)) return false;
  return true;
};

export const setThemeStyles = (styles: ThemeStyles) => {
  if (styles.colors) {
    setColors(styles.colors);
  }
};

const setColors = (colors: ThemeStyleColors) => {
  const { primary: primaryStr } = colors;

  // Primary color
  if (primaryStr) {
    const primary = parseColorStr(primaryStr);
    if (primary) {
      setPrimaryColor(primary);
    } else {
      console.error('Invalid primary color format: ', primaryStr);
    }
  }
};

const setPrimaryColor = (val: Color) => {
  const rootEl = document.documentElement;
  const darkRootEl = document.querySelector('.root.dark');
  const primary = color(val); // #0aa, rgb(0, 170, 170)
  /* Numerid variables  */
  rootEl.style.setProperty('--color9', primary.hex()); // #0aa;
  rootEl.style.setProperty('--color15', primary.darken(0.1).hex()); // #099, rgb(0, 153, 153)
  rootEl.style.setProperty('--color33', primary.lighten(0.3).hex()); // #06c5c5, rgb(6,197,197) (equivalent rgb(77, 196, 196));
  rootEl.style.setProperty('--color40', primary.lighten(0.6).hex()); // #9cdddb, rgb(156,221,219) (equivalent rgb(153, 221, 221));
  rootEl.style.setProperty('--color43', primary.lighten(0.7).hex()); // #b7dddd, 	rgb(183,221,221)
  rootEl.style.setProperty('--color42', primary.lighten(0.8).hex()); // #c6efef, rgb(198,239,239)
  rootEl.style.setProperty('--color48', primary.darken(0.1).alpha(0.6).rgb()); // rgba(37, 156, 154, 0.6)
  rootEl.style.setProperty('--color47', primary.darken(0.1).alpha(0.4).rgb()); // rgba(37, 156, 154, 0.4)
  /* Named variables */
  rootEl.style.setProperty('--primary-color', primary.rgbBody()); // rgb(0, 170, 170)
  rootEl.style.setProperty('--primary-brighter-color', primary.darken(0.1).rgbBody()); // rgb(0, 153, 153);
  rootEl.style.setProperty('--primary-darker-color', primary.darken(0.4).rgbBody()); // rgb(0, 102, 102);
  if (darkRootEl instanceof HTMLElement) {
    darkRootEl.style.setProperty('--primary-color', primary.darken(0.1).rgbBody()); // rgb(0, 153, 153)
    darkRootEl.style.setProperty('--primary-brighter-color', primary.rgbBody()); // rgb(0, 170, 170)
  }
};
