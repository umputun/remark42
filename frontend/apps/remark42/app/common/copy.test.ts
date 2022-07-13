import { copy } from './copy';

describe('copy to clipboard', () => {
  it('should call `execCommand` for old browser', async () => {
    document.execCommand = jest.fn(() => true);

    await copy('text');
    expect(document.execCommand).toHaveBeenCalledTimes(1);
  });

  it('should call `clipboard.write` for new browser', async () => {
    const clipboardWrite = jest.fn(() => Promise.resolve());
    window.ClipboardItem = jest.fn();

    Object.defineProperty(navigator, 'clipboard', {
      value: {
        write: clipboardWrite,
      },
    });

    await copy('text');
    expect(clipboardWrite).toHaveBeenCalledTimes(1);
  });
});
