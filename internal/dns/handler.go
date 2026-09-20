package dns

import (
	"fmt"
	"net/http"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/httpx"
)

// Handler adapts the dns use-cases to the HTTP/JSON gateway (ADR 0019).
type Handler struct {
	svc *Service
}

// NewHandler wires the dns REST gateway over the given service.
func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// Routes returns the dns REST endpoints; credential connect/disconnect lives on the connectors surface, not here.
func (h *Handler) Routes() http.Handler {
	mux := httpx.NewServeMux()
	mux.HandleFunc("POST /api/dns/verify", h.verify)
	mux.HandleFunc("GET /api/dns/zones", h.listZones)
	mux.HandleFunc("GET /api/dns/zones/{zoneID}/records", h.listRecords)
	mux.HandleFunc("POST /api/dns/zones/{zoneID}/records", h.createRecord)
	mux.HandleFunc("PATCH /api/dns/zones/{zoneID}/records/{recordID}", h.updateRecord)
	mux.HandleFunc("DELETE /api/dns/zones/{zoneID}/records/{recordID}", h.deleteRecord)
	mux.HandleFunc("GET /api/dns/zones/{zoneID}/records/{recordID}/propagation", h.checkPropagation)
	mux.HandleFunc("POST /api/dns/instance-record", h.createInstanceRecord)
	mux.HandleFunc("GET /api/dns/service-hostnames", h.listServiceHostnames)
	mux.HandleFunc("GET /api/dns/service-hostnames/{service}", h.getServiceHostname)
	mux.HandleFunc("POST /api/dns/service-hostnames", h.setServiceHostname)
	mux.HandleFunc("DELETE /api/dns/service-hostnames/{service}", h.removeServiceHostname)
	mux.HandleFunc("POST /api/dns/tunnels", h.createTunnel)
	mux.HandleFunc("GET /api/dns/tunnels", h.listTunnels)
	mux.HandleFunc("GET /api/dns/tunnels/{tunnelID}", h.getTunnel)
	mux.HandleFunc("GET /api/dns/tunnels/{tunnelID}/status", h.tunnelStatus)
	mux.HandleFunc("POST /api/dns/tunnels/{tunnelID}/route", h.routeTunnelHostname)
	mux.HandleFunc("POST /api/dns/tunnels/{tunnelID}/rotate", h.rotateTunnelCredentials)
	mux.HandleFunc("POST /api/dns/tunnels/{tunnelID}/agent", h.provisionTunnelAgent)
	mux.HandleFunc("DELETE /api/dns/tunnels/{tunnelID}", h.deleteTunnel)
	mux.HandleFunc("POST /api/dns/reverse-proxy", h.provisionReverseProxy)
	mux.HandleFunc("POST /api/dns/gateways", h.createGateway)
	mux.HandleFunc("GET /api/dns/gateways", h.listGateways)
	mux.HandleFunc("GET /api/dns/gateways/{gatewayID}", h.getGateway)
	mux.HandleFunc("DELETE /api/dns/gateways/{gatewayID}", h.deleteGateway)
	mux.HandleFunc("POST /api/dns/exposures", h.createExposure)
	mux.HandleFunc("GET /api/dns/exposures", h.listExposures)
	mux.HandleFunc("GET /api/dns/exposures/{exposureID}", h.getExposure)
	mux.HandleFunc("DELETE /api/dns/exposures/{exposureID}", h.deleteExposure)
	mux.HandleFunc("GET /api/dns/services/{service}/exposures", h.exposuresForService)
	return mux
}

func (h *Handler) verify(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.VerifyCredentials(r.Context()); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *Handler) listZones(w http.ResponseWriter, r *http.Request) {
	zones, err := h.svc.ListZones(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, zones)
}

