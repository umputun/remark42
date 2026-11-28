import { SVG_NS, svgElement } from '../../common/dom.js';

/**
 * The verification badge, drawn as the preact VerificationIcon drew it; extra attributes follow
 * the size and shape ones.
 *
 * @param {number} [size]
 * @param {Record<string, string | undefined>} [attributes]
 */
export function verificationIcon(size = 12, attributes = {}) {
  return svgElement(
    'svg',
    { xmlns: SVG_NS, width: String(size), height: String(size), viewBox: '0 0 16 16', ...attributes },
    [
      svgElement('path', {
        fill: 'currentColor',
        d: 'm5.823 14.822-1.28.206a.824.824 0 0 1-.9-.52l-.463-1.212a.824.824 0 0 0-.476-.476l-1.212-.462a.824.824 0 0 1-.52-.9l.206-1.281a.824.824 0 0 0-.175-.65L.185 8.52a.824.824 0 0 1 0-1.04l.818-1.006a.824.824 0 0 0 .175-.65L.972 4.542a.824.824 0 0 1 .52-.9l1.212-.463a.824.824 0 0 0 .476-.476l.462-1.212a.824.824 0 0 1 .9-.52l1.281.206a.824.824 0 0 0 .65-.175L7.48.185a.824.824 0 0 1 1.04 0l1.006.818a.824.824 0 0 0 .65.175l1.281-.206a.824.824 0 0 1 .9.52l.463 1.212c.084.22.257.392.476.476l1.212.462c.365.14.582.515.52.9l-.206 1.281a.824.824 0 0 0 .175.65l.818 1.007a.824.824 0 0 1 0 1.04l-.818 1.006a.824.824 0 0 0-.175.65l.206 1.281a.824.824 0 0 1-.52.9l-1.212.463a.824.824 0 0 0-.476.476l-.462 1.212a.824.824 0 0 1-.9.52l-1.281-.206a.824.824 0 0 0-.65.175l-1.007.818a.824.824 0 0 1-1.04 0l-1.006-.818a.824.824 0 0 0-.65-.175z',
      }),
      svgElement('path', {
        fill: 'none',
        stroke: '#fff',
        'stroke-width': '1.6',
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        d: 'M4.755 8.252 7 10.5l4.495-4.495',
      }),
    ]
  );
}
