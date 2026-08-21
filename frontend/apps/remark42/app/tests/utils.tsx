import { h, ComponentChild } from 'preact';
import { render as originalRender } from '@testing-library/preact';
import { Provider } from 'store/context';

import { IntlProvider } from 'common/intl';
import en from 'locales/en.json';
import { mockStore } from '__stubs__/store';
import { StoreState } from 'store';

export function render(children: ComponentChild, s: Partial<StoreState> = {}) {
  const props = originalRender(
    <IntlProvider locale="en" messages={en}>
      <Provider store={mockStore(s)}>{children}</Provider>
    </IntlProvider>
  );

  return {
    ...props,
    rerender(children: ComponentChild) {
      props.rerender(
        <IntlProvider locale="en" messages={en}>
          <Provider store={mockStore(s)}>{children}</Provider>
        </IntlProvider>
      );
    },
  };
}
