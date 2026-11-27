// A list of DOM nodes kept in step with a list of items, for the widget's plain JavaScript
// components.

/**
 * Keeps the children of parent in step with a list of items, one node per key. A node is built
 * once for its key and kept while the key stays; the nodes whose keys leave are removed; and the
 * fewest nodes are moved to bring the rest into order. A node already in its place relative to
 * the others is never detached, so it keeps its focus, selection and any transition.
 *
 * @template T
 * @param {Element} parent  holds this list's nodes and nothing else
 * @param {(item: T) => string} keyOf
 * @param {(item: T) => Element} create  builds the node for an item seen for the first time
 * @param {(node: Element, item: T) => void} [refresh]  brings a kept node up to date with its item
 */
export function keyedList(parent, keyOf, create, refresh) {
  /** @type {Map<string, Element>} */
  let nodes = new Map();

  return {
    /** @param {T[]} items  each with its own key */
    update(items) {
      /** @type {Map<string, Element>} */
      const next = new Map();
      for (const item of items) {
        const key = keyOf(item);
        if (next.has(key)) {
          throw new Error(`keyedList: the key ${key} appears twice`);
        }
        let node = nodes.get(key);
        if (node) {
          refresh?.(node, item);
        } else {
          node = create(item);
        }
        next.set(key, node);
      }
      for (const [key, node] of nodes) {
        if (!next.has(key)) {
          node.remove();
        }
      }
      nodes = next;
      arrange(parent, Array.from(next.values()));
    },
    /** @param {string} key */
    node(key) {
      return nodes.get(key);
    },
  };
}

/**
 * Puts the nodes into parent in this order, moving only those outside the longest run that is
 * already in order.
 *
 * @param {Element} parent
 * @param {Element[]} order
 */
function arrange(parent, order) {
  /** @type {Map<Node, number>} */
  const position = new Map();
  Array.from(parent.childNodes).forEach((node, i) => position.set(node, i));
  const stay = longestIncreasing(order.map((node) => position.get(node) ?? -1));

  /** @type {Node | null} */
  let after = null;
  for (let i = order.length - 1; i >= 0; i--) {
    if (!stay.has(i)) {
      parent.insertBefore(order[i], after);
    }
    after = order[i];
  }
}

/**
 * The indexes of the longest strictly increasing run of values, skipping negative ones.
 *
 * @param {number[]} values
 * @returns {Set<number>}
 */
function longestIncreasing(values) {
  /** @type {number[]} index of the smallest tail of a run of each length */
  const tails = [];
  /** @type {number[]} */
  const previous = new Array(values.length).fill(-1);

  values.forEach((value, i) => {
    if (value < 0) {
      return;
    }
    let low = 0;
    let high = tails.length;
    while (low < high) {
      const middle = (low + high) >> 1;
      if (values[tails[middle]] < value) {
        low = middle + 1;
      } else {
        high = middle;
      }
    }
    previous[i] = low > 0 ? tails[low - 1] : -1;
    tails[low] = i;
  });

  /** @type {Set<number>} */
  const run = new Set();
  for (let i = tails.length > 0 ? tails[tails.length - 1] : -1; i >= 0; i = previous[i]) {
    run.add(i);
  }
  return run;
}
