import '@testing-library/jest-dom';
import { waitFor, fireEvent, screen } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as pq from 'utils/parse-query';
import * as postMessage from 'utils/post-message';
import * as reader from 'utils/reader';
import type { Comment, User } from 'common/types';

import { Profile } from './profile';

const userParamsStub = {
  id: '1',
  name: 'username',
  picture: '/avatar.png',
};

const userStub: User = {
  ...userParamsStub,
  ip: '',
  picture: '/avatar.png',
  admin: false,
  block: false,
  verified: false,
};

const commentStub: Comment = {
  id: '1',
  pid: '2',
  text: 'comment content',
  locator: {
    site: '',
    url: '',
  },
  score: 0,
  vote: 0,
  voted_ips: [],
  time: '2021-04-02T14:52:39.985281605-05:00',
  user: userStub,
};
const commentsStub = [commentStub, commentStub, commentStub];

/** Opens the profile of the reader: the widget frame answers that the reader is the one shown */
function openAsReader(params: Record<string, string>) {
  jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...params, token: 'token' }));
  jest
    .spyOn(reader, 'askReader')
    .mockImplementation(async () => ({ id: params.id, name: params.name, picture: params.picture }));
}

describe('<Profile />', () => {
  it('should render user without comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    const getUserComments = jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: [], count: 0 }));

    const { findByText, queryByLabelText, queryByRole, queryByTestId } = render(<Profile />);

    expect(getUserComments).toHaveBeenCalledWith('1', { limit: 10, skip: 0 });
    expect(await findByText('No comments yet.')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).not.toBeInTheDocument();
    expect(queryByLabelText('Loading...')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render user with comments without load more button', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => userParamsStub);
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: commentsStub, count: commentsStub.length }));

    const { findByText, queryByRole, queryByTestId } = render(<Profile />);

    expect(await findByText('Comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(commentsStub.length.toString());
    expect(queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render user with comments with load more button', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => userParamsStub);
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: new Array(10).fill(commentStub), count: 15 }));

    const { findByText, queryByRole, queryByTestId } = render(<Profile />);

    expect(await findByText('Comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent('15');
    expect(queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).toBeInTheDocument();
  });

  it('should render current user without comments', async () => {
    openAsReader(userParamsStub);
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: [], count: 0 }));

    const { getByText, findByTitle, findByText, queryByTestId, queryByRole } = render(<Profile />);

    expect(await findByTitle('Sign out')).toBeInTheDocument();
    expect(getByText('Request deletion of my data')).toBeInTheDocument();
    expect(await findByText('No comments yet.')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render current user with comments without load more button', async () => {
    openAsReader(userParamsStub);
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: commentsStub, count: commentsStub.length }));

    const { findByText, queryByTitle, queryByText, queryByTestId, queryByRole } = render(<Profile />);

    expect(await findByText('Your comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(commentsStub.length.toString());
    expect(queryByTitle('Sign out')).toBeInTheDocument();
    expect(queryByText('Request deletion of my data')).toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render current user with comments with load more button', async () => {
    const comments = Array(15).fill(commentStub);
    openAsReader(userParamsStub);
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments, count: comments.length }));

    const { findByText, queryByTitle, queryByText, queryByTestId, queryByRole } = render(<Profile />);

    expect(await findByText('Your comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(comments.length.toString());
    expect(queryByTitle('Sign out')).toBeInTheDocument();
    expect(queryByText('Request deletion of my data')).toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).toBeInTheDocument();
  });

  it('leaves the sign-out to the widget and sends no logout request of its own', async () => {
    openAsReader(userParamsStub);
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: [], count: 0 }));
    const posted = jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
    posted.mockClear();
    fetchMock.mockClear();

    const { getByTitle, findByTitle } = render(<Profile />);

    fireEvent.click(await findByTitle('Sign out'));

    // the page removes this frame on the message, and a request sent from it would be cancelled
    expect(posted).toHaveBeenCalledWith({ profile: null, signout: true });
    await waitFor(() => expect(getByTitle('Sign out')).toBeDisabled());
    expect(fetchMock.mock.calls.filter(([url]) => String(url).includes('/auth/logout'))).toEqual([]);
  });

  it('should render user without footer', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: commentsStub, count: commentsStub.length }));

    const { container } = render(<Profile />);

    expect(container.querySelector('profile-footer')).not.toBeInTheDocument();
  });

  it('load more button should disappear if there no more comments to fetch', async () => {
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: new Array(10).fill(commentStub), count: 15 }));

    const { findByRole, getByRole, queryByRole, queryByTestId } = render(<Profile />);

    expect(await findByRole('button', { name: /load more/i })).toBeInTheDocument();
    fireEvent.click(getByRole('button', { name: /load more/i }));
    expect(queryByTestId('spinner')).toBeInTheDocument();

    await waitFor(() => {
      expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
    });
  });

  it('preloader should not be rendered after click on load more button', async () => {
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: new Array(10).fill(commentStub), count: 15 }));

    const { findByRole, queryByTestId } = render(<Profile />);

    fireEvent.click(await findByRole('button', { name: /load more/i }));
    expect(queryByTestId('preloader')).not.toBeInTheDocument();
  });

  it('should not render removal button for anonymous user', async () => {
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: new Array(10).fill(commentStub), count: 15 }));
    // an anonymous user is recognised by the id prefix, and the button is only ever offered on
    // the reader's own profile, so both have to hold for the assertion to mean anything
    openAsReader({ ...userParamsStub, id: 'anonymous_1', name: 'anonymous_1' });

    render(<Profile />);

    expect(await screen.findByText('Your comments')).toBeInTheDocument();
    expect(screen.queryByText(/request deletion of my data/i)).not.toBeInTheDocument();
  });

  describe('edit avatar link', () => {
    beforeEach(() => {
      jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: [], count: 0 }));
    });

    it('offers the reader who signed in with email a link to the Gravatar avatar page', async () => {
      openAsReader({ ...userParamsStub, id: 'email_1' });

      render(<Profile />);

      const link = await screen.findByRole('link', { name: 'Edit avatar' });

      expect(link).toHaveAttribute('href', 'https://gravatar.com/profile/avatars');
      expect(link).toHaveAttribute('target', '_blank');
      expect(link).toHaveAttribute('rel', 'noopener noreferrer');
      expect(screen.getByText(/sign out and sign in again/i)).toBeInTheDocument();
    });

    it('does not offer it on the profile of another reader who signed in with email', async () => {
      jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, id: 'email_1' }));

      render(<Profile />);

      expect(await screen.findByText('No comments yet.')).toBeInTheDocument();
      expect(screen.queryByRole('link', { name: 'Edit avatar' })).not.toBeInTheDocument();
      expect(screen.queryByText(/gravatar/i)).not.toBeInTheDocument();
    });

    it.each(['github_1', 'anonymous_1', 'telegram_1'])(
      'does not offer it to the reader signed in as %s, whose avatar is not from Gravatar',
      async (id) => {
        openAsReader({ ...userParamsStub, id });

        render(<Profile />);

        expect(await screen.findByText('No comments yet.')).toBeInTheDocument();
        expect(screen.queryByRole('link', { name: 'Edit avatar' })).not.toBeInTheDocument();
        expect(screen.queryByText(/gravatar/i)).not.toBeInTheDocument();
      }
    );
  });

  describe("whether the profile is the reader's own", () => {
    beforeEach(() => {
      // the spies outlive each case, and a call made in an earlier one would satisfy these
      jest
        .spyOn(api, 'getUserComments')
        .mockClear()
        .mockImplementation(async () => ({ comments: [], count: 0 }));
      jest.spyOn(reader, 'askReader').mockClear();
    });

    it('shows the reader the widget answers for when the page names nobody', async () => {
      jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ token: 'token' }));
      const ask = jest.spyOn(reader, 'askReader').mockImplementation(async () => userParamsStub);

      const { container } = render(<Profile />);

      expect(await screen.findByTitle('Sign out')).toBeInTheDocument();
      expect(ask).toHaveBeenCalledWith('token');
      // the sidebar slides in only with this class, and its close button is on it
      await waitFor(() => expect(container.querySelector('.profile')).toHaveClass('rootAppear'));
      expect(screen.getByText('username')).toBeInTheDocument();
      await waitFor(() => expect(api.getUserComments).toHaveBeenCalledWith('1', { limit: 10, skip: 0 }));
    });

    it('takes it from the widget only, never from the page', async () => {
      // the page can put anything in the query, the flag the profile once trusted among it
      jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, current: '1', token: 'token' }));
      const ask = jest.spyOn(reader, 'askReader').mockImplementation(async () => ({ ...userParamsStub, id: 'other' }));

      render(<Profile />);

      expect(await screen.findByText('No comments yet.')).toBeInTheDocument();
      await waitFor(() => expect(ask).toHaveBeenCalled());
      await ask.mock.results[0].value;
      expect(screen.queryByTitle('Sign out')).not.toBeInTheDocument();
    });

    it('closes when nobody is signed in to show the own profile of', async () => {
      jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ token: 'token' }));
      jest.spyOn(reader, 'askReader').mockImplementation(async () => null);
      const posted = jest.spyOn(postMessage, 'postMessageToParent').mockReturnValue(true);
      posted.mockClear();

      const { container } = render(<Profile />);

      await waitFor(() => expect(posted).toHaveBeenCalledWith({ profile: null }));
      expect(container.querySelector('.profile')).toBeNull();
    });
  });

  it('sets dir on the root element from the active locale (direction mapping itself is covered by common/direction.test.ts)', () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));

    const { container } = render(<Profile />, {}, 'he');

    expect(container.querySelector('.profile')).toHaveAttribute('dir', 'rtl');
  });
});
