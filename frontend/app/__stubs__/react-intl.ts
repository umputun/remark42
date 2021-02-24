jest.mock('react-intl', () => {
  const messages = require('locales/en.json');
  const reactIntl = jest.requireActual('react-intl');
  const intlProvider = new reactIntl.IntlProvider({ locale: 'en', messages }, {});

  return {
    ...reactIntl,
    useIntl: () => intlProvider.state.intl,
  };
});
