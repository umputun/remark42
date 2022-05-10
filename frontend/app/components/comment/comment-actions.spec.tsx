import { h } from 'preact';
import '@testing-library/jest-dom';
import { CommentActions, Props } from './comment-actions';
import { render } from 'tests/utils';
import { fireEvent, screen, waitFor } from '@testing-library/preact';

function getProps(): Props {
  return {
    pinned: false,
    admin: false,
    currentUser: false,
    copied: false,
    bannedUser: false,
    readOnly: false,
    editing: false,
    replying: false,
    onCopy: jest.fn(),
    onDelete: jest.fn(),
    onToggleEditing: jest.fn(),
    onTogglePin: jest.fn(),
    onToggleReplying: jest.fn(),
    onHideUser: jest.fn(),
    onBlockUser: jest.fn(),
    onUnblockUser: jest.fn(),
    onDisableEditing: jest.fn(),
    editable: false,
    editDeadline: undefined,
  };
}
describe('<CommentActions/>', () => {
  let props: Props;

  beforeEach(() => {
    props = getProps();
  });
  afterEach(() => {
    jest.resetAllMocks();
  })

  it('should render "Reply"', () => {
    render(<CommentActions {...props} />);
    expect(screen.getByText('Reply')).toBeVisible();
  });

  it('should not render "Reply" in read only mode', () => {
    props.readOnly = true;
    render(<CommentActions {...props} />);
    expect(screen.queryByText('Reply')).not.toBeInTheDocument();
  });

  it('should not render "Cancel" instead "Reply" in replying mode', () => {
    props.replying = true;
    render(<CommentActions {...props} />);
    expect(screen.queryByText('Reply')).not.toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it('should render "Hide" on comments not from currentUser', () => {
    props.currentUser = false;
    render(<CommentActions {...props} />);
    expect(screen.getByText('Hide')).toBeVisible();
  });

  it('should not render "Hide" on comments not from currentUser', () => {
    props.currentUser = true;
    render(<CommentActions {...props} />);
    expect(screen.queryByText('Hide')).not.toBeInTheDocument();
  });

  it('should render "Edit" and timer when editing is available', async () => {
    Object.assign(props, { editable: true, editDeadline: Date.now() + 300 * 1000 });
    render(<CommentActions {...props} />);
    expect(screen.getByText('Edit')).toBeInTheDocument();
    await waitFor(() => expect(['300s', '299s']).toContain(screen.getByRole('timer').textContent));
  });

  it('should render "Cancel" instead "Edit" in editing mode', async () => {
    Object.assign(props, { editable: true, editing: true, editDeadline: Date.now() + 300 * 1000 });
    render(<CommentActions {...props} />);
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it.each([
    [{ editable: false, editDeadline: Date.now() + 300 * 1000 }],
    [{ editable: true, editDeadline: undefined }],
  ] as Partial<Props>[][])('should not render "Edit" when editing is not available', (override) => {
    Object.assign(props, override);
    render(<CommentActions {...props} />);
    expect(screen.getByText('Hide')).toBeInTheDocument();
  });

  it('should render "Delete" for current user comments', () => {
    props.currentUser = true;
    render(<CommentActions {...props} />);
    expect(screen.getByText('Delete')).toBeInTheDocument();
  });

  it('should not render "Delete" for other users comments', () => {
    render(<CommentActions {...props} />);
    expect(screen.queryByText('Delete')).not.toBeInTheDocument();
  });

  describe('admin actions', () => {
    it('should render "Copy"', () => {
      props.admin = true;
      render(<CommentActions {...props} />);
      expect(screen.getByText('Copy')).toBeInTheDocument();
    });

    it('should render "Copied" when comment copied', () => {
      Object.assign(props, { admin: true, copied: true });
      render(<CommentActions {...props} />);
      expect(screen.getByText('Copied!')).toBeInTheDocument();
    });

    it('should render "Pin"', () => {
      props.admin = true;
      render(<CommentActions {...props} />);
      expect(screen.getByText('Pin')).toBeInTheDocument();
    });

    it('should render "Unpin" when comment is pinned', () => {
      Object.assign(props, { admin: true, pinned: true });
      render(<CommentActions {...props} />);
      expect(screen.getByText('Unpin')).toBeInTheDocument();
    });

    it.each([[{ currentUser: false, admin: true }], [{ currentUser: true, admin: true }]] as Partial<Props>[][])(
      'should render "Delete" on all comments for admin',
      (override) => {
        Object.assign(props, override);
        render(<CommentActions {...props} />);
        expect(screen.getByText('Delete')).toBeInTheDocument();
      }
    );

    it('should render admin actions in right order', () => {
      props.admin = true;
      render(<CommentActions {...props} />);
      expect(screen.getByTestId('comment-actions-additional').children[0]).toHaveTextContent('Hide');
      expect(screen.getByTestId('comment-actions-additional').children[1]).toHaveTextContent('Copy');
      expect(screen.getByTestId('comment-actions-additional').children[2]).toHaveTextContent('Pin');
      expect(screen.getByTestId('comment-actions-additional').children[3]).toHaveTextContent('Block');
      expect(screen.getByTestId('comment-actions-additional').children[4]).toHaveTextContent('Delete');
    });

    it('calls `onToggleEditing` when edit button is pressed', () => {
      render(<CommentActions {...props} />);
      fireEvent(screen.getByText('Edit'), new MouseEvent('click', { bubbles: true }));
      expect(props.onToggleEditing).toHaveBeenCalledTimes(1);
      fireEvent(screen.getByText('Cancel'), new MouseEvent('click', { bubbles: true }));
      expect(props.onToggleEditing).toHaveBeenCalledTimes(2);
    })
  });
});
