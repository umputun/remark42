// Building elements for the components written in plain JavaScript, attribute by attribute in the
// order preact writes them, so the markup comes out as the preact components drew it.

/**
 * @typedef {string | number | boolean | null | undefined} AttributeValue
 */

/**
 * Sets the attributes in the order given. `class` sets the class name, `value` and `checked` set
 * the properties, as preact does (a text field then shows the value without carrying it as an
 * attribute). true writes an empty attribute, and false, null and undefined leave it out.
 *
 * @param {Element} element
 * @param {Record<string, AttributeValue>} attributes
 */
export function setAttributes(element, attributes) {
  for (const [name, value] of Object.entries(attributes)) {
    if (name === 'value' || name === 'checked') {
      /** @type {any} */ (element)[name] = value;
    } else if (value === undefined || value === null || value === false) {
      element.removeAttribute(name === 'className' ? 'class' : name);
    } else if (name === 'class' || name === 'className') {
      element.setAttribute('class', String(value));
    } else {
      element.setAttribute(name, value === true ? '' : String(value));
    }
  }
}

/**
 * An HTML element with these attributes, holding these children; a string child is a text node.
 *
 * @template {keyof HTMLElementTagNameMap} K
 * @param {K} tag
 * @param {Record<string, AttributeValue>} [attributes]
 * @param {Array<Node | string>} [children]
 * @returns {HTMLElementTagNameMap[K]}
 */
export function element(tag, attributes = {}, children = []) {
  const node = document.createElement(tag);
  setAttributes(node, attributes);
  node.append(...children);
  return node;
}
