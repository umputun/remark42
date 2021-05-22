import { h } from 'preact';
import '@testing-library/jest-dom';
import { waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as pq from 'utils/parse-query';
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

describe('<Profile />', () => {
  it('should render preloader', () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    const { container } = render(<Profile />);

    expect(container.querySelector('[aria-label="Loading..."]')).toBeInTheDocument();
  });

  it('should render without comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    const getUserComments = jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: [] }));

    const { getByText } = render(<Profile />);

    await waitFor(() => expect(getUserComments).toHaveBeenCalledWith('1'));
    await waitFor(() => expect(getByText("Don't have comments yet")).toBeInTheDocument());
  });

  it('should render user with comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => userParamsStub);
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: commentsStub }));

    const { getByText } = render(<Profile />);

    await waitFor(() => expect(getByText('Recent comments')).toBeInTheDocument());
  });

  it('shoud render current user without comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, current: '1' }));

    const { getByText, getByTitle } = render(<Profile />);

    expect(getByTitle('Sign Out')).toBeInTheDocument();
    expect(getByText('Request my data removal')).toBeInTheDocument();
  });

  it('shoud render current user with comments', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub, current: '1' }));
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: commentsStub }));

    const { getByText, getByTitle } = render(<Profile />);

    expect(getByTitle('Sign Out')).toBeInTheDocument();
    expect(getByText('Request my data removal')).toBeInTheDocument();
    await waitFor(() => expect(getByText('My recent comments')).toBeInTheDocument());
  });

  it('should render user without footer', async () => {
    jest.spyOn(pq, 'parseQuery').mockImplementation(() => ({ ...userParamsStub }));
    jest.spyOn(api, 'getUserComments').mockImplementation(async () => ({ comments: commentsStub }));

    const { container } = render(<Profile />);

    expect(container.querySelector('profile-footer')).not.toBeInTheDocument();
  });
});
