-- we define some OIDs we would like to poll
INSERT INTO metrics (id, active, name, oid, description, export_as_label) VALUES
(1, true, 'sysDescr', '.1.3.6.1.2.1.1.1.0', 'A textual description of the entity. This value should include the full name and version of the system.', true),
(2, true, 'sysUpTime', '.1.3.6.1.2.1.1.3.0', 'The system uptime in hundredths of seconds.', false),
(4, true, 'sysLocation', '.1.3.6.1.2.1.1.6.0', 'The physical location of this node.', true),
(3, true, 'sysName', '.1.3.6.1.2.1.1.5.0', 'An administratively-assigned name for this managed node. By convention, it''s the node''s fqdn.', true),
(5, true, 'ifNumber', '.1.3.6.1.2.1.2.1.0', 'The number of network interfaces (regardless of their current state) present on this system.', false),
(6, true, 'ifIndex', ' .1.3.6.1.2.1.2.2.1.1', 'A unique index value for each interface.', false),
(7, true, 'ifName', '.1.3.6.1.2.1.31.1.1.1.1', 'The name of the interface.', true),
(8, true, 'ifDescr', '.1.3.6.1.2.1.2.2.1.2', 'The description of the interface.', true),
(9, true, 'ifAdminStatus', '.1.3.6.1.2.1.2.2.1.7', 'The administrative state of the interface. Possible values: up(1), down(2), testing(3).', false),
(10, true, 'ifOperStatus', '.1.3.6.1.2.1.2.2.1.8', 'The operational state of the interface. Possible values: up(1), down(2), testing(3), unknown(4), dormant(5), notPresent(6).', false),
(11, true, 'ifInOctets', '.1.3.6.1.2.1.2.2.1.10', 'The total number of octets received on the interface, including framing characters.', false),
(12, true, 'ifInDiscards', '.1.3.6.1.2.1.2.2.1.13', 'The number of inbound packets which were discarded.', false),
(13, true, 'ifInErrors', '.1.3.6.1.2.1.2.2.1.14', 'The number of inbound packets that contain errors preventing them from being deliverable.', false),
(14, true, 'ifOutOctets', '.1.3.6.1.2.1.2.2.1.16', 'The total number of octets transmitted out of the interface, including framing characters.', false),
(15, true, 'ifOutDiscards', '.1.3.6.1.2.1.2.2.1.19', 'The number of outbound packets which were chosen to be discarded.', false),
(16, true, 'ifOutErrors', '.1.3.6.1.2.1.2.2.1.20', 'The number of outbound packets that could not be transmitted because of errors.', false);

-- we define some measures to group metrics
INSERT INTO measures (id, name, description, is_indexed, index_metric_id) VALUES
(1, 'sysInfo', 'system info: description, uptime, name, location and number of interfaces', false, NULL),
(2, 'ifStatus', 'The administrative and operational status of each network interface.', true, 6),
(3, 'ifInCounters', 'Incoming packet counters for each interface.', true, 6),
(4, 'ifOutCounters', 'Outgoing packet counters for each interface.', true, 6);

-- we define a generic switch profile
INSERT INTO profiles (id, category, vendor, model) VALUES
(1, 'SWITCH', 'GENERIC', 'GENERIC');

-- we associate the measures to the profiles
INSERT INTO profile_measures (profile_id, measure_id) VALUES
(1, 1),
(1, 2),
(1, 3);

-- we associate the metrics to the measures
INSERT INTO measure_metrics (measure_id, metric_id) VALUES
(1, 1),
(1, 2),
(1, 3),
(1, 4),
(1, 5),
(2, 6),
(2, 7),
(2, 8),
(2, 9),
(2, 10),
(3, 6),
(3, 11),
(3, 12),
(3, 13),
(4, 6),
(4, 14),
(4, 15),
(4, 16);
