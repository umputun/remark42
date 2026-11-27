import { SVG_NS, svgElement } from '../../common/dom.js';

/**
 * The sign-out arrow, drawn as the preact SignOutIcon draws it; extra attributes follow the size
 * and shape ones.
 *
 * @param {number | string} [size]
 * @param {Record<string, string | undefined>} [attributes]
 */
export function signOutIcon(size = 16, attributes = {}) {
  return svgElement(
    'svg',
    { width: String(size), height: String(size), viewBox: '0 0 16 16', fill: 'none', xmlns: SVG_NS, ...attributes },
    [
      svgElement('path', {
        stroke: 'currentColor',
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
        'stroke-width': '1.5',
        d: 'M5.7 15H2.6A1.6 1.6 0 011 13.4V2.6A1.6 1.6 0 012.6 1h3M11.1 11.9L15 8l-3.9-3.9M15 8H5.7',
      }),
    ]
  );
}
