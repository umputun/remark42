import { fireEvent, screen } from '@testing-library/preact';

import { render } from 'tests/utils';

import { SubscribeByRSS, createSubscribeUrl } from '.';

import styles from './subscribe-by-rss.module.css';

/** Renders the widget and opens the dropdown, since the links only exist while it is open. */
function renderOpened() {
  const result = render(<SubscribeByRSS userId="user-1" />, { theme: 'light' });

  // the toggle is the only button until the dropdown opens, so this does not depend
  // on the button's title copy
  fireEvent.click(screen.getByRole('button'));

  return result;
}

describe('<SubscribeByRSS/>', () => {
  it('should be render links in dropdown', () => {
    const { container } = renderOpened();

    expect(container.querySelectorAll(`.${styles.link}`)).toHaveLength(3);
  });

  it('should have userId in replies link', () => {
    const { container } = renderOpened();
    const links = container.querySelectorAll(`.${styles.link}`);

    expect(links[2].getAttribute('href')).toBe(createSubscribeUrl('reply', '&user=user-1'));
  });
});
