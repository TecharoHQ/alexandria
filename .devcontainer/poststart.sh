#!/usr/bin/env bash

sudo chown -R vscode:vscode /workspace/alexandria/node_modules

go mod download &
npm ci &

wait
