# Scheduler plugins for telemetry

Some manual work is needed to make it work as we didn't include full `scheduler-plugins` repo here.

## Source preparation:

1. Clone [scheduler-plugins repo](https://github.com/kubernetes-sigs/scheduler-plugins) into `$GOPATH/src/sigs.k8s.io` (tested with scheduler-plugins v0.27.8, last commit hash `cd3e4fb`)
2. In this directory run `make copy-to-scheduler`, it will copy source code of plugins into that repository
3. Include plugins in the main function of scheduler:
```go
     command := app.NewSchedulerCommand(app.WithPlugin(internaltelemetry.Name, internaltelemetry.New))
```

## Building

Run `make build`, this will build docker image of scheduler with applied plugins using makefile from `scheduler-plugins`

## Installation

Assuming that kind is used, run `make install`, this will copy docker image to control-plane node, copy configurations and restart kubelet so that new scheduler is used.

## Development

In general development of plugins should be perfomed in cloned `scheduler-plugins` repo, copy plugin sources back here when done. To rebuild and reinstall scheduler in running kind cluster run `make all`.

## Testing

The configuration file `config/kube-scheduler.yaml` uses multiple scheduling profiles, to schedule workloads using selected profile set pod `spec.schedulerName: <profile name>`, without that the default profile without any extra plugins (except those built-in ones) is used.
