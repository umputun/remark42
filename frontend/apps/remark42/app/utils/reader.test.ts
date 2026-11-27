import type { User } from 'common/types';

import { READER_TIMEOUT, answerReaderAsks, askReader, issueReaderToken } from './reader';

const user: User = {
  id: 'github_reader',
  name: 'Reader',
  picture: 'http://localhost/reader.png',
  ip: 'ip-hash',
  admin: true,
  block: false,
  verified: true,
};

function sibling(): Window {
  const frame = document.createElement('iframe');
  document.body.appendChild(frame);
  return frame.contentWindow!;
}

function deliver(data: unknown, source: Window | null, origin = window.location.origin) {
  window.dispatchEvent(new MessageEvent('message', { data, origin, source }));
}

describe('answerReaderAsks', () => {
  let stop: () => void;

  afterEach(() => {
    stop();
    document.body.innerHTML = '';
  });

  it('answers the frame that asks with a token it issued, addressed to this origin only', () => {
    stop = answerReaderAsks(() => user);
    const asker = sibling();
    const answer = jest.spyOn(asker, 'postMessage').mockImplementation(() => undefined);
    const token = issueReaderToken();

    deliver({ readerAsk: token }, asker);

    // the reader's fields the profile shows and nothing else of the record
    expect(answer).toHaveBeenCalledWith(
      { readerAnswer: token, reader: { id: 'github_reader', name: 'Reader', picture: 'http://localhost/reader.png' } },
      window.location.origin
    );
  });

  it('answers null when nobody is signed in', () => {
    stop = answerReaderAsks(() => null);
    const asker = sibling();
    const answer = jest.spyOn(asker, 'postMessage').mockImplementation(() => undefined);
    const token = issueReaderToken();

    deliver({ readerAsk: token }, asker);

    expect(answer).toHaveBeenCalledWith({ readerAnswer: token, reader: null }, window.location.origin);
  });

  it('answers no other origin and no token it did not issue', () => {
    stop = answerReaderAsks(() => user);
    const asker = sibling();
    const answer = jest.spyOn(asker, 'postMessage').mockImplementation(() => undefined);
    const token = issueReaderToken();

    deliver({ readerAsk: token }, asker, 'https://embedding.example');
    deliver({ readerAsk: 'another widget' }, asker);

    expect(answer).not.toHaveBeenCalled();
  });

  it('stops answering once stopped', () => {
    stop = answerReaderAsks(() => user);
    const asker = sibling();
    const answer = jest.spyOn(asker, 'postMessage').mockImplementation(() => undefined);

    stop();
    deliver({ readerAsk: issueReaderToken() }, asker);

    expect(answer).not.toHaveBeenCalled();
  });
});

describe('askReader', () => {
  afterEach(() => {
    jest.useRealTimers();
    document.body.innerHTML = '';
  });

  it('asks every other frame of the page, addressed to this origin only', () => {
    const frames = [sibling(), sibling()];
    const asks = frames.map((frame) => jest.spyOn(frame, 'postMessage').mockImplementation(() => undefined));

    askReader('token');

    for (const ask of asks) {
      expect(ask).toHaveBeenCalledWith({ readerAsk: 'token' }, window.location.origin);
    }
  });

  it('takes the answer to its own token from this origin', async () => {
    const asked = askReader('token');

    deliver({ readerAnswer: 'token', reader: { id: 'claimed' } }, null, 'https://embedding.example');
    deliver({ readerAnswer: 'another token', reader: { id: 'other' } }, null);
    deliver({ readerAnswer: 'token', reader: { id: 'github_reader', name: 'Reader', picture: '' } }, null);

    await expect(asked).resolves.toEqual({ id: 'github_reader', name: 'Reader', picture: '' });
  });

  it('takes no reader when no answer comes', async () => {
    jest.useFakeTimers();
    const asked = askReader('token');

    jest.advanceTimersByTime(READER_TIMEOUT);

    await expect(asked).resolves.toBeNull();
  });
});
