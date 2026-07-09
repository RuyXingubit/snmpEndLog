package handlers

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nms-web/internal/db"
)

// Device represents a network device for templates.
type Device struct {
	ID              int
	Hostname        string
	IPAddress       string
	SNMPVersion     string
	Community       *string
	Status          string
	Vendor          *string
	SysName         *string
	SysDescr        *string
	SysLocation     *string
	SysUptime       *int64
	Voltage         *float64
	BoardName       *string
	SerialNumber    *string
	FirmwareVersion *string
	PollInterval    int
	PingEnabled     bool
	SnmpEnabled     bool
	LastPolledAt    *time.Time
	LastSeenAt      *time.Time
}

// InterfaceInfo represents a device interface for templates.
type InterfaceInfo struct {
	IfIndex      int
	IfDescr      *string
	IfAlias      *string
	IfSpeed      *int64
	IfAdminStatus int
	IfOperStatus  int
	InBps        *float64
	OutBps       *float64
	InErrors     *int64
	OutErrors    *int64
	VlanType     *string
	NativeVlan   *int
}

// BgpPeer represents a BGP peer for templates.
type BgpPeer struct {
	PeerAddr           string
	PeerAs             *int64
	State              *int
	AdminStatus        *int
	InUpdates          *int64
	OutUpdates         *int64
	PrefixesReceived   *int64
	PrefixesAdvertised *int64
	FsmEstablishedTime *int64
}

// HandleDevices renders the device list/management page.
func HandleDevices(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rows, err := db.Pool.Query(ctx, `
		SELECT id, hostname, host(ip_address), snmp_version, community, status,
		       vendor, sys_name, sys_descr, sys_location, sys_uptime,
		       board_name, serial_number, firmware_version, voltage,
		       poll_interval, ping_enabled, snmp_enabled, last_polled_at, last_seen_at
		FROM devices
		ORDER BY id DESC
	`)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var devices []Device
	for rows.Next() {
		var d Device
		err := rows.Scan(
			&d.ID, &d.Hostname, &d.IPAddress, &d.SNMPVersion, &d.Community, &d.Status,
			&d.Vendor, &d.SysName, &d.SysDescr, &d.SysLocation, &d.SysUptime,
			&d.BoardName, &d.SerialNumber, &d.FirmwareVersion, &d.Voltage,
			&d.PollInterval, &d.PingEnabled, &d.SnmpEnabled, &d.LastPolledAt, &d.LastSeenAt,
		)
		if err != nil {
			continue
		}
		devices = append(devices, d)
	}

	renderTemplate(w, "devices.html", map[string]interface{}{
		"Title":   "Equipamentos",
		"Devices": devices,
	}, r)
}

