import { parseBooleansFromDictionary } from './parse-booleans-from-dictionary';

const defaultProps = {
  components: 'embed,counter',
  host: 'http://127.0.0.1:9000',
  locale: 'ru',
  site_id: 'remark',
  theme: 'dark',
};

describe('getConfigMerge', () => {
  it('when we need to get one field and it is "true"', () => {
    const params = {
      ...defaultProps,
      simple: 'true',
      simple_view: 'true',
    };
    expect(parseBooleansFromDictionary(params, 'simple_view')).toEqual({ simple_view: true });
  });

  it('when we need to get one field and it is "false"', () => {
    const params = {
      ...defaultProps,
      simple: 'true',
      simple_view: 'false',
    };
    expect(parseBooleansFromDictionary(params, 'simple_view')).toEqual({ simple_view: false });
  });
  it('when we need to get one or more fields', () => {
    const params = {
      ...defaultProps,
      simple: 'false',
      simple_view: 'true',
    };
    expect(parseBooleansFromDictionary(params, 'simple_view', 'simple')).toEqual({
      simple: false,
      simple_view: true,
    });
  });

  it('when the required field does not exist', () => {
    const params = {
      ...defaultProps,
      simple: 'true',
    };
    expect(parseBooleansFromDictionary(params, 'simple_view')).toEqual({});
  });

  it('when the field has the wrong format', () => {
    const params = {
      ...defaultProps,
      simple: 'true',
      simple_view: 'dark',
    };
    expect(parseBooleansFromDictionary(params, 'simple_view', 'simple')).toEqual({ simple: true });
  });
});
