const fs = require('fs');
const path = require('path');

const webpack = require('webpack');
const ExtractText = require('extract-text-webpack-plugin');
const Clean = require('clean-webpack-plugin');
const Html = require('html-webpack-plugin');
const Provide = webpack.ProvidePlugin;
const Define = webpack.DefinePlugin;

const publicFolder = path.resolve(__dirname, 'public');
const env = process.env.NODE_ENV || 'dev';
const hash = env === 'production' ? '.[chunkhash]' : '.[hash]';

module.exports = {
  context: __dirname,
  entry: {
    app: './app/app',
  },
  output: {
    path: publicFolder,
    // publicPath: '/path/to/public',
    filename: `app${hash}.js`
  },
  resolve: {
    extensions: ['.jsx', '.js'],
    modules: [path.resolve(__dirname, 'app'), path.resolve(__dirname, 'node_modules')]
  },
  module: {
    rules: [
      {
        test: /\.jsx?$/,
        exclude: /(node_modules|\.vendor\.js$)/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: [
              ['env', {
                targets: ['> 1%', 'android >= 4.4.4', 'ios >= 9'],
                useBuiltIns: true,
              }],
            ],
            plugins: ['transform-object-rest-spread', ['transform-react-jsx', { 'pragma': 'h' }]],
          }
        }
      },
      {
        test: /\.scss$/,
        use: [
          // TODO: style-loader must be turned on only for dev; for build we need extractCSS
          'style-loader',
          'css-loader',
          {
            loader: 'postcss-loader',
            options: {
              plugins: [
                require('autoprefixer')({browsers: ['> 1%']}),
                require('postcss-url')({url: 'inline', maxSize: 5}),
                require('postcss-csso')
              ]
            }
          },
          {
            loader: 'sass-loader',
            options: {
              // data: fs.readFileSync(path.resolve(__dirname, 'common/vars/vars.scss'), 'utf-8')
            }
          }
        ]
      },
      {
        test: /\.(png|jpg|jpeg|gif)$/,
        use: {
          loader: 'file-loader',
          options: {
            name: `files/[name].[hash].[ext]`
          }
        }
      }
    ]
  },
  plugins: [
    new Clean(publicFolder),
    new Provide({
      b: 'bem-react-helper',
    }),
    new Define({
      'process.env.NODE_ENV': JSON.stringify(process.env.NODE_ENV),
    }),
    new Html({ template: path.resolve(__dirname, 'index.ejs') }), // we should add it only in dev
    new ExtractText({
      filename: `app${hash}.css`,
      allChunks: true
    }),
    // new webpack.optimize.UglifyJsPlugin(), // TODO: enable it
    new webpack.optimize.ModuleConcatenationPlugin(),
  ],
  watch: env === 'dev',
  watchOptions: {
    ignored: /(node_modules|\.vendor\.js$)/
  },
  devServer: {
    host: '0.0.0.0',
    port: 8080,
    contentBase: publicFolder,
  },
};