func (h *Handler) listRecords(w http.ResponseWriter, r *http.Request) {
	records, err := h.svc.ListRecords(r.Context(), r.PathValue("zoneID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, records)
}

func (h *Handler) createRecord(w http.ResponseWriter, r *http.Request) {
	var req RecordInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	rec, err := h.svc.CreateRecord(r.Context(), r.PathValue("zoneID"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, rec)
}

func (h *Handler) updateRecord(w http.ResponseWriter, r *http.Request) {
	var req RecordInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	rec, err := h.svc.UpdateRecord(r.Context(), r.PathValue("zoneID"), r.PathValue("recordID"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, rec)
}

func (h *Handler) deleteRecord(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteRecord(r.Context(), r.PathValue("zoneID"), r.PathValue("recordID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) checkPropagation(w http.ResponseWriter, r *http.Request) {
	records, err := h.svc.ListRecords(r.Context(), r.PathValue("zoneID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	var rec *Record
	for i := range records {
		if records[i].ID == r.PathValue("recordID") {
			rec = &records[i]
			break
		}
	}
	if rec == nil {
		httpx.WriteError(w, fmt.Errorf("%w: record %s not found in zone", apperrs.ErrNotFound, r.PathValue("recordID")))
		return
	}
	if err := h.svc.CheckPropagation(r.Context(), r.PathValue("zoneID"), *rec); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "propagated"})
}

type instanceRecordRequest struct {
	ZoneID string     `json:"zone_id"`
	Zone   string     `json:"zone"`
	Type   RecordType `json:"type"`
	Target string     `json:"target"`
}

func (h *Handler) createInstanceRecord(w http.ResponseWriter, r *http.Request) {
	var req instanceRecordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	rec, err := h.svc.CreateInstanceRecord(r.Context(), req.ZoneID, req.Zone, req.Type, req.Target)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, rec)
}

func (h *Handler) listServiceHostnames(w http.ResponseWriter, r *http.Request) {
	shs, err := h.svc.ListServiceHostnames(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, shs)
}

func (h *Handler) getServiceHostname(w http.ResponseWriter, r *http.Request) {
	sh, err := h.svc.GetServiceHostname(r.Context(), r.PathValue("service"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, sh)
}

func (h *Handler) setServiceHostname(w http.ResponseWriter, r *http.Request) {
	var req ServiceHostnameInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	sh, err := h.svc.SetServiceHostname(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, sh)
}

func (h *Handler) removeServiceHostname(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.RemoveServiceHostname(r.Context(), r.PathValue("service")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) createTunnel(w http.ResponseWriter, r *http.Request) {
	var req CreateTunnelInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	t, err := h.svc.CreateTunnel(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, t)
}

func (h *Handler) listTunnels(w http.ResponseWriter, r *http.Request) {
	tunnels, err := h.svc.ListTunnels(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, tunnels)
}

func (h *Handler) getTunnel(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetTunnel(r.Context(), r.PathValue("tunnelID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

// tunnelStatus is polled by onboarding until cloudflared reports in, so it reads Cloudflare live instead of the stored row.
func (h *Handler) tunnelStatus(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.TunnelStatus(r.Context(), r.PathValue("tunnelID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) routeTunnelHostname(w http.ResponseWriter, r *http.Request) {
	var req RouteTunnelInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	req.TunnelID = r.PathValue("tunnelID")
	t, err := h.svc.RouteTunnelHostname(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) rotateTunnelCredentials(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.RotateTunnelCredentials(r.Context(), r.PathValue("tunnelID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, t)
}

func (h *Handler) provisionTunnelAgent(w http.ResponseWriter, r *http.Request) {
	var req AgentSpec
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.ProvisionTunnelAgent(r.Context(), r.PathValue("tunnelID"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) provisionReverseProxy(w http.ResponseWriter, r *http.Request) {
	var req AgentSpec
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	p, err := h.svc.ProvisionReverseProxy(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, p)
}

func (h *Handler) deleteTunnel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteTunnel(r.Context(), r.PathValue("tunnelID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) createGateway(w http.ResponseWriter, r *http.Request) {
	var req CreateGatewayInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	g, err := h.svc.CreateGateway(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, g)
}

func (h *Handler) listGateways(w http.ResponseWriter, r *http.Request) {
	gws, err := h.svc.ListGateways(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, gws)
}

func (h *Handler) getGateway(w http.ResponseWriter, r *http.Request) {
	g, err := h.svc.GetGateway(r.Context(), r.PathValue("gatewayID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, g)
}

func (h *Handler) deleteGateway(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteGateway(r.Context(), r.PathValue("gatewayID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) createExposure(w http.ResponseWriter, r *http.Request) {
	var req CreateExposureInput
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	e, err := h.svc.CreateExposure(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusCreated, e)
}

func (h *Handler) listExposures(w http.ResponseWriter, r *http.Request) {
	exps, err := h.svc.ListExposures(r.Context())
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, exps)
}

func (h *Handler) getExposure(w http.ResponseWriter, r *http.Request) {
	e, err := h.svc.GetExposure(r.Context(), r.PathValue("exposureID"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, e)
}

func (h *Handler) deleteExposure(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteExposure(r.Context(), r.PathValue("exposureID")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) exposuresForService(w http.ResponseWriter, r *http.Request) {
	exps, err := h.svc.ExposuresForService(r.Context(), r.PathValue("service"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, exps)
}
