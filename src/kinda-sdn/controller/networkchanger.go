package controller

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Fl0k3n/k8s-inc/kinda-sdn/model"
)

type DeviceItem struct {
	Name string `json:"name"`
	DeviceType string `json:"deviceType"`
	Links []string `json:"links"`
}

type AddDevicesRequest struct {
	Devices []DeviceItem `json:"devices"`
}

type ChangeProgramRequest struct {
	DeviceName string `json:"deviceName"`
	ProgramName string `json:"programName"`
}

type BasicResponse struct {
	Message string `json:"message"`
}

func sendBasicResponse(w http.ResponseWriter, statusCode int, message string) {
	w.WriteHeader(statusCode)
	w.Header().Set("Content-Type", "application/json")
	resp := BasicResponse{Message: message}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		fmt.Printf("Failed to send response %v\n", err)
	}
}

func (k *KindaSdn) AddDevicesHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)		
		return
	}
	body := AddDevicesRequest{}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		sendBasicResponse(w, http.StatusBadRequest, "Failed to parse")
		return
	}
	if len(body.Devices) == 0 {
		sendBasicResponse(w, http.StatusBadRequest, "Requires at least 1 device to add")
		return
	}
	devices := []model.Device{}
	for _, dev := range body.Devices {
		baseDev := model.BaseDevice{
			Name: dev.Name,
			Links: make([]*model.Link, len(dev.Links)),
		}
		for i, target := range dev.Links {
			baseDev.Links[i] = &model.Link{
				To: target,
				MacAddr: "00:00:00:00:00:00",
				Ipv4: "0.0.0.0",
				Mask: 0,
			}
		}
		switch dev.DeviceType {
		case model.NET:
			devices = append(devices, &model.NetDevice{
				BaseDevice: baseDev,
			})
		case model.INC_SWITCH:
			devices = append(devices, &model.IncSwitch{
				BaseDevice: baseDev,
				Arch: model.BMv2,
				GrpcUrl: "0.0.0.0:0",
				InstalledProgram: "forward",
				AllowClientProgrammability: false,
			})
		default:
			sendBasicResponse(w, http.StatusBadRequest, fmt.Sprintf("invalid device type %s", dev.DeviceType))
			return
		}
	}
	if err := k.AddDevices(devices); err != nil {
		sendBasicResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	sendBasicResponse(w, http.StatusOK, "Devices added")
}

func (k *KindaSdn) ChangeProgramHandler(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)		
		return
	}
	body := ChangeProgramRequest{}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		sendBasicResponse(w, http.StatusBadRequest, "Failed to parse")
		return
	}
	if err := k.ChangeProgram(body.DeviceName, body.ProgramName); err != nil {
		sendBasicResponse(w, http.StatusBadRequest, err.Error())
		return
	}
	sendBasicResponse(w, http.StatusOK, "Program changed")
}
