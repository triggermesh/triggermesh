
FROM mcr.microsoft.com/vscode/devcontainers/go

RUN sudo apt install wget

RUN wget https://github.com/microsoft/vscode-dev-containers/blob/main/script-library/kubectl-helm-debian.sh

COPY ./dev-script.sh /tmp/library-scripts/
RUN apt-get update && bash /tmp/library-scripts/dev-script.sh
