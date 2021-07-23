#!/bin/bash

rm -rf target
mkdir target
podman build -t uptimebuild .
podman run -u root --rm -v $(readlink -f target/):/uptime/target:Z uptimebuild
