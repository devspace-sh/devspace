FROM node:8.11.4

RUN mkdir -p /app/website
WORKDIR /app/website

COPY website/package.json .
RUN npm install

COPY . /app

# Application port
EXPOSE 3000

# Remote debugging port
EXPOSE 9229

CMD ["npm", "start"]
