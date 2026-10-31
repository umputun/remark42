import '@testing-library/jest-dom';
import { fireEvent, waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';

import { displayWebsite, ProfileWebsite, safeWebsite } from './website';

// the unsafe schemes are test data that must be rejected, never followed
const SCRIPT_URL = ['javascript', 'alert(1)'].join(':');

describe('<ProfileWebsite />', () => {
  afterEach(() => jest.restoreAllMocks());

  it('shows another user website as an external nofollow link', () => {
    const getMyWebsite = jest.spyOn(api, 'getMyWebsite');
    const { getByRole, queryByRole } = render(
      <ProfileWebsite website="https://www.example.com/blog/" isCurrent={false} />
    );

    const link = getByRole('link', { name: 'example.com/blog' });
    expect(link).toHaveAttribute('href', 'https://www.example.com/blog/');
    expect(link).toHaveAttribute('rel', 'nofollow ugc noopener noreferrer');
    expect(link).toHaveAttribute('target', '_blank');
    expect(queryByRole('button')).not.toBeInTheDocument();
    expect(getMyWebsite).not.toHaveBeenCalled();
  });

  it('renders nothing for a missing or unsafe website', () => {
    const { container, rerender } = render(<ProfileWebsite isCurrent={false} />);
    expect(container).toBeEmptyDOMElement();
    rerender(<ProfileWebsite website={SCRIPT_URL} isCurrent={false} />);
    expect(container).toBeEmptyDOMElement();
    rerender(<ProfileWebsite website="http://0x7f.0.0.1" isCurrent={false} />);
    expect(container).toBeEmptyDOMElement();
  });

  it('offers no editing when the server disables the feature', async () => {
    const getMyWebsite = jest.spyOn(api, 'getMyWebsite').mockRejectedValue(new Error('forbidden'));
    const { container } = render(<ProfileWebsite isCurrent />);
    await waitFor(() => expect(getMyWebsite).toHaveBeenCalled());
    expect(container).toBeEmptyDOMElement();
  });

  it('lets the current user add, fix and remove the website', async () => {
    jest.spyOn(api, 'getMyWebsite').mockResolvedValue({ website: '' });
    const setMyWebsite = jest
      .spyOn(api, 'setMyWebsite')
      .mockRejectedValueOnce(new Error('invalid website'))
      .mockResolvedValueOnce({ website: 'https://me.example.com' })
      .mockResolvedValueOnce({ website: '' });
    const { findByRole, getByRole, getByLabelText, queryByRole, findByText } = render(<ProfileWebsite isCurrent />);

    fireEvent.click(await findByRole('button', { name: 'Add your website' }));
    const input = getByLabelText('Website');

    fireEvent.input(input, { target: { value: 'ftp://nope' } });
    fireEvent.click(getByRole('button', { name: 'Save' }));
    expect(await findByText('Enter a valid web address starting with http or https')).toBeInTheDocument();

    fireEvent.input(input, { target: { value: '  me.example.com ' } });
    fireEvent.click(getByRole('button', { name: 'Save' }));
    await waitFor(() => expect(setMyWebsite).toHaveBeenLastCalledWith('me.example.com'));
    expect(await findByRole('link', { name: 'me.example.com' })).toHaveAttribute('href', 'https://me.example.com/');

    fireEvent.click(getByRole('button', { name: 'Edit website' }));
    fireEvent.click(getByRole('button', { name: 'Remove website' }));
    await waitFor(() => expect(setMyWebsite).toHaveBeenLastCalledWith(''));
    await waitFor(() => expect(queryByRole('link')).not.toBeInTheDocument());
    expect(await findByRole('button', { name: 'Add your website' })).toBeInTheDocument();
  });
});

describe('website helpers', () => {
  it('accepts only http(s) URLs', () => {
    expect(safeWebsite('https://example.com')).toBe('https://example.com/');
    expect(safeWebsite('http://example.com/a')).toBe('http://example.com/a');
    expect(safeWebsite(SCRIPT_URL)).toBeNull();
    expect(safeWebsite('data:text/html,hi')).toBeNull();
    expect(safeWebsite('not a url')).toBeNull();
    expect(safeWebsite(undefined)).toBeNull();
  });

  it('rejects IP addresses in any form, localhost and credentials', () => {
    [
      'http://127.1',
      'http://0x7f.0.0.1',
      'http://0177.0.0.1',
      'http://2130706433',
      'http://10.1',
      'http://[::1]/',
    ].forEach((value) => expect(safeWebsite(value)).toBeNull());
    expect(safeWebsite('http://localhost:8080')).toBeNull();
    expect(safeWebsite('https://app.localhost')).toBeNull();
    expect(safeWebsite('https://user:pass@example.com')).toBeNull();
    expect(safeWebsite('https://1.example.com')).toBe('https://1.example.com/');
  });

  it('shortens the display text', () => {
    expect(displayWebsite('https://www.example.com/')).toBe('example.com');
    expect(displayWebsite('https://sub.example.com/a/b/?x=1')).toBe('sub.example.com/a/b/?x=1');
  });
});
