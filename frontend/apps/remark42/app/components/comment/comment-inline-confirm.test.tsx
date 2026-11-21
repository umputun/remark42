import { h } from 'preact';
import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';

import type { IntlShape } from 'common/intl';
import { useIntl } from 'common/intl';
import { render } from 'tests/utils';

import type { CommentProps } from './comment';
import { Comment } from './comment';

jest.mock('common/settings', () => ({ ...jest.requireActual('common/settings'), inlineConfirm: true }));

function CommentWithIntl(props: CommentProps) {
  const intl = useIntl();

  return <Comment {...props} intl={intl} />;
}

function getProps(): CommentProps {
  return {
    isCommentsDisabled: false,
    theme: 'light',
    post_info: {
      url: 'http://localhost/post/1',
      count: 2,
      read_only: false,
    },
    view: 'main',
    data: {
      id: 'comment_id',
      text: 'test comment',
      vote: 0,
      time: new Date().toString(),
      pid: 'parent_id',
      score: 0,
      voted_ips: [],
      locator: {
        url: 'somelocatorurl',
        site: 'remark',
      },
      user: {
        id: 'someone',
        picture: 'http://localhost/somepicture-url',
        name: 'username',
        ip: '',
        admin: false,
        block: false,
        verified: false,
      },
    },
    user: {
      id: 'testuser',
      picture: 'http://localhost/testuser-url',
      name: 'test',
      ip: '',
      admin: false,
      block: false,
      verified: false,
    },
    intl: {} as IntlShape,
  };
}

describe('<Comment /> with inline_confirm', () => {
  let confirm: jest.SpyInstance;

  beforeEach(() => {
    confirm = jest.spyOn(window, 'confirm');
  });

  afterEach(() => {
    confirm.mockRestore();
  });

  function renderOwnComment() {
    const props = getProps();
    props.data.user = { ...props.user! };
    props.removeComment = jest.fn();
    props.setReplyEditState = jest.fn();
    render(<CommentWithIntl {...props} />);
    return props;
  }

  it('asks for confirmation inline instead of window.confirm', () => {
    const props = renderOwnComment();

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));

    expect(confirm).not.toHaveBeenCalled();
    expect(screen.getByRole('alertdialog', { name: 'Do you really want to delete this comment?' })).toBeInTheDocument();
    expect(props.removeComment).not.toHaveBeenCalled();
  });

  it('runs the action and closes the dialog on confirm', () => {
    const props = renderOwnComment();

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }));

    expect(props.removeComment).toHaveBeenCalledWith('comment_id');
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });

  it('closes the dialog without running the action on cancel', () => {
    const props = renderOwnComment();

    fireEvent.click(screen.getByRole('button', { name: 'Delete' }));
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

    expect(props.removeComment).not.toHaveBeenCalled();
    expect(screen.queryByRole('alertdialog')).not.toBeInTheDocument();
  });
});
