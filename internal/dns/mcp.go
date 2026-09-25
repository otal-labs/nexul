package dns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	apperrs "github.com/otal-labs/nexul/internal/platform/errors"
	"github.com/otal-labs/nexul/internal/platform/mcptool"
)

// MCPTools returns the DNS record, gateway, exposure, and tunnel tools; Cloudflare credentials come from connectors.
func MCPTools(s *Service) []mcptool.Tool {
	tools := slices.Concat(recordTools(s), gatewayTools(s), exposureTools(s), tunnelTools(s))
	for i := range tools {
		tools[i].Call = explainNotConnected(tools[i].Call)
	}
	return tools
}

// explainNotConnected names the fix for a missing Cloudflare connection, which the adapter would hide as internal.
func explainNotConnected(call func(context.Context, json.RawMessage) (any, error)) func(context.Context, json.RawMessage) (any, error) {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		out, err := call(ctx, args)
		if errors.Is(err, ErrCloudflareNotConnected) {
			return nil, fmt.Errorf("%w: Cloudflare is not connected to this instance; an owner connects it in Settings, then retry",
				apperrs.ErrInvalid)
		}
		return out, err
	}
}

// listedBy points a not-found error at the tool that lists valid ids; the sentinel stays for the adapter.
func listedBy(err error, lister string) error {
	if errors.Is(err, apperrs.ErrNotFound) {
		return fmt.Errorf("%w; %s", err, lister)
	}
	return err
}

// overlay applies one patch field: a field the caller omitted keeps its current value.
func overlay[T any](dst *T, v *T) {
	if v != nil {
		*dst = *v
	}
}

func shapeAll[T, R any](items []T, shape func(T) R) []R {
	out := make([]R, 0, len(items))
	for _, item := range items {
		out = append(out, shape(item))
	}
	return out
}

// recordResult is a record as the model reads it; Propagated is set only when the caller asked for the check.
type recordResult struct {
	ID         string     `json:"id"`
	ZoneID     string     `json:"zone_id"`
	Type       RecordType `json:"type"`
	Name       string     `json:"name"`
	Content    string     `json:"content"`
	TTL        int        `json:"ttl"`
	Proxied    bool       `json:"proxied"`
	Propagated *bool      `json:"propagated,omitempty"`
}

func toRecordResult(r Record) recordResult {
	return recordResult{ID: r.ID, ZoneID: r.ZoneID, Type: r.Type, Name: r.Name, Content: r.Content, TTL: r.TTL, Proxied: r.Proxied}
}

type zoneListIn struct {
	mcptool.PageArgs
}

type recordListIn struct {
	ZoneID           string     `json:"zone_id" jsonschema:"The zone's id, from dns_zone_list."`
	ID               string     `json:"id,omitempty" jsonschema:"Only the record with this id."`
	Name             string     `json:"name,omitempty" jsonschema:"Only records with this full name, for example api.example.com."`
	Type             RecordType `json:"type,omitempty" jsonschema:"Only records of this type: A, AAAA, CNAME, or TXT."`
	CheckPropagation bool       `json:"check_propagation,omitempty" jsonschema:"Also resolve each returned record through public DNS and set propagated. Defaults to false."`
	mcptool.PageArgs
}

func (in recordListIn) matches(r Record) bool {
	if in.ID != "" && r.ID != in.ID {
		return false
	}
	if in.Name != "" && !strings.EqualFold(strings.TrimSuffix(r.Name, "."), strings.TrimSuffix(in.Name, ".")) {
		return false
	}
	return in.Type == "" || r.Type == in.Type
}

type recordCreateIn struct {
	ZoneID  string     `json:"zone_id" jsonschema:"The zone's id, from dns_zone_list."`
	Type    RecordType `json:"type" jsonschema:"The record type: A, AAAA, CNAME, or TXT."`
	Name    string     `json:"name" jsonschema:"The record name, relative to the zone or in full, for example api or api.example.com; @ is the zone apex."`
	Content string     `json:"content" jsonschema:"What the record answers: an IPv4 address for A, an IPv6 address for AAAA, a hostname for CNAME, text for TXT."`
	TTL     int        `json:"ttl,omitzero" jsonschema:"Time to live in seconds, 60 to 86400. Defaults to 1, the provider's automatic TTL."`
	Proxied bool       `json:"proxied,omitempty" jsonschema:"Route traffic through Cloudflare's proxy. Defaults to false; a CNAME to a tunnel (*.cfargotunnel.com) resolves only when proxied."`
}

type recordUpdateIn struct {
	ZoneID  string      `json:"zone_id" jsonschema:"The zone's id, from dns_zone_list."`
	ID      string      `json:"id" jsonschema:"The record's id, from dns_record_list."`
	Type    *RecordType `json:"type,omitempty" jsonschema:"New record type: A, AAAA, CNAME, or TXT. Omit to keep it."`
	Name    *string     `json:"name,omitempty" jsonschema:"New record name, for example api.example.com. Omit to keep it."`
	Content *string     `json:"content,omitempty" jsonschema:"New content, for example 203.0.113.10. Omit to keep it."`
	TTL     *int        `json:"ttl,omitempty" jsonschema:"New time to live in seconds; 1 is automatic. Omit to keep it."`
	Proxied *bool       `json:"proxied,omitempty" jsonschema:"Whether Cloudflare's proxy serves the record. Omit to keep it."`
}

