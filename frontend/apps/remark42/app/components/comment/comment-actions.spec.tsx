import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';

import { createIntl } from 'common/intl';
import en from 'locales/en.json';
import { draw } from 'tests/plain';

import type { ActionsProps as Props } from './actions.js';
import { mountActions } from './actions.js';

function drawActions(props: Props) {
  return draw('div', (root) => mountActions(root, props, createIntl('en', en)));
}

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
    drawActions(props);
    expect(screen.queryByText('Reply')).not.toBeInTheDocument();
    expect(screen.getByText('Cancel')).toBeInTheDocument();
  });

  it.each([
    [{ editable: false, editDeadline: Date.now() + 300 * 1000 }],
    [{ editable: true, editDeadline: undefined }],
  ] as Partial<Props>[][])('should not render "Edit" when editing is not available', (override) => {
    Object.assign(props, override);
    drawActions(props);
    expect(screen.getByText('Hide user')).toBeInTheDocument();
  });

  it('should not render "Delete" for current user comments when editDeadline is undefined', () => {
    props.currentUser = true;
    props.editDeadline = undefined; // set editDeadline to undefined
    drawActions(props);
    expect(screen.queryByText('Delete')).not.toBeInTheDocument();
  });

  describe('admin actions', () => {
    it.each([[{ currentUser: false, admin: true }], [{ currentUser: true, admin: true }]] as Partial<Props>[][])(
      'should render "Delete" on all comments for admin',
      (override) => {
        Object.assign(props, override);
        drawActions(props);
        expect(screen.getByText('Delete')).toBeInTheDocument();
      }
    );

    it('calls `onToggleEditing` when edit button is pressed', () => {
      props.editable = true;
      props.editDeadline = Date.now() + 300 * 1000;
      drawActions(props);
      fireEvent(screen.getByText('Edit'), new MouseEvent('click', { bubbles: true }));
      expect(props.onToggleEditing).toHaveBeenCalledTimes(1);
    });
  });
});
