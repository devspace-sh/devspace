#!/bin/bash
set +e  # Continue on errors

export NODE_ENV=development
if [ -f "yarn.lock" ] && [ ! -d "node_modules" ]; then
   echo "Installing Yarn Dependencies"
   yarn
else 
   if [ -f "package.json" ] && [ ! -d "node_modules" ]; then
      echo "Installing NPM Dependencies"
      npm install
   fi
fi

COLOR_CYAN="\033[0;36m"
COLOR_RESET="\033[0m"

echo -e "${COLOR_CYAN}
   ____              ____
  |  _ \  _____   __/ ___| _ __   __ _  ___ ___
  | | | |/ _ \ \ / /\___ \| '_ \ / _\` |/ __/ _ \\
  | |_| |  __/\ V /  ___) | |_) | (_| | (_|  __/
  |____/ \___| \_/  |____/| .__/ \__,_|\___\___|
                          |_|
${COLOR_RESET}
Welcome to your development container!
This is how you can work with it:
- Run \`${COLOR_CYAN}npm start${COLOR_RESET}\` to start the application
- ${COLOR_CYAN}Files will be synchronized${COLOR_RESET} between your local machine and this container
- Use \`${COLOR_CYAN}ssh my-dev.quickstart.devspace${COLOR_RESET}\` to access the application via SSH
- Some ports will be forwarded, so you can access this container on your local machine via ${COLOR_CYAN}http://localhost:3000${COLOR_RESET}
"

bash