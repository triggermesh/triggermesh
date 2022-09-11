
FROM mcr.microsoft.com/vscode/devcontainers/go

RUN sudo apt install curl

COPY ./dev-script.sh /tmp/library-scripts/
RUN apt-get update && bash /tmp/library-scripts/dev-script.sh
