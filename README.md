# k8s-inc

Extending Kubernetes with in-network computing.

- [src/sdn-shim](src/sdn-shim/) - contains implementation of SDN shim Kubernetes controller that exposes and manages network information in the cluster
- [src/inc-operator](src/inc-operator/) - contains controllers for in-network telemetry (auto configuration of monitoring of traffic between deployments)
- [src/kinda-sdn](src/kinda-sdn/) - constains implementation of SDN controller and plugin for in-network telemetry P4 application for BMv2 virtual switches
- [src/sched](src/sched/) - contains configuration files for installing in-network telemetry Kubernetes scheduler plugin, source code of scheduler is in [separate repo](https://github.com/Fl0k3n/scheduler-plugins)
- [src/proto](src/proto/) - constains gRPC API definitions for SDN controller API and in-network telemetry plugin API
- [src/libs](src/libs/) - constains tools for configuring BMv2 virtual switches used by the SDN controller
- [src/eval](src/eval/) - constains evaluation tools used in the thesis and in-network telemetry collector


Refer to individual READMEs and Makefiles for building and running instructions.
