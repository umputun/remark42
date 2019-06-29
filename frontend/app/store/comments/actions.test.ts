import fetchMock from 'fetch-mock';

import { Comment, User } from '@app/common/types';
import { fetchNewComments, unwrapNewComments } from './actions';
import { StoreState } from '..';
import { mockHeaders } from '@app/testUtils/mockHeaders';
import '@app/testUtils/mockPostInfo';
import { mockStore } from '@app/testUtils/mockStore';
import { mapTree } from './utils';

type TestUser = Pick<User, 'id'>;
type TestComment = Pick<Comment, 'id' | 'pid' | 'text' | 'edit' | 'new'> & { user: TestUser };
interface TestNode {
  comment: TestComment;
  replies?: TestNode[];
}

describe(`fetchNewComments`, () => {
  beforeAll(() => {
    mockHeaders.mock();
  });

  afterAll(() => {
    mockHeaders.restore();
  });

  afterEach(() => {
    fetchMock.restore();
  });

  const user1: TestUser = { id: 'first' };
  const user2: TestUser = { id: 'second' };

  const getExampleState: () => { hiddenUsers: StoreState['hiddenUsers']; comments: TestNode[] } = () => ({
    hiddenUsers: {},
    comments: [
      {
        comment: {
          id: 'first',
          pid: '',
          text: 'first comment',
          user: user1,
        },
      },
      {
        comment: {
          id: 'second',
          pid: '',
          text: 'second comment',
          user: user2,
        },
        replies: [
          {
            comment: {
              id: '1r2',
              pid: 'second',
              text: 'reply',
              user: user1,
            },
          },
        ],
      },
    ],
  });

  it(`do nothing if there is no update`, async () => {
    fetchMock.mock(/.*/, {
      comments: [],
    });
    const dispatch = jest.fn();
    await fetchNewComments()(dispatch, getExampleState as any, undefined);
    expect(dispatch).not.toBeCalled();
  });

  it(`adds comment`, async () => {
    const store = mockStore(getExampleState());
    const newComment: TestComment = {
      id: 'new',
      pid: '',
      text: 'new comment',
      user: user1,
    };
    fetchMock.mock(/.*/, {
      comments: [newComment],
    });
    await store.dispatch(fetchNewComments());
    expect(store.getActions()).toIncludeAllMembers([
      {
        type: 'COMMENTS/SET',
        comments: [...getExampleState().comments, { comment: newComment }],
      },
    ]);
  });

  it(`adds comment reply`, async () => {
    const store = mockStore(getExampleState());
    const newComment: TestComment = {
      id: 'new',
      pid: 'second',
      text: 'new comment',
      user: user1,
    };
    fetchMock.mock(/.*/, {
      comments: [newComment],
    });
    await store.dispatch(fetchNewComments());

    const expected = {
      type: 'COMMENTS/SET',
      comments: getExampleState().comments,
    };
    expected.comments[1].replies!.push({ comment: { ...newComment, new: true } });

    expect(store.getActions()).toIncludeAllMembers([expected]);
  });

  it(`edits comment`, async () => {
    const store = mockStore(getExampleState());
    const newComment: TestComment = {
      id: 'second',
      pid: '',
      text: 'second comment edited',
      user: user1,
      edit: {
        summary: '',
        time: '00:00',
      },
    };
    fetchMock.mock(/.*/, {
      comments: [newComment],
    });
    await store.dispatch(fetchNewComments());

    const expected = {
      type: 'COMMENTS/SET',
      comments: getExampleState().comments,
    };
    expected.comments[1].comment = newComment;

    expect(store.getActions()).toIncludeAllMembers([expected]);
  });
});

describe(`unwrapNewComments`, () => {
  beforeAll(() => {
    mockHeaders.mock();
  });

  afterAll(() => {
    mockHeaders.restore();
  });

  afterEach(() => {
    fetchMock.restore();
  });

  const user1: TestUser = { id: 'first' };
  const user2: TestUser = { id: 'second' };

  const getExampleState: () => { hiddenUsers: StoreState['hiddenUsers']; comments: TestNode[] } = () => ({
    hiddenUsers: {},
    comments: [
      {
        comment: {
          id: 'first',
          pid: '',
          text: 'first comment',
          user: user1,
        },
      },
      {
        comment: {
          id: 'second',
          pid: '',
          text: 'second comment',
          user: user2,
        },
        replies: [
          {
            comment: {
              id: '2r2',
              pid: 'second',
              text: 'reply',
              user: user1,
              new: true,
            },
            replies: [
              {
                comment: {
                  id: '2r3',
                  pid: '2r2',
                  text: 'reply',
                  user: user1,
                  new: true,
                },
              },
            ],
          },
        ],
      },
      {
        comment: {
          id: 'third',
          pid: '',
          text: 'third comment',
          user: user2,
        },
        replies: [
          {
            comment: {
              id: '3r2',
              pid: 'second',
              text: 'reply',
              user: user1,
              new: true,
            },
            replies: [
              {
                comment: {
                  id: '3r3',
                  pid: '1r2',
                  text: 'reply',
                  user: user1,
                  new: true,
                },
              },
            ],
          },
        ],
      },
    ],
  });

  it(`unsets 'new' flag recursively`, async () => {
    const store = mockStore(getExampleState());
    await store.dispatch(unwrapNewComments('second'));
    expect(store.getActions()).toIncludeAllMembers([
      {
        type: 'COMMENTS/SET',
        comments: mapTree(getExampleState().comments as any, c => {
          if (!c.new) return c;
          if (c.id === '3r2') return c;
          if (c.id === '3r3') return c;
          return { ...c, new: false };
        }),
      },
    ]);
  });
});
