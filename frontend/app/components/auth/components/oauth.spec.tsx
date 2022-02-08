import { h } from 'preact';
import { render } from 'tests/utils';
import { BASE_URL } from 'common/constants.config';

import { OAuth } from './oauth';

describe('<OAuth />', () => {
  it('should have permanent class name', () => {
    const { container } = render(<OAuth providers={['google']} />);

    expect(container.querySelector('ul')?.getAttribute('class')).toContain('oauth');
    expect(container.querySelector('li')?.getAttribute('class')).toContain('oauth-item');
    expect(container.querySelector('a')?.getAttribute('class')).toContain('oauth-button');
    expect(container.querySelector('img')?.getAttribute('class')).toContain('oauth-icon');
  });

  it('should have rigth `href`', () => {
    const { container } = render(<OAuth providers={['google']} />);

    expect(container.querySelector('a')?.getAttribute('href')).toBe(
      `${BASE_URL}/auth/google/login?from=http%3A%2F%2Flocalhost%2F%3FselfClose&site=remark`
    );
  });
});
