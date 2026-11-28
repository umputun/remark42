import { verificationIcon } from './verification-icon.js';

describe('the verification icon', () => {
  it('is drawn at the default size', () => {
    const icon = verificationIcon(undefined, { title: 'icon' });
    expect(icon.getAttribute('width')).toBe('12');
    expect(icon.getAttribute('height')).toBe('12');
    expect(icon.getAttribute('title')).toBe('icon');
  });

  it('is drawn at the size given', () => {
    const icon = verificationIcon(16, { title: 'icon' });
    expect(icon.getAttribute('width')).toBe('16');
    expect(icon.getAttribute('height')).toBe('16');
  });
});
