import { SVG_NS, svgElement } from '../../common/dom.js';

/**
 * The cross, drawn as the preact CrossIcon draws it; extra attributes follow the size and shape
 * ones.
 *
 * @param {number | string} [size]
 * @param {Record<string, string | undefined>} [attributes]
 */
export function crossIcon(size = 14, attributes = {}) {
  return svgElement(
    'svg',
    { width: String(size), height: String(size), viewBox: '0 0 14 14', fill: 'none', xmlns: SVG_NS, ...attributes },
    [
      svgElement('path', {
        d: 'M2 2L12 12M12 2L2 12',
        stroke: 'currentColor',
        'stroke-width': '2',
        'stroke-linecap': 'round',
        'stroke-linejoin': 'round',
      }),
    ]
  );
}
