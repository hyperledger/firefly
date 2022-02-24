BEGIN;
UPDATE events SET etype='bockchain_event' WHERE etype='bockchain_event_received';
COMMIT;