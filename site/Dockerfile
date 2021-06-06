FROM node:14-alpine

WORKDIR site

COPY ./ /site
COPY ../docs /site/src/docs

RUN yarn --frozen-lockfile && \
	yarn build && \
	ls -la /site/build

CMD ["sleep", "100"]

