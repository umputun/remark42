import { copy } from './copy';

describe('copy to clipboard', () => {
  it('should call `execCommand` for old browser', async () => {
    document.execCommand = jest.fn(() => true);

    await copy('text');
    expect(document.execCommand).toHaveBeenCalledTimes(1);
  });

  // runs before the test that defines `navigator.clipboard`, which stays defined afterwards
  it('copies markup as text for old browser', async () => {
    const content = 'name\n<img src="x" onerror="alert(1)"> &lt;b&gt;';
    let copied: { text: string | null; elements: number; whiteSpace: string } | undefined;

    document.execCommand = jest.fn(() => {
      const container = document.body.lastElementChild as HTMLElement;

      copied = {
        text: container.textContent,
        elements: container.querySelectorAll('*').length,
        whiteSpace: container.style.whiteSpace,
      };
      return true;
    });

    await copy(content);
    expect(copied).toEqual({ text: content, elements: 0, whiteSpace: 'pre' });
  });

  it('should call `clipboard.write` for new browser', async () => {
    const clipboardWrite = jest.fn(() => Promise.resolve());
    Object.defineProperty(window, 'ClipboardItem', { value: jest.fn(), writable: true });

    Object.defineProperty(navigator, 'clipboard', {
      value: {
        write: clipboardWrite,
      },
    });

    await copy('text');
    expect(clipboardWrite).toHaveBeenCalledTimes(1);
  });
});
