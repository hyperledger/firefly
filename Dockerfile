FROM node:14-buster

ADD ./core /app

WORKDIR /app

RUN npm i \
 && npm t \
 && npm run build

CMD node build/app.js
