import { h } from 'preact';
import '@testing-library/jest-dom';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as pq from 'utils/parse-query';
import type { Comment, User } from 'common/types';

import { Profile } from './profile';
import { fireEvent } from '@testing-library/dom';

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

describe('<Profile />', () => {
  it('should render preloader', () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    const { queryByLabelText, queryByRole, queryByTestId } = render(<Profile />);

    expect(queryByLabelText('Loading...')).toBeInTheDocument();
    expect(queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    expect(queryByRole('heading', { name: /my comments/i })).not.toBeInTheDocument();
    expect(queryByRole('heading', { name: /comments/i })).not.toBeInTheDocument();
    expect(queryByTestId('comments-counter')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render error', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    jest.spyOn(api, 'getUserComments').mockImplementation(() => {
      throw new Error('error');
    });
    const { queryByLabelText, queryByRole, findByRole, queryByTestId } = render(<Profile />);

    expect(await findByRole('button', { name: /retry/i })).toBeInTheDocument();
    expect(queryByLabelText('Loading...')).not.toBeInTheDocument();
    expect(queryByRole('heading', { name: /my comments/i })).not.toBeInTheDocument();
    expect(queryByRole('heading', { name: /comments/i })).not.toBeInTheDocument();
    expect(queryByTestId('comments-counter')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render user without comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    const getUserComments = jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: [], count: 0 }));

    const { findByText, queryByLabelText, queryByRole, queryByTestId } = render(<Profile />);

    expect(getUserComments).toHaveBeenCalledWith('1', { limit: 10, skip: 0 });
    expect(await findByText("Don't have comments yet")).toBeInTheDocument();
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

    const { findByText, queryByLabelText, queryByRole, queryByTestId } = render(<Profile />);

    expect(await findByText('Comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(commentsStub.length.toString());
    expect(queryByLabelText('Loading...')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render user with comments with load more button', async () => {
    const comments = new Array(15).fill(commentStub);
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => userParamsStub);
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments, count: comments.length }));

    const { findByText, queryByLabelText, queryByRole, queryByTestId } = render(<Profile />);

    expect(await findByText('Comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(comments.length.toString());
    expect(queryByLabelText('Loading...')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /retry/i })).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).toBeInTheDocument();
  });

  it('should render current user without comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, current: '1' }));
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: [], count: 0 }));

    const { getByText, getByTitle, findByText, queryByTestId, queryByRole } = render(<Profile />);

    expect(getByTitle('Sign Out')).toBeInTheDocument();
    expect(getByText('Request my data removal')).toBeInTheDocument();
    expect(await findByText("Don't have comments yet")).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).not.toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render current user with comments without load more button', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, current: '1' }));
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: commentsStub, count: commentsStub.length }));

    const { findByText, queryByTitle, queryByText, queryByTestId, queryByRole } = render(<Profile />);

    expect(await findByText('My comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(commentsStub.length.toString());
    expect(queryByTitle('Sign Out')).toBeInTheDocument();
    expect(queryByText('Request my data removal')).toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });

  it('should render current user with comments with load more button', async () => {
    const comments = Array(15).fill(commentStub);
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, current: '1' }));
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments, count: comments.length }));

    const { findByText, queryByTitle, queryByText, queryByTestId, queryByRole } = render(<Profile />);

    expect(await findByText('My comments')).toBeInTheDocument();
    expect(queryByTestId('comments-counter')).toHaveTextContent(comments.length.toString());
    expect(queryByTitle('Sign Out')).toBeInTheDocument();
    expect(queryByText('Request my data removal')).toBeInTheDocument();
    expect(queryByRole('button', { name: /load more/i })).toBeInTheDocument();
  });

  it('should render user without footer', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    jest
      .spyOn(api, 'getUserComments')
      .mockImplementation(async () => ({ comments: commentsStub, count: commentsStub.length }));

    const { container } = render(<Profile />);

    expect(container.querySelector('profile-footer')).not.toBeInTheDocument();
  });

  it('load more button should dissapear if there no more comments to fetch', async () => {
    const comments = Array(20).fill(commentStub);
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments, count: comments.length }));

    const { findByRole, queryByRole } = render(<Profile />);

    expect(await findByRole('button', { name: /load more/i })).toBeInTheDocument();
    fireEvent.click(await findByRole('button', { name: /load more/i }));
    expect(queryByRole('button', { name: /load more/i })).not.toBeInTheDocument();
  });
});