// HandleDeviceDetail renders a single device detail page.
func HandleDeviceDetail(w http.ResponseWriter, r *http.Request) {
	// Extract device ID from URL path: /devices/123 or /devices/123/edit
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/devices/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Redirect(w, r, "/devices", http.StatusSeeOther)
		return
	}

	// Route /devices/{id}/edit to the edit page
	if len(parts) >= 2 && parts[1] == "edit" {
		HandleDeviceEditPage(w, r)
		return
	}

	deviceID, err := strconv.Atoi(parts[0])
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Fetch device
	var d Device
	err = db.Pool.QueryRow(ctx, `
		SELECT id, hostname, host(ip_address), snmp_version, community, status,
		       vendor, sys_name, sys_descr, sys_location, sys_uptime,
		       board_name, serial_number, firmware_version, voltage,
		       poll_interval, ping_enabled, snmp_enabled, last_polled_at, last_seen_at
		FROM devices WHERE id = $1
	`, deviceID).Scan(
		&d.ID, &d.Hostname, &d.IPAddress, &d.SNMPVersion, &d.Community, &d.Status,
		&d.Vendor, &d.SysName, &d.SysDescr, &d.SysLocation, &d.SysUptime,
		&d.BoardName, &d.SerialNumber, &d.FirmwareVersion, &d.Voltage,
		&d.PollInterval, &d.PingEnabled, &d.SnmpEnabled, &d.LastPolledAt, &d.LastSeenAt,
	)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	// Fetch interfaces with latest traffic
	irows, err := db.Pool.Query(ctx, `
		SELECT i.if_index, i.if_descr, i.if_alias, i.if_speed,
		       COALESCE(i.if_admin_status, 1), COALESCE(i.if_oper_status, 1),
		       mt.in_bps, mt.out_bps, mt.in_errors, mt.out_errors,
               i.vlan_type, i.native_vlan
		FROM interfaces i
		LEFT JOIN LATERAL (
			SELECT in_bps, out_bps, in_errors, out_errors
			FROM metric_traffic
			WHERE device_id = i.device_id AND if_index = i.if_index
			ORDER BY time DESC LIMIT 1
		) mt ON TRUE
		WHERE i.device_id = $1
		ORDER BY i.if_index
	`, deviceID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer irows.Close()

	var interfaces []InterfaceInfo
	for irows.Next() {
		var iface InterfaceInfo
		err := irows.Scan(
			&iface.IfIndex, &iface.IfDescr, &iface.IfAlias, &iface.IfSpeed,
			&iface.IfAdminStatus, &iface.IfOperStatus,
			&iface.InBps, &iface.OutBps, &iface.InErrors, &iface.OutErrors,
            &iface.VlanType, &iface.NativeVlan,
		)
		if err != nil {
			continue
		}
		interfaces = append(interfaces, iface)
	}

	// Fetch BGP Peers
	brows, err := db.Pool.Query(ctx, `
		SELECT peer_addr, peer_as, state, admin_status,
		       in_updates, out_updates, prefixes_received, prefixes_advertised, fsm_established_time
		FROM bgp_peers
		WHERE device_id = $1
		ORDER BY peer_addr
	`, deviceID)
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer brows.Close()

	var bgpPeers []BgpPeer
	for brows.Next() {
		var p BgpPeer
		err := brows.Scan(
			&p.PeerAddr, &p.PeerAs, &p.State, &p.AdminStatus,
			&p.InUpdates, &p.OutUpdates, &p.PrefixesReceived, &p.PrefixesAdvertised, &p.FsmEstablishedTime,
		)
		if err == nil {
			bgpPeers = append(bgpPeers, p)
		}
	}

	renderTemplate(w, "device.html", map[string]interface{}{
		"Title":      d.Hostname,
		"Device":     d,
		"Interfaces": interfaces,
		"BgpPeers":   bgpPeers,
	}, r)
}

// HandleDeviceAdd processes the add device form (POST).
func HandleDeviceAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	hostname := strings.TrimSpace(r.FormValue("hostname"))
	ipAddress := strings.TrimSpace(r.FormValue("ip_address"))
	snmpVersion := r.FormValue("snmp_version")
	community := r.FormValue("community")
	pollInterval, _ := strconv.Atoi(r.FormValue("poll_interval"))
	if pollInterval <= 0 {
		pollInterval = 300
	}

	// Strip CIDR notation if present (e.g., "10.0.0.1/24" -> "10.0.0.1")
	if idx := strings.Index(ipAddress, "/"); idx != -1 {
		ipAddress = ipAddress[:idx]
	}

	snmpEnabled := r.FormValue("snmp_enabled") == "on"

	// Validate
	if hostname == "" || ipAddress == "" {
		http.Error(w, "Hostname e IP são obrigatórios", http.StatusBadRequest)
		return
	}

	if net.ParseIP(ipAddress) == nil {
		http.Error(w, "Endereço IP inválido", http.StatusBadRequest)
		return
	}

	if snmpVersion == "v2c" {
		_, err := db.Pool.Exec(ctx, `
			INSERT INTO devices (hostname, ip_address, snmp_version, community, poll_interval, snmp_enabled)
			VALUES ($1, $2::inet, $3, $4, $5, $6)
		`, hostname, ipAddress, snmpVersion, community, pollInterval, snmpEnabled)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao adicionar: %v", err), http.StatusBadRequest)
			return
		}
	} else {
		// SNMPv3
		v3User := r.FormValue("snmpv3_user")
		v3AuthProto := r.FormValue("snmpv3_auth_proto")
		v3AuthPass := r.FormValue("snmpv3_auth_pass")
		v3PrivProto := r.FormValue("snmpv3_priv_proto")
		v3PrivPass := r.FormValue("snmpv3_priv_pass")
		v3SecLevel := r.FormValue("snmpv3_sec_level")

		_, err := db.Pool.Exec(ctx, `
			INSERT INTO devices
				(hostname, ip_address, snmp_version, snmpv3_user,
				 snmpv3_auth_proto, snmpv3_auth_pass,
				 snmpv3_priv_proto, snmpv3_priv_pass, snmpv3_sec_level,
				 poll_interval, snmp_enabled)
			VALUES ($1, $2::inet, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		`, hostname, ipAddress, snmpVersion, v3User,
			v3AuthProto, v3AuthPass, v3PrivProto, v3PrivPass, v3SecLevel,
			pollInterval, snmpEnabled)
		if err != nil {
			http.Error(w, fmt.Sprintf("Erro ao adicionar: %v", err), http.StatusBadRequest)
			return
		}
	}

	http.Redirect(w, r, "/devices", http.StatusSeeOther)
}

