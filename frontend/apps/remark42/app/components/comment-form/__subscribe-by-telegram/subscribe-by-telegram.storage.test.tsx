import { h } from 'preact';

import { user } from '__stubs__/user';
import { render } from 'tests/utils';

import { SubscribeByTelegram } from '.';

afterEach(() => {
  jest.restoreAllMocks();
});

// a frame whose storage the browser blocks throws on reading sessionStorage at all
it('draws the closed Telegram button where the browser blocks storage in the frame', () => {
  jest.spyOn(window, 'sessionStorage', 'get').mockImplementation(() => {
    throw new DOMException('The operation is insecure.', 'SecurityError');
  });

  const view = render(<SubscribeByTelegram />, { user, theme: 'light' });
  view.rerender(<SubscribeByTelegram />);

  expect(view.getByTitle('Subscribe by Telegram')).toBeTruthy();
});
