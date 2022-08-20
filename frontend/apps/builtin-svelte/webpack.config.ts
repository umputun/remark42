import { isProduction } from './webpack/webpack.common';
import { devConfig } from './webpack/webpack.dev';
import { prodConfig } from './webpack/webpack.prod';

export default isProduction ? prodConfig : devConfig;
