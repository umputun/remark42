import { openProfile, closeProfile } from './profile';

describe('openProfile', () => {
  afterEach(() => closeProfile());

  it('opens the profile as a dialog with a title of its own', () => {
    openProfile({ id: 'user', name: 'user name', picture: '' });

    const dialog = document.querySelector('[role="dialog"]');
    expect(dialog).not.toBeNull();
    expect(dialog?.getAttribute('aria-label')).toBe('Profile | Remark42');
    expect(dialog?.querySelector('iframe')?.title).toBe('Profile | Remark42');
  });
});
