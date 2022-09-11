
FROM mcr.microsoft.com/vscode/devcontainers/go

RUN sudo apt install curl

RUN curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
RUN unzip awscliv2.zip
RUN ./aws/install

RUN curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-cli-401.0.0-linux-x86_64.tar.gz
RUN tar -xf google-cloud-cli-401.0.0-linux-x86_64.tar.gz
RUN ./google-cloud-sdk/install.sh

COPY ./dev-script.sh /tmp/library-scripts/
RUN apt-get update && bash /tmp/library-scripts/dev-script.sh
