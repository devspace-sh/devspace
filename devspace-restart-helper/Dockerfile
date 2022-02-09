# Development & Test Environment
ARG BASEIMAGE=alpine:latest
FROM ${BASEIMAGE}

ARG WORKING_DIR=/.devspace
ENV WORKING_DIR=${WORKING_DIR}

# Create project directory (workdir)
WORKDIR ${WORKING_DIR}

# Path
ENV PATH="${WORKING_DIR}:${PATH}"

RUN apk add --no-cache build-base git psmisc mandoc man-pages mandoc-apropos less less-doc || apt-get update && apt-get -y install gcc git; rm -rf /var/lib/apt/lists/*

# Copy devspace-restart-helper script
ADD devspace-restart-helper.sh devspace-restart-helper.sh
RUN chmod +x devspace-restart-helper.sh

# Copy process.sh
ADD test/process.sh process.sh
RUN chmod +x process.sh

ENTRYPOINT ["devspace-restart-helper.sh"]
CMD ["process.sh", "arg1", "-arg2=1", "--arg3"]