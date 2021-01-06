const constantsMock: typeof import('common/constants') = {
  ...jest.requireActual('common/constants'),
  BASE_URL: 'https://demo.remark42.com/',
};

module.exports = constantsMock;
