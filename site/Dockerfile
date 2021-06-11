FROM node:14-alpine

WORKDIR site

COPY ./ /site

RUN yarn --frozen-lockfile

CMD yarn build

