import '@testing-library/jest-dom';
import { screen } from '@testing-library/preact';

import { render } from 'tests/utils';

import { TelegramLink } from './telegram-link';

const handleSubmitSpy = jest.fn();

const defaultProps = {
  bot: 'remark42_bot',
  token: '718b5a43fb18136beb273e9261dd895578d00f63',
  onSubmit: handleSubmitSpy,
};

describe('<TelegramLink/>', () => {
  let widthBackup: number;
  beforeAll(() => {
    widthBackup = window.screen.width;
  });
  afterAll(() => {
    Object.defineProperties(window.screen, {
      width: {
        writable: true,
        value: widthBackup!,
      },
    });
  });
  it('should render text', () => {
    render(<TelegramLink {...defaultProps} />);
    expect(screen.getByText(/Open Telegram/)).toBeInTheDocument();
    expect(screen.getByText(/by the link/)).toBeInTheDocument();
    expect(screen.getByText(/and click “Start” there./)).toBeInTheDocument();
    expect(screen.getByText(/Afterwards, click “Check” below./)).toBeInTheDocument();
  });
  it('should show QR code on desktop', () => {
    Object.defineProperties(window.screen, {
      width: {
        writable: true,
        value: 1000,
      },
    });
    render(<TelegramLink {...defaultProps} />);
    expect(screen.getByText(/or by scanning the QR code/)).toBeInTheDocument();
    expect(screen.getByAltText(/Telegram QR-code/)).toBeInTheDocument();
  });

  it('should show contain correct QR src', () => {
    Object.defineProperties(window.screen, {
      width: {
        writable: true,
        value: 1000,
      },
    });
    render(<TelegramLink {...defaultProps} />);
    expect(screen.getByAltText(/Telegram QR-code/)).toHaveAttribute(
      'src',
      `http://test.com/api/v1/qr/telegram?url=https://t.me/${defaultProps.bot}/?start=${defaultProps.token}`
    );
  });
  it('should NOT show QR code on mobile', () => {
    Object.defineProperties(window.screen, {
      width: {
        writable: true,
        value: 500,
      },
    });
    render(<TelegramLink {...defaultProps} />);
    expect(screen.queryByAltText(/Telegram QR-code/)).not.toBeInTheDocument();
  });

  it('should show error if any', () => {
    render(<TelegramLink {...defaultProps} errorMessage="Foo Error" />);
    expect(screen.getByText('Foo Error')).toBeInTheDocument();
  });
});
