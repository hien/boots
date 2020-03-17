package coreos

import (
	"fmt"
	"net/http"

	"github.com/packethost/tinkerbell/files/ignition"
	"github.com/packethost/tinkerbell/files/unit"
	"github.com/packethost/tinkerbell/installers"
	"github.com/packethost/tinkerbell/job"
)

func init() {
	http.HandleFunc("/coreos/ignition.json", serveIgnitionConfig("coreos"))
	http.HandleFunc("/flatcar/ignition.json", serveIgnitionConfig("flatcar"))
}

func buildNetworkUnits(j job.Job) (nu ignition.NetworkUnits) {
	configureBondDevUnit(j, nu.Add("00-bond.netdev"))
	configureNetworkUnit(j, nu.Add("00-bond.network"))

	for i, port := range j.Interfaces() {
		filename := fmt.Sprintf("%02d-nic%d.network", i+1, i)
		u := unit.New(filename)
		if ok := configureBondSlaveUnit(j, u, port); ok {
			nu.Append(u)
		}
	}
	return
}

func buildSystemdUnits(j job.Job) (su ignition.SystemdUnits) {
	configureInstaller(j, su.Add("install.service"))
	return
}

func serveIgnitionConfig(distro string) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		j, err := job.CreateFromRemoteAddr(req.RemoteAddr)
		if err != nil {
			installers.Logger(distro).With("client", req.RemoteAddr).Error(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		c := ignition.Config{
			Network: buildNetworkUnits(j),
			Systemd: buildSystemdUnits(j),
		}
		if err := c.Render(w); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			j.Error(err, "unable to render ignition config")
		}
	}
}
