import { h, ComponentChild } from 'preact';
import { IntlProvider } from 'react-intl';
import { render as originalRender } from '@testing-library/preact';

import en from 'locales/en.json';

export function render(children: ComponentChild) {
  return originalRender(
    <IntlProvider locale="en" messages={en}>
      {children}
    </IntlProvider>
  );
}
