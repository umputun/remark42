import { h } from 'preact';
import '@testing-library/jest-dom';
import type { Props } from './comment-actions';
import { CommentActions } from './comment-actions';
import { render } from 'tests/utils';
import { fireEvent, screen } from '@testing-library/preact';

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
  });

  it('should not render "Cancel" instead "Reply" in replying mode', () => {
    props.replying = true;
    render(<CommentActions {...props} />);
    expect(screen.queryByText('Reply')).not.toBeInTheDocument();
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

  it('should not render "Delete" for current user comments when editDeadline is undefined', () => {
    props.currentUser = true;
    props.editDeadline = undefined; // set editDeadline to undefined
    render(<CommentActions {...props} />);
    expect(screen.queryByText('Delete')).not.toBeInTheDocument();
  });

  describe('admin actions', () => {
    it.each([[{ currentUser: false, admin: true }], [{ currentUser: true, admin: true }]] as Partial<Props>[][])(
      'should render "Delete" on all comments for admin',
      (override) => {
        Object.assign(props, override);
        render(<CommentActions {...props} />);
        expect(screen.getByText('Delete')).toBeInTheDocument();
      }
    );

    it('calls `onToggleEditing` when edit button is pressed', () => {
      props.editable = true;
      props.editDeadline = Date.now() + 300 * 1000;
      render(<CommentActions {...props} />);
      fireEvent(screen.getByText('Edit'), new MouseEvent('click', { bubbles: true }));
      expect(props.onToggleEditing).toHaveBeenCalledTimes(1);
    });
  });
});
