FROM node:13.14-alpine

# Set working directory
WORKDIR /app

# Add package.json to WORKDIR and install dependencies
COPY package*.json ./
RUN npm install

# Add source code files to WORKDIR
COPY . .

# Application port (optional)
EXPOSE 3000

# Debugging port (optional)
# For remote debugging, add this port to devspace.yaml: dev.ports[*].forward[*].port: 9229
EXPOSE 9229

# Container start command (DO NOT CHANGE and see note below)
CMD ["npm", "start"]

# To start using a different `npm run [name]` command (e.g. to use nodemon + debugger),
# edit devspace.yaml:
# 1) remove: images.app.injectRestartHelper (or set to false)
# 2) add this: images.app.cmd: ["npm", "run", "dev"]