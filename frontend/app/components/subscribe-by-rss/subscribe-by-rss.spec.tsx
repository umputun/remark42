import { screen } from '@testing-library/preact';
import { User } from 'common/types';
import { h } from 'preact';
import { render } from 'tests/utils';

import { SubscribeByRSS, createSubscribeUrl } from './subscribe-by-rss';

jest.mock('react-redux', () => ({
  useSelector: jest.fn((fn) => fn({ theme: 'light' })),
}));

describe('<SubscribeByRSS/>', () => {
  describe('for unauthorized', () => {
    it('should render links in dropdown', () => {
      render(<SubscribeByRSS />);

      expect(screen.getByText('Thread')).toBeInTheDocument();
      expect(screen.getByText('Site')).toBeInTheDocument();
      expect(screen.getByText('Replies')).not.toBeInTheDocument();
    });
  });

  describe('for authorized', () => {
    const user = { id: 'user-1' } as User;
    it('should render links in dropdown', () => {
      render(<SubscribeByRSS />, { user });

      expect(screen.getByText('Thread')).toBeInTheDocument();
      expect(screen.getByText('Site')).toBeInTheDocument();
      expect(screen.getByText('Replies')).toBeInTheDocument();
    });

    it('should have userId in replies link', () => {
      render(<SubscribeByRSS />, { user });

      expect(screen.getByText('Replies').getAttribute('href')).toBe(createSubscribeUrl('reply', '&user=user-1'));
    });
  });
});
