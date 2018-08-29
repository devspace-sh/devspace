FROM node:8.11

RUN mkdir -p /app/website
WORKDIR /app/website

COPY website/package.json .
RUN npm install

COPY . /app

CMD ["npm", "start"]
