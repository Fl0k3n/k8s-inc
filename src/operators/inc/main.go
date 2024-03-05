/*
Copyright 2024.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	incv1alpha1 "operators/api/v1alpha1"
	"operators/controllers"
	"operators/services/devicecon"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(incv1alpha1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func oldmain() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "fe33898a.kntp.com",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.IncSwitchReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "IncSwitch")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func main() {
	con := devicecon.NewP4RuntimeConnector("127.0.0.1:9560", 0)
	logger := zap.New(zap.UseDevMode(true))
	logCtx := log.IntoContext(context.Background(), logger) 
	ctx, cancel := context.WithTimeout(logCtx, 2 * time.Second)
	defer cancel()
	err := con.Open(ctx)
	if err != nil {
		fmt.Println("failed")
		fmt.Println(err)
		return
	} 
	defer con.Close()

	fmt.Println("connected")
	binPath := "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v4.0/int4.json"
	p4infoPath := "/home/flok3n/develop/virtual/telemetry2/int-platforms/p4src/int_v4.0/int4.txt"
		
	err = con.InstallProgram(logCtx, binPath, p4infoPath)
	// err = con.UseInstalledProgram(p4infoPath); _ = binPath
	if err != nil {
		fmt.Println("Failed to install program")
		fmt.Println(err)
		return
	}

	fmt.Println("installed program")
	// a, e := con.BuildActionEntry("ingress.Forward.reply_arp", map[devicecon.ActionParamName]string{
	// 	"targetMac": "00:10:0a:00:AF:00",
	// })
	a, e := con.BuildActionEntry("configure_source", map[devicecon.ActionParamName]string{
		"max_hop": "4",
		"hop_metadata_len": "10",
		"ins_cnt": "8",
		"ins_mask": "65280",
	})
	if e != nil {
		fmt.Println("Convert failed")
		fmt.Println(e)
	} else {
		fmt.Println(a)
	}

	m, e := con.BuildMatchEntry("tb_int_source", map[devicecon.MatchFieldName]string{
		"hdr.ipv4.srcAddr": "10.10.0.2&&&0xFFFFFFFF",
		"hdr.ipv4.dstAddr": "10.10.3.2&&&0xFFFFFFFF",
		"meta.layer34_metadata.l4_src": "0x11FF&&&0xFFFF",
		"meta.layer34_metadata.l4_dst": "0x22FF&&&0xFFFF",
	})
	if e != nil {
		fmt.Println("Convert failed")
		fmt.Println(e)
	} else {
		fmt.Println(m)
	}

	e = con.WriteMatchActionEntry(logCtx, "tb_int_source", m, a)
	if e != nil {
		fmt.Println("failed to add entry")
		if devicecon.IsEntryExistsError(e) {
			fmt.Println("entry exists")
		} else {
			if re, ok := devicecon.AsRpcError(e); ok {
				fmt.Println(re.Error())
			} else {
				panic(e)
			}
		}
	} else {
		fmt.Println("OK ADDED")
	}
}