type recordDeleteIn struct {
	ZoneID string `json:"zone_id" jsonschema:"The zone's id, from dns_zone_list."`
	ID     string `json:"id" jsonschema:"The record's id, from dns_record_list."`
}

func recordTools(s *Service) []mcptool.Tool {
	return []mcptool.Tool{
		mcptool.New("dns_zone_list", "List DNS zones",
			"Lists the DNS zones the connected Cloudflare account can edit, with each zone's id and domain name. "+
				"Call it first: dns_record_list, dns_record_create, gateway_create, and exposure_create all take a zone "+
				"id and name from it. A successful call also confirms the Cloudflare credentials work; a missing or "+
				"rejected token returns an error instead.",
			mcptool.Hints{ReadOnly: true},
			func(ctx context.Context, in zoneListIn) (any, error) {
				zones, err := s.ListZones(ctx)
				if err != nil {
					return nil, err
				}
				return mcptool.Paginate(zones, in.PageArgs), nil
			}),
		mcptool.New("dns_record_list", "List DNS records",
			"Lists a zone's DNS records at Cloudflare, optionally narrowed by id, name, or type. With "+
				"check_propagation it also resolves each returned record through public DNS and sets propagated, "+
				"which is how to confirm a new or changed record is live; a proxied record never reports propagated, "+
				"because public DNS answers with Cloudflare's own addresses. Records an exposure or tunnel created "+
				"appear here too; change those through exposure_delete or dns_tunnel_update so Nexul's own view stays true.",
			mcptool.Hints{ReadOnly: true},
			func(ctx context.Context, in recordListIn) (any, error) {
				records, err := s.ListRecords(ctx, in.ZoneID)
				if err != nil {
					return nil, listedBy(err, "dns_zone_list lists zones")
				}
				var matched []Record
				for _, r := range records {
					if in.matches(r) {
						matched = append(matched, r)
					}
				}
				page := mcptool.Paginate(matched, in.PageArgs)
				out := mcptool.Page[recordResult]{Items: make([]recordResult, 0, len(page.Items)),
					Total: page.Total, HasMore: page.HasMore, NextOffset: page.NextOffset}
				for _, r := range page.Items {
					res := toRecordResult(r)
					if in.CheckPropagation {
						propagated := s.CheckPropagation(ctx, in.ZoneID, r) == nil
						res.Propagated = &propagated
					}
					out.Items = append(out.Items, res)
				}
				return out, nil
			}),
		mcptool.New("dns_record_create", "Create DNS record",
			"Creates an A, AAAA, CNAME, or TXT record in a zone at Cloudflare and returns it with its id. Use it for "+
				"records Nexul does not manage on its own, such as the instance's own hostname; to make a deployed "+
				"container reachable use exposure_create, which creates its record itself. It never replaces a record: "+
				"a name another record already holds fails or adds a second answer, and dns_record_update changes an "+
				"existing one.",
			mcptool.Hints{Additive: true},
			func(ctx context.Context, in recordCreateIn) (any, error) {
				ttl := in.TTL
				if ttl == 0 {
					ttl = 1
				}
				rec, err := s.CreateRecord(ctx, in.ZoneID, RecordInput{
					Type: in.Type, Name: in.Name, Content: in.Content, TTL: ttl, Proxied: in.Proxied,
				})
				if err != nil {
					return nil, listedBy(err, "dns_zone_list lists zones")
				}
				return toRecordResult(*rec), nil
			}),
		mcptool.New("dns_record_update", "Update DNS record",
			"Changes a DNS record at Cloudflare; only the fields you pass change, and every omitted field, proxied "+
				"included, keeps its current value. Find the record's id with dns_record_list. Returns the record as "+
				"stored after the change. A record an exposure or tunnel owns changes through exposure_delete and "+
				"exposure_create or dns_tunnel_update instead.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in recordUpdateIn) (any, error) {
				cur, err := s.GetRecord(ctx, in.ZoneID, in.ID)
				if err != nil {
					return nil, listedBy(err, "dns_record_list lists the zone's records")
				}
				patch := RecordInput{Type: cur.Type, Name: cur.Name, Content: cur.Content, TTL: cur.TTL, Proxied: cur.Proxied}
				overlay(&patch.Type, in.Type)
				overlay(&patch.Name, in.Name)
				overlay(&patch.Content, in.Content)
				overlay(&patch.TTL, in.TTL)
				overlay(&patch.Proxied, in.Proxied)
				rec, err := s.UpdateRecord(ctx, in.ZoneID, in.ID, patch)
				if err != nil {
					return nil, err
				}
				return toRecordResult(*rec), nil
			}),
		mcptool.New("dns_record_delete", "Delete DNS record",
			"Deletes a DNS record from a zone at Cloudflare and returns its id with deleted set. Deleting a record "+
				"that is already gone succeeds, so a repeat is safe. To take down a hostname an exposure routes, use "+
				"exposure_delete, which removes the record and the exposure together.",
			mcptool.Hints{Idempotent: true},
			func(ctx context.Context, in recordDeleteIn) (any, error) {
				if err := s.DeleteRecord(ctx, in.ZoneID, in.ID); err != nil {
					return nil, err
				}
				return mcptool.Gone(in.ID), nil
			}),
	}
}
