import { colorToHexStr, colorToNumStr, colorToRgbStr, darkenColor, lightenColor, parseColorStr } from 'utils/colors';
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
  console.log('[+]: setThemeStyle', styles);
  if (styles.colors) {
    setThemeStylesColors(styles.colors);
  }
};

const setThemeStylesColors = (colors: ThemeStyleColors) => {
  const { primary: primaryStr } = colors;
  const rootEl = document.documentElement;
  const darkRootEl = document.querySelector('.root.dark');
  // Primary color
  if (primaryStr) {
    const primary = parseColorStr(primaryStr); // #0aa, rgb(0, 170, 170)
    if (primary) {
      /* Numerid variables  */
      rootEl.style.setProperty('--color9', colorToHexStr(primary)); // #0aa;
      rootEl.style.setProperty('--color15', colorToHexStr(darkenColor(primary, 0.1))); // #099, rgb(0, 153, 153)
      rootEl.style.setProperty('--color33', colorToHexStr(lightenColor(primary, 0.3))); // #06c5c5, rgb(6,197,197) (equivalent rgb(77, 196, 196));
      rootEl.style.setProperty('--color40', colorToHexStr(lightenColor(primary, 0.6))); // #9cdddb, rgb(156,221,219) (equivalent rgb(153, 221, 221));
      rootEl.style.setProperty('--color43', colorToHexStr(lightenColor(primary, 0.7))); // #b7dddd, 	rgb(183,221,221)
      rootEl.style.setProperty('--color42', colorToHexStr(lightenColor(primary, 0.8))); // #c6efef, rgb(198,239,239)
      rootEl.style.setProperty('--color48', colorToRgbStr({ ...darkenColor(primary, 0.1), a: 0.6 })); // rgba(37, 156, 154, 0.6)
      rootEl.style.setProperty('--color47', colorToRgbStr({ ...darkenColor(primary, 0.1), a: 0.4 })); // rgba(37, 156, 154, 0.4)
      /* Named variables */
      const dark = darkenColor(primary, 0.1); // rgb(0, 153, 153);
      const moreDark = darkenColor(primary, 0.4); // rgb(0, 102, 102);
      rootEl.style.setProperty('--primary-color', colorToNumStr(primary)); // rgb(0, 170, 170)
      rootEl.style.setProperty('--primary-brighter-color', colorToNumStr(dark)); // rgb(0, 153, 153);
      rootEl.style.setProperty('--primary-darker-color', colorToNumStr(moreDark)); // rgb(0, 102, 102);
      if (darkRootEl instanceof HTMLElement) {
        darkRootEl.style.setProperty('--primary-color', colorToNumStr(dark)); // rgb(0, 153, 153)
        darkRootEl.style.setProperty('--primary-brighter-color', colorToNumStr(primary)); // rgb(0, 170, 170)
      }
    } else {
      console.error('Invalid primary color format: ', primaryStr);
    }
  }
};
