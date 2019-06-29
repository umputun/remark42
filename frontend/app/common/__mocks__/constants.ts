// @ts-ignore
const mock: typeof import('@app/common/constants') = {
  ...jest.requireActual('@app/common/constants'),
  BASE_URL: 'https://demo.remark42.com/',
};

module.exports = mock;
