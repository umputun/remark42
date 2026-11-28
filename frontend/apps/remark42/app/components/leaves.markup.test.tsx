import { createIntl } from 'common/intl';
import en from 'locales/en.json';
import { avatarElement } from 'components/avatar/avatar-element.js';
import { iconButtonElement } from 'components/icon-button/icon-button-element.js';
import { crossIcon } from 'components/icons/cross-icon.js';
import { signOutIcon } from 'components/icons/signout-icon.js';
import { verificationIcon } from 'components/icons/verification-icon.js';
import { spinnerElement } from 'components/spinner/spinner-element.js';

const intl = createIntl('en', en);

function titled(button: HTMLButtonElement, title: string) {
  button.title = title;
  return button;
}

// the markup of the preact leaves whose last users were the comment and the profile, recorded
// before they were removed; innerHTML keeps the attributes in the order they were written
describe('the leaves of the comment and the profile draw the markup of the preact ones', () => {
  it.each<[string, () => string]>([
    ['the cross', () => crossIcon().outerHTML],
    ['the cross with a size and a class', () => crossIcon(20, { class: 'x' }).outerHTML],
    ['the sign-out icon', () => signOutIcon().outerHTML],
    ['the sign-out icon at 14', () => signOutIcon('14').outerHTML],
    ['the spinner', () => spinnerElement(intl).outerHTML],
    ['the gray spinner', () => spinnerElement(intl, 'gray').outerHTML],
    ['an icon button', () => titled(iconButtonElement(), 'Go').outerHTML],
    ['an icon button with a class', () => iconButtonElement('extra').outerHTML],
    ['the avatar: a picture', () => avatarElement({ url: 'https://example.com/a.png', size: 20 }).img.outerHTML],
    ['the avatar: the ghost', () => avatarElement({ size: 24 }).img.outerHTML],
    [
      'the avatar: an identicon',
      () => avatarElement({ url: 'https://example.com/a.png', size: 20, identicon: true }).img.outerHTML,
    ],
    ['the verification icon', () => verificationIcon().outerHTML],
    ['the verification icon at 16 with a title', () => verificationIcon(16, { title: 'icon' }).outerHTML],
    [
      'the verification icon with a title and a class',
      () => verificationIcon(12, { title: 'Verified user', class: 'verificationIcon' }).outerHTML,
    ],
  ])('%s', (_, draw) => {
    expect(draw()).toMatchSnapshot();
  });
});