// HandleDeviceDelete deletes a device (POST).
func HandleDeviceDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deviceID, err := strconv.Atoi(r.FormValue("device_id"))
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	_, err = db.Pool.Exec(ctx, "DELETE FROM devices WHERE id = $1", deviceID)
	if err != nil {
		http.Error(w, "Error deleting device", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/devices", http.StatusSeeOther)
}

// HandleDeviceEdit updates a device (POST).
func HandleDeviceEdit(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	deviceID, err := strconv.Atoi(r.FormValue("device_id"))
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	hostname := strings.TrimSpace(r.FormValue("hostname"))
	community := strings.TrimSpace(r.FormValue("community"))
	pollInterval, _ := strconv.Atoi(r.FormValue("poll_interval"))
	if pollInterval <= 0 {
		pollInterval = 300
	}
	pingEnabled := r.FormValue("ping_enabled") == "on"
	snmpEnabled := r.FormValue("snmp_enabled") == "on"

	if hostname == "" {
		http.Error(w, "Hostname é obrigatório", http.StatusBadRequest)
		return
	}

	_, err = db.Pool.Exec(ctx, `
		UPDATE devices SET
			hostname = $1,
			community = $2,
			poll_interval = $3,
			ping_enabled = $4,
			snmp_enabled = $5,
			updated_at = NOW()
		WHERE id = $6
	`, hostname, community, pollInterval, pingEnabled, snmpEnabled, deviceID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Erro ao atualizar: %v", err), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/devices/%d", deviceID), http.StatusSeeOther)
}

// HandleDeviceEditPage renders the device edit/settings page (GET).
func HandleDeviceEditPage(w http.ResponseWriter, r *http.Request) {
	// Parse device ID from URL: /devices/{id}/edit
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}
	deviceID, err := strconv.Atoi(parts[1])
	if err != nil {
		http.Error(w, "Invalid device ID", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var d Device
	err = db.Pool.QueryRow(ctx, `
		SELECT id, hostname, host(ip_address), snmp_version, community, status,
		       vendor, sys_name, sys_descr, sys_location, sys_uptime,
		       board_name, serial_number, firmware_version, voltage,
		       poll_interval, ping_enabled, snmp_enabled, last_polled_at, last_seen_at
		FROM devices WHERE id = $1
	`, deviceID).Scan(
		&d.ID, &d.Hostname, &d.IPAddress, &d.SNMPVersion, &d.Community, &d.Status,
		&d.Vendor, &d.SysName, &d.SysDescr, &d.SysLocation, &d.SysUptime,
		&d.BoardName, &d.SerialNumber, &d.FirmwareVersion, &d.Voltage,
		&d.PollInterval, &d.PingEnabled, &d.SnmpEnabled, &d.LastPolledAt, &d.LastSeenAt,
	)
	if err != nil {
		http.Error(w, "Device not found", http.StatusNotFound)
		return
	}

	renderTemplate(w, "device_edit.html", map[string]interface{}{
		"Title":  d.Hostname + " — Configurações",
		"Device": d,
	}, r)
}
