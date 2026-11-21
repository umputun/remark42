import { h } from 'preact';
import '@testing-library/jest-dom';
import { fireEvent, screen } from '@testing-library/preact';

import { render } from 'tests/utils';

import { ConfirmDialog } from './confirm-dialog';

describe('<ConfirmDialog />', () => {
  it('labels the dialog with its message and focuses cancel', () => {
    render(<ConfirmDialog message="Delete this comment?" onConfirm={jest.fn()} onCancel={jest.fn()} />);

    expect(screen.getByRole('alertdialog', { name: 'Delete this comment?' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus();
  });

  it('calls onConfirm and onCancel from the buttons', () => {
    const onConfirm = jest.fn();
    const onCancel = jest.fn();
    render(<ConfirmDialog message="Sure?" onConfirm={onConfirm} onCancel={onCancel} />);

    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }));
    expect(onConfirm).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));
    expect(onCancel).toHaveBeenCalledTimes(1);
  });

  it('cancels on Escape', () => {
    const onCancel = jest.fn();
    render(<ConfirmDialog message="Sure?" onConfirm={jest.fn()} onCancel={onCancel} />);

    fireEvent.keyDown(screen.getByRole('button', { name: 'Cancel' }), { key: 'Escape' });
    expect(onCancel).toHaveBeenCalledTimes(1);
  });
});
