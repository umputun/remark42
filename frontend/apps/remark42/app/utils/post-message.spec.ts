import { updateIframeHeight } from './post-message';

describe('updateIframeHeight', () => {
  const postMessage = jest.fn();
  let originalParent: Window;

  beforeAll(() => {
    originalParent = window.parent;
    // postMessageToParent bails out when window.parent is window, which is the case in jsdom
    Object.defineProperty(window, 'parent', { value: { postMessage }, writable: true, configurable: true });
  });

  afterAll(() => {
    Object.defineProperty(window, 'parent', { value: originalParent, writable: true, configurable: true });
  });

  beforeEach(() => {
    postMessage.mockClear();
    jest.spyOn(document.body, 'offsetHeight', 'get').mockReturnValue(500);
  });

  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('should report the document height without adding to it', () => {
    updateIframeHeight();

    expect(postMessage).toHaveBeenCalledWith({ height: 500 }, '*');
  });

  it('should report the dropdown height when it exceeds the document height', () => {
    const dropdown = document.createElement('div');

    jest.spyOn(dropdown, 'getBoundingClientRect').mockReturnValue({ top: 100 } as DOMRect);
    jest.spyOn(dropdown, 'scrollHeight', 'get').mockReturnValue(600);

    updateIframeHeight(dropdown);

    // 20px allowance for the shadow under the dropdown
    expect(postMessage).toHaveBeenCalledWith({ height: 720 }, '*');
  });

  it('should report the document height when it exceeds the dropdown height', () => {
    const dropdown = document.createElement('div');

    jest.spyOn(dropdown, 'getBoundingClientRect').mockReturnValue({ top: 10 } as DOMRect);
    jest.spyOn(dropdown, 'scrollHeight', 'get').mockReturnValue(50);

    updateIframeHeight(dropdown);

    expect(postMessage).toHaveBeenCalledWith({ height: 500 }, '*');
  });
});
