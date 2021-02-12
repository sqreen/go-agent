#!/usr/bin/env bash
go tool nm $1 | grep _sqreen_hook_callback_ | wc -l