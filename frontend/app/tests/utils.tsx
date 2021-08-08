import { h, ComponentChild } from 'preact';
import { IntlProvider } from 'react-intl';
import { render as originalRender } from '@testing-library/preact';
import { Provider } from 'react-redux';

import en from 'locales/en.json';
import { mockStore } from '__stubs__/store';
import { StoreState } from 'store';

export function render(children: ComponentChild, s: Partial<StoreState> = {}) {
  return originalRender(
    <IntlProvider locale="en" messages={en}>
      <Provider store={mockStore(s)}>{children}</Provider>
    </IntlProvider>
  );
}
