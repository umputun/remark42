import { keyedList } from './list.js';

type Item = { id: string; label: string };

function list() {
  const parent = document.createElement('ul');
  const created: string[] = [];
  const rows = keyedList<Item>(
    parent,
    (item) => item.id,
    (item) => {
      created.push(item.id);
      const li = document.createElement('li');
      li.tabIndex = 0;
      li.textContent = item.label;
      return li;
    },
    (node, item) => {
      node.textContent = item.label;
    }
  );
  const set = (ids: string, label = '') => rows.update(ids.split('').map((id) => ({ id, label: id + label })));
  const texts = () =>
    Array.from(parent.children)
      .map((li) => li.textContent)
      .join(' ');
  return { parent, rows, created, set, texts };
}

// the nodes removed from or put into the parent, counted by watching its child list
function moves(parent: Element, change: () => void) {
  const observer = new MutationObserver(() => {});
  observer.observe(parent, { childList: true });
  change();
  const records = observer.takeRecords();
  observer.disconnect();
  return records.reduce((n, r) => n + r.removedNodes.length, 0);
}

describe('keyedList', () => {
  it('builds a node per item, in order', () => {
    const { set, texts, created } = list();
    set('abc');

    expect(texts()).toBe('a b c');
    expect(created).toEqual(['a', 'b', 'c']);
  });

  it('keeps the node of every key that stays, and brings it up to date', () => {
    const { parent, rows, set, texts, created } = list();
    set('abc');
    const before = ['a', 'b', 'c'].map((key) => rows.node(key));

    set('abc', '!');

    expect(texts()).toBe('a! b! c!');
    ['a', 'b', 'c'].forEach((key, i) => expect(rows.node(key)).toBe(before[i]));
    Array.from(parent.children).forEach((node, i) => expect(node).toBe(before[i]));
    expect(created).toHaveLength(3);
  });

  it.each([
    ['removing one', 'abcd', 'abd', 0],
    ['inserting one', 'abd', 'abcd', 0],
    ['moving the last to the front', 'abcd', 'dabc', 1],
    ['moving the first to the end', 'abcd', 'bcda', 1],
    ['swapping two neighbours', 'abcd', 'acbd', 1],
    ['reversing', 'abcd', 'dcba', 3],
    ['removing, inserting and moving at once', 'abcde', 'eaxbd', 1],
  ])('%s leaves the order right and moves the fewest nodes', (_, from, to, moved) => {
    const { parent, rows, set, texts } = list();
    set(from);
    const kept = to.split('').filter((key) => from.includes(key));
    const nodes = kept.map((key) => rows.node(key));

    const removed = moves(parent, () => set(to));

    expect(texts()).toBe(to.split('').join(' '));
    kept.forEach((key, i) => expect(rows.node(key)).toBe(nodes[i]));
    expect(removed - (from.length - kept.length)).toBe(moved);
  });

  it('keeps the focus on a row that stays while others move round it', () => {
    const { parent, rows, set } = list();
    document.body.append(parent);
    set('abcd');
    (rows.node('b') as HTMLElement).focus();

    set('cbda');

    expect(document.activeElement).toBe(rows.node('b'));
    parent.remove();
  });

  it('empties the parent when the items run out', () => {
    const { parent, set } = list();
    set('abc');
    set('');

    expect(parent.childNodes).toHaveLength(0);
  });

  it('refuses a key that appears twice', () => {
    const { set } = list();

    expect(() => set('aba')).toThrow('the key a appears twice');
  });
});
