import { SVG_NS, svgElement } from '../../common/dom.js';

/**
 * The arrow pointing down, drawn as the preact ArrowIcon draws it; extra attributes follow the
 * size and shape ones.
 *
 * @param {number} [size]
 * @param {Record<string, string | undefined>} [attributes]
 */
export function arrowIcon(size = 14, attributes = {}) {
  return svgElement(
    'svg',
    { width: String(size), height: String(size), viewBox: '0 0 28 28', fill: 'none', xmlns: SVG_NS, ...attributes },
    [
      svgElement('path', {
        d: 'M6 11.5L14.5 19L22 11',
        stroke: 'currentColor',
        'stroke-width': '4',
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
      }),
    ]
  );
}
