package handler

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/deepfence/ThreatMapper/deepfence_server/controls"
	"github.com/deepfence/ThreatMapper/deepfence_server/model"
	ctl "github.com/deepfence/golang_deepfence_sdk/utils/controls"
	"github.com/deepfence/golang_deepfence_sdk/utils/log"
)

func (h *Handler) GetAgentControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	var agentId model.AgentId

	err = json.Unmarshal(data, &agentId)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	actions, errs := controls.GetAgentActions(ctx, agentId.NodeId, agentId.AvailableWorkload)
	for _, err := range errs {
		if err != nil {
			log.Warn().Msgf("Cannot some actions for %s: %v, skipping", agentId.NodeId, err)
		}
	}

	res := ctl.AgentControls{
		BeatRateSec: 30,
		Commands:    actions,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Error().Msgf("Cannot marshal controls: %v", err)
		respondWith(ctx, w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)

	if err != nil {
		log.Error().Msgf("Cannot send controls: %v", err)
		w.WriteHeader(http.StatusGone)
		return
	}
}

func (h *Handler) GetAgentInitControls(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	var agentId model.InitAgentReq

	err = json.Unmarshal(data, &agentId)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	err = controls.CompleteAgentUpgrade(ctx, agentId.Version, agentId.NodeId)
	if err != nil {
		respondWith(ctx, w, http.StatusInternalServerError, err)
		return
	}

	actions, err := controls.GetPendingAgentScans(ctx, agentId.NodeId)
	if err != nil {
		log.Warn().Msgf("Cannot get actions: %s, skipping", err)
	}

	res := ctl.AgentControls{
		BeatRateSec: 30,
		Commands:    actions,
	}
	b, err := json.Marshal(res)
	if err != nil {
		log.Error().Msgf("Cannot marshal controls: %v", err)
		respondWith(ctx, w, http.StatusInternalServerError, err)
		return
	}

	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(b)

	if err != nil {
		log.Error().Msgf("Cannot send controls: %v", err)
		w.WriteHeader(http.StatusGone)
		return
	}
}

func (h *Handler) ScheduleAgentUpgrade(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	data, err := io.ReadAll(r.Body)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	var agentUp model.AgentUpgrade

	err = json.Unmarshal(data, &agentUp)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	url, err := controls.GetAgentVersionTarball(ctx, agentUp.Version)
	if err != nil {
		respondWith(ctx, w, http.StatusBadRequest, err)
		return
	}

	internal_req := ctl.StartAgentUpgradeRequest{
		HomeDirectoryUrl: url,
		Version:          agentUp.Version,
	}

	b, err := json.Marshal(internal_req)
	if err != nil {
		log.Error().Msg(err.Error())
		respondError(err, w)
		return
	}

	action := ctl.Action{
		ID:             ctl.StartAgentUpgrade,
		RequestPayload: string(b),
	}

	err = controls.ScheduleAgentUpgrade(ctx, agentUp.Version, []string{agentUp.NodeId}, action)
	if err != nil {
		log.Error().Msgf("Cannot schedule agent upgrade: %v", err)
		respondWith(ctx, w, http.StatusInternalServerError, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}
