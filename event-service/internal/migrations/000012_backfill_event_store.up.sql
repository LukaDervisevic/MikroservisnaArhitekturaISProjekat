INSERT INTO event_service.event_store_events
    (event_id, aggregate_id, aggregate_type, version, event_type, payload, occurred_at)
SELECT
    gen_random_uuid(),
    e.id,
    'Event',
    1,
    'EventCreated',
    jsonb_build_object(
        'eventId', gen_random_uuid(),
        'aggregateId', e.id,
        'version', 1,
        'occurredAt', to_jsonb(now()),
        'name', e.name,
        'cotisationPrice', e.cotisation_price,
        'agenda', COALESCE(e.agenda, ''),
        'type', e.type,
        'dateTime', e.date_time,
        'locationId', e.location_id
    ),
    now()
FROM event_service.events e
WHERE NOT EXISTS (
    SELECT 1 FROM event_service.event_store_events s WHERE s.aggregate_id = e.id
);

SELECT setval('event_service.events_id_seq',
              GREATEST((SELECT COALESCE(MAX(id), 1) FROM event_service.events), 1));
