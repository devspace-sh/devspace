FROM node:13.14-alpine

RUN mkdir /app
WORKDIR /app

COPY package.json .
RUN npm install

COPY . .

# Application port
EXPOSE 3000

# Remote debugging port
EXPOSE 9229

CMD ["npm", "start"]
