FROM node:16-alpine

WORKDIR /site
COPY ./ /site
RUN yarn --frozen-lockfile

CMD yarn build

