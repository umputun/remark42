const {createTransformer} = require(`babel-jest`);
const babelOptions = require('./babelOptions');


module.exports = createTransformer({
  babelrc: false,
  ...babelOptions,
});
