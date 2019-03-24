export function createDomContainer(setup: (domContainer: HTMLElement) => void): void {
  let domContainer: HTMLElement | null = null;
  beforeAll(() => {
    domContainer = document.createElement('div');
    (document.body || document.documentElement).appendChild(domContainer);
    setup(domContainer);
  });

  beforeEach(() => {
    domContainer!.innerHTML = '';
  });

  afterAll(() => {
    domContainer!.parentNode!.removeChild(domContainer!);
    domContainer = null;
  });
}
