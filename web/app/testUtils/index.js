export function createDomContainer(setup) {
  let domContainer = null;
  beforeAll(() => {
    domContainer = document.createElement('div');
    (document.body || document.documentElement).appendChild(domContainer);
    setup({ domContainer });
  });

  beforeEach(() => {
    domContainer.innerHTML = '';
  });

  afterAll(() => {
    domContainer.parentNode.removeChild(domContainer);
    domContainer = null;
  });
}
