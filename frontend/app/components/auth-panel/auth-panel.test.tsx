import { h } from 'preact';

import type { User } from 'common/types';
import { render } from 'tests/utils';

import { AuthPanel, Props } from './auth-panel';
import type { StoreState } from 'store';

const defaultProps = {
  postInfo: {
    read_only: false,
    url: 'https://example.com',
    count: 3,
  },
  hiddenUsers: {},
} as Props;

function getProps(props: Partial<Props>): Props {
  return {
    ...defaultProps,
    ...props,
  } as Props;
}

function renderAuthPanel(props: Props) {
  const initialStore = ({
    user: null,
    theme: 'light',
    comments: {
      sort: '-score',
    },
    provider: { name: 'google' },
  } as unknown) as StoreState;

  return render(<AuthPanel {...props} />, initialStore);
}

describe('<AuthPanel />', () => {
  describe('For not authorized : null', () => {
    it('should not render settings if there is no hidden users', () => {
      const props = getProps({
        user: null,
        postInfo: { ...defaultProps.postInfo, read_only: true },
      });
      const { container } = renderAuthPanel(props);

      expect(container.querySelector('.auth-panel__admin-action')).not.toBeInTheDocument();
    });

    it('should render settings if there is some hidden users', () => {
      const props = getProps({
        user: null,
        postInfo: { ...defaultProps.postInfo, read_only: true },
        hiddenUsers: { hidden_joe: {} as User },
      });
      const { container } = renderAuthPanel(props);

      expect(container.querySelector('.auth-panel__admin-action')).toHaveTextContent('Show settings');
    });
  });

  describe('For authorized user', () => {
    it('should render info about current user', () => {
      const props = getProps({
        user: { id: 'john', name: 'John', picture: '', ip: '', admin: false, block: false, verified: true },
      });
      const { container } = renderAuthPanel(props);

      expect(container.querySelectorAll('.auth-panel__column')).toHaveLength(2);
      expect(container.querySelector('.auth-panel__column')?.textContent).toContain('You logged in as John');
    });
  });
  describe('For admin user', () => {
    it('should render admin action', () => {
      const props = getProps({
        user: { id: 'test', admin: true, name: 'John', block: false, verified: true, ip: '', picture: '' },
      });
      const { container } = renderAuthPanel(props);

      expect(container.querySelector('.auth-panel__admin-action')).toHaveTextContent('Show settings');
    });
  });
});
