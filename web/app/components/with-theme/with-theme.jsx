/** @jsx h */
import { h } from 'preact';
import store from 'common/store';

const withTheme = PlainComponent => {
  const ThemedComponent = props => {
    const { mods = {} } = props;
    const theme = store.get('theme');

    return <PlainComponent {...props} mods={{ theme, ...mods }} />;
  };

  return ThemedComponent;
};

export default withTheme;
