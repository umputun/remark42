import { defineMessages } from 'react-intl';

export const messages = defineMessages<string>({
  telegramMessage1: {
    id: 'auth.telegram-message-1',
    defaultMessage: 'Open Telegram',
  },
  telegramLink: {
    id: 'auth.telegram-link',
    defaultMessage: 'by the link',
  },
  telegramMessage2: {
    id: 'auth.telegram-message-2',
    defaultMessage: 'and click “Start” there.',
  },
  telegramMessage3: {
    id: 'auth.telegram-message-3',
    defaultMessage: 'Afterwards, click “Check” below.',
  },
  telegramOptionalQR: {
    id: 'auth.telegram-optional-qr',
    defaultMessage: 'or by scanning the QR code',
  },
  telegramQR: {
    id: 'auth.telegram-qr',
    defaultMessage: 'Telegram QR-code',
  },
  telegramCheck: {
    id: 'auth.telegram-check',
    defaultMessage: 'Check',
  },
});
