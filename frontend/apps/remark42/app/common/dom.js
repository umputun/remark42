// Small helpers for the widget's components written in plain JavaScript.

export const SVG_NS = 'http://www.w3.org/2000/svg';

/**
 * The class names that are set, joined with spaces, in the order given.
 *
 * @param {Array<string | false | null | undefined>} names
 */
export function classes(...names) {
  return names.filter(Boolean).join(' ');
}

/**
 * An SVG element with the attributes set in the order given, holding the children. An attribute
 * whose value is undefined is left out, as preact leaves it out: a class a CSS module does not
 * define comes through as undefined in the real build.
 *
 * @param {string} name
 * @param {Record<string, string | undefined>} attributes
 * @param {SVGElement[]} [children]
 * @returns {SVGElement}
 */
export function svgElement(name, attributes, children = []) {
  const element = /** @type {SVGElement} */ (document.createElementNS(SVG_NS, name));
  for (const [attribute, value] of Object.entries(attributes)) {
    if (value !== undefined) {
      element.setAttribute(attribute, value);
    }
  }
  element.append(...children);
  return element;
}

/**
 * Makes the children of parent exactly these nodes, in this order. Nodes that leave are removed
 * first and the rest are moved only when out of place, so a node that stays where it was is never
 * detached and keeps its focus.
 *
 * @param {Element} parent
 * @param {Node[]} children
 */
export function placeChildren(parent, children) {
  const wanted = new Set(children);
  for (const node of Array.from(parent.childNodes)) {
    if (!wanted.has(node)) {
      node.remove();
    }
  }
  children.forEach((child, i) => {
    if (parent.childNodes[i] !== child) {
      parent.insertBefore(child, parent.childNodes[i] ?? null);
    }
  });
}
