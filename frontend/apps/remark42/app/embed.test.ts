process.env.REMARK_NODE = 'remark42';

const MARKER = 'data-remark42-iframe';
const MARKED = `iframe[${MARKER}]`;

async function mount(placeholder = '') {
  document.body.innerHTML = `<div id="remark42">${placeholder}</div>`;
  jest.resetModules();
  await import('./embed');

  return document.getElementById('remark42')!;
}

describe('embed', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
    document.body.innerHTML = '';
  });

  it('creates its iframe rather than adopting an element placeholder', async () => {
    const root = await mount('<noscript>Please enable JavaScript to view the comments.</noscript>');

    expect(root.querySelectorAll(MARKED)).toHaveLength(1);
  });

  it('creates its iframe rather than adopting an unmarked one', async () => {
    const root = await mount('<iframe title="placeholder"></iframe>');

    expect(root.querySelectorAll('iframe')).toHaveLength(2);
    expect(root.querySelectorAll(MARKED)).toHaveLength(1);
  });

  it('creates its iframe rather than adopting a marked one further down the tree', async () => {
    const root = await mount(`<div><iframe ${MARKER}></iframe></div>`);

    expect(root.querySelectorAll('iframe')).toHaveLength(2);
    expect(root.querySelectorAll(`:scope > ${MARKED}`)).toHaveLength(1);
  });

  it('reuses its own iframe on a second createInstance', async () => {
    const root = await mount();
    const first = root.querySelector(MARKED);

    window.REMARK42.createInstance(window.remark_config);

    expect(root.querySelectorAll('iframe')).toHaveLength(1);
    expect(root.querySelector(MARKED)).toBe(first);
  });
});
