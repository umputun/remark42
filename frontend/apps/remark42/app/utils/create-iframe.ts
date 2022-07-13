import { BASE_URL } from 'common/constants.config';
import { setStyles, setAttributes, StylesDeclaration } from 'utils/set-dom-props';

type Params = { [key: string]: unknown; __colors__?: Record<string, string>; styles?: StylesDeclaration };

export function createIframe({ __colors__, styles, ...params }: Params) {
  const iframe = document.createElement('iframe');
  const query = new URLSearchParams(params as Record<string, string>).toString();

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
    ...styles,
  });

  return iframe;
}
