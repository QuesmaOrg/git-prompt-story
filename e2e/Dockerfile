FROM golang:1.24-alpine

# Install dependencies including libfaketime for clock manipulation
RUN apk add --no-cache git bash jq yq libfaketime

WORKDIR /build
COPY . .
RUN go build -o /usr/local/bin/git-prompt-story .

WORKDIR /workspace
RUN git config --global user.email "test@test.com" && \
    git config --global user.name "Test User" && \
    git config --global init.defaultBranch main

COPY e2e/ /e2e/
ENTRYPOINT ["/e2e/run-tests.sh"]
