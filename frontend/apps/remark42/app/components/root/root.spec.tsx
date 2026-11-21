import '@testing-library/jest-dom';
import { fireEvent, screen, waitFor } from '@testing-library/preact';

import { render } from 'tests/utils';
import * as api from 'common/api';
import * as postMessage from 'utils/post-message';
import type { User } from 'common/types';
import type { StoreState } from 'store';

import { ConnectedRoot } from './root';

// a site that sets max_shown_comments, which settings.ts resolves without regard to the device.
// jsdom's user agent names no mobile platform, so this renders the desktop widget
jest.mock('common/settings', () => ({ ...jest.requireActual('common/settings'), maxShownComments: 2 }));
// what a thread renders is covered by its own tests; here only which threads the root renders matters
jest.mock('components/thread', () => ({
  Thread: ({ id }: { id: string }) => <div data-testid="thread">{id}</div>,
}));

const stateStub: Partial<StoreState> = {
  comments: {
    sort: '-active',
    isFetching: false,
    childComments: {},
    topComments: [],
    pinnedComments: [],
    allComments: {},
    activeComment: null,
  },
  collapsedThreads: {},
  theme: 'light',
  info: { url: 'test-url', count: 0, read_only: false },
  hiddenUsers: {},
  bannedUsers: [],
  user: null,
};

describe('<ConnectedRoot />', () => {
  afterEach(() => {
    jest.restoreAllMocks();
  });

  it('reports iframe height only after the initial user fetch settles', async () => {
    let resolveUser!: (user: User | null) => void;
    jest.spyOn(api, 'getUser').mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveUser = resolve;
        })
    );
    // keep comments loading so only the user fetch controls the first height report
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);

    render(<ConnectedRoot />, stateStub);

    // while the global preloader is shown, no height must be sent to the parent page,
    // otherwise the parent shrinks the iframe to the preloader size and it blinks
    expect(updateIframeHeight).not.toHaveBeenCalled();

    resolveUser(null);

    await waitFor(() => expect(updateIframeHeight).toHaveBeenCalled());
  });

  it('falls back to reporting iframe height when the user fetch hangs', () => {
    jest.useFakeTimers();
    try {
      jest.spyOn(api, 'getUser').mockImplementation(() => new Promise(() => undefined));
      jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
      const updateIframeHeight = jest.spyOn(postMessage, 'updateIframeHeight').mockImplementation(() => undefined);

      render(<ConnectedRoot />, stateStub);
      expect(updateIframeHeight).not.toHaveBeenCalled();

      jest.advanceTimersByTime(5000);
      expect(updateIframeHeight).toHaveBeenCalled();
    } finally {
      jest.useRealTimers();
    }
  });

  it('sets dir on the root element from the active locale (direction mapping itself is covered by common/direction.test.ts)', () => {
    jest.spyOn(api, 'getUser').mockImplementation(() => new Promise(() => undefined));
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));

    const { container } = render(<ConnectedRoot />, stateStub, 'he');

    expect(container.firstElementChild).toHaveAttribute('dir', 'rtl');
  });

  it('limits the top-level comments to max_shown_comments on desktop and shows ten more on demand', async () => {
    jest.spyOn(api, 'getUser').mockResolvedValue(null);
    jest.spyOn(api, 'getPostComments').mockImplementation(() => new Promise(() => undefined));
    const topComments = Array.from({ length: 15 }, (_, i) => `comment-${i}`);

    render(<ConnectedRoot />, { ...stateStub, comments: { ...stateStub.comments!, topComments } });

    await waitFor(() => expect(screen.getAllByTestId('thread')).toHaveLength(2));
    expect(screen.getAllByTestId('thread').map((thread) => thread.textContent)).toEqual(['comment-0', 'comment-1']);

    fireEvent.click(screen.getByRole('button', { name: 'Show more' }));
    expect(screen.getAllByTestId('thread')).toHaveLength(12);

    fireEvent.click(screen.getByRole('button', { name: 'Show more' }));
    expect(screen.getAllByTestId('thread')).toHaveLength(15);
    expect(screen.queryByRole('button', { name: 'Show more' })).not.toBeInTheDocument();
  });
});
