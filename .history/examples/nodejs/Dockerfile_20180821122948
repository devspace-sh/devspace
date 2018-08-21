FROM node:8.11

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . /app

CMD ["npm", "start"]
