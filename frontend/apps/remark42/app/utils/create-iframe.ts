import { BASE_URL } from 'common/constants.config';
import { ThemeStyling, themeStylingToUrlSearchParams } from 'common/theme';
import { setAttributes, setStyles, StylesDeclaration } from 'utils/set-dom-props';

type Params = {
  [key: string]: unknown;
  __colors__?: Record<string, string>;
  styles?: StylesDeclaration;
  styling?: ThemeStyling;
};

export function createIframe({ __colors__, styles, styling, ...params }: Params) {
  const iframe = document.createElement('iframe');
  console.log(params);
  const query = new URLSearchParams({
    ...(params as Record<string, string>),
    ...themeStylingToUrlSearchParams(styling),
  }).toString();

  setAttributes(iframe, {
    src: `${BASE_URL}/web/iframe.html?${query}`,
    name: JSON.stringify({ __colors__ }),
    frameborder: '0',
    allowtransparency: 'true',
    scrolling: 'no',
    tabindex: '0',
    title: 'Comments | Remark42',
    horizontalscrolling: 'no',
    verticalscrolling: 'no',
  });
  setStyles(iframe, {
    height: '100%',
    width: '100%',
    border: 'none',
    padding: 0,
    margin: 0,
    overflow: 'hidden',
    colorScheme: 'none',
    ...styles,
  });

  return iframe;
}
