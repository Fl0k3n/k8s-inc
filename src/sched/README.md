# Scheduler plugins for telemetry

Scheduler code is stored in a fork of `scheduler-plugins`. This directory stores only config files. Clone [scheduler-plugins repo](https://github.com/Fl0k3n/scheduler-plugins) into `$GOPATH/src/sigs.k8s.io` 

## Building

Run `make build`, this will build docker image of scheduler with applied plugins using makefile from `scheduler-plugins`

## Installation

Assuming that kind is used, run `make install`, this will copy docker image to control-plane node, copy configurations and restart kubelet so that new scheduler is used.

## Development

Development of plugins should be perfomed in forked `scheduler-plugins` repo. To rebuild and reinstall scheduler in a running kind cluster run `make all`.

## Testing

The configuration file `config/kube-scheduler.yaml` uses multiple scheduling profiles, to schedule workloads using selected profile set pod `spec.schedulerName: <profile name>`, without that the default profile without any extra plugins (except the built-in ones) is used.
