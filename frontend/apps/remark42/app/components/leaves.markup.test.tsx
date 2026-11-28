import { h } from 'preact';
import { render } from '@testing-library/preact';

import { IntlProvider } from 'common/intl';
import en from 'locales/en.json';
import { Avatar } from 'components/avatar';
import { IconButton } from 'components/icon-button/icon-button';
import { CrossIcon } from 'components/icons/cross';
import { SignOutIcon } from 'components/icons/signout';
import { VerificationIcon } from 'components/icons/verification';
import { Spinner } from 'components/spinner/spinner';

function drawn(node: preact.ComponentChild) {
  return render(
    <IntlProvider locale="en" messages={en}>
      {node}
    </IntlProvider>
  ).container.innerHTML;
}

// the markup of the preact leaves whose last users were the comment and the profile, recorded
// before they were removed; innerHTML keeps the attributes in the order they were written
describe('the leaves of the comment and the profile draw the markup of the preact ones', () => {
  it.each<[string, () => string]>([
    ['the cross', () => drawn(<CrossIcon />)],
    ['the cross with a size and a class', () => drawn(<CrossIcon size={20} class="x" />)],
    ['the sign-out icon', () => drawn(<SignOutIcon />)],
    ['the sign-out icon at 14', () => drawn(<SignOutIcon size="14" />)],
    ['the spinner', () => drawn(<Spinner />)],
    ['the gray spinner', () => drawn(<Spinner color="gray" />)],
    ['an icon button', () => drawn(<IconButton title="Go" />)],
    ['an icon button with a class', () => drawn(<IconButton className="extra" />)],
    ['the avatar: a picture', () => drawn(<Avatar url="https://example.com/a.png" size={20} />)],
    ['the avatar: the ghost', () => drawn(<Avatar size={24} />)],
    ['the avatar: an identicon', () => drawn(<Avatar url="https://example.com/a.png" size={20} identicon />)],
    ['the verification icon', () => drawn(<VerificationIcon />)],
    ['the verification icon at 16 with a title', () => drawn(<VerificationIcon size={16} title="icon" />)],
    [
      'the verification icon with a title and a class',
      () => drawn(<VerificationIcon title="Verified user" className="verificationIcon" />),
    ],
  ])('%s', (_, draw) => {
    expect(draw()).toMatchSnapshot();
  });
});
