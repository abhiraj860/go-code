-- Development seed data for catalog-svc.
-- Idempotent: safe to re-run against an already-seeded database.

INSERT INTO venue (id, name, city, country_code, address, latitude, longitude) VALUES
    ('ven-wankhede',  'Wankhede Stadium',      'Mumbai',    'IN', 'Vinoo Mankad Rd, Churchgate',  18.9389, 72.8258),
    ('ven-nsci',      'NSCI Dome',             'Mumbai',    'IN', 'Lala Lajpat Rai Marg, Worli',  19.0008, 72.8206),
    ('ven-chinnaswamy','M. Chinnaswamy Stadium','Bengaluru','IN', 'MG Road, Bengaluru',           12.9788, 77.5996)
ON CONFLICT (id) DO NOTHING;

INSERT INTO seat_map (id, venue_id, viewbox_width, viewbox_height, version) VALUES
    ('map-nsci-main',     'ven-nsci',       1000, 800, 1),
    ('map-wankhede-main', 'ven-wankhede',   1600, 1600, 1)
ON CONFLICT (id) DO NOTHING;

-- A small but realistically shaped seat map: 3 sections x 4 rows x 10 seats.
-- Generated rather than written out, so the seed stays readable.
INSERT INTO seat (seat_map_id, id, section, row_label, seat_number, pricing_tier_id, x, y)
SELECT
    'map-nsci-main',
    section || '-' || row_label || '-' || seat_number,
    section,
    row_label,
    seat_number::text,
    CASE section WHEN 'A' THEN 'tier-vip' WHEN 'B' THEN 'tier-gold' ELSE 'tier-silver' END,
    (seat_number * 40)::real,
    (CASE section WHEN 'A' THEN 0 WHEN 'B' THEN 200 ELSE 400 END
        + (ascii(row_label) - ascii('A')) * 45)::real
FROM
    (VALUES ('A'), ('B'), ('C')) AS s(section),
    (VALUES ('A'), ('B'), ('C'), ('D')) AS r(row_label),
    generate_series(1, 10) AS seat_number
ON CONFLICT (seat_map_id, id) DO NOTHING;

INSERT INTO event (id, title, kind, status, venue_id, seat_map_id,
                   starts_at, ends_at, sale_opens_at, poster_url, tags, version) VALUES
    ('evt-arijit-mumbai', 'Arijit Singh Live in Mumbai', 1, 2, 'ven-nsci', 'map-nsci-main',
     now() + interval '30 days', now() + interval '30 days 3 hours',
     now() - interval '1 day', 'https://cdn.example/arijit.jpg',
     ARRAY['bollywood', 'live-music'], 1),

    ('evt-coldplay-mumbai', 'Coldplay: Music of the Spheres', 1, 1, 'ven-nsci', 'map-nsci-main',
     now() + interval '90 days', now() + interval '90 days 3 hours',
     now() + interval '7 days', 'https://cdn.example/coldplay.jpg',
     ARRAY['rock', 'live-music', 'international'], 1),

    ('evt-mi-vs-rcb', 'Mumbai Indians vs RCB', 2, 2, 'ven-wankhede', 'map-wankhede-main',
     now() + interval '14 days', now() + interval '14 days 4 hours',
     now() - interval '10 days', 'https://cdn.example/mivrcb.jpg',
     ARRAY['cricket', 'ipl'], 1),

    -- Cancelled and sold-out events exist so the browse query can be shown
    -- excluding them via the partial index.
    ('evt-cancelled-demo', 'Cancelled Show', 3, 4, 'ven-nsci', 'map-nsci-main',
     now() + interval '20 days', now() + interval '20 days 2 hours',
     now() - interval '5 days', '', ARRAY['theatre'], 1)
ON CONFLICT (id) DO NOTHING;

INSERT INTO pricing_tier (id, event_id, name, amount_minor, currency_code) VALUES
    ('tier-vip',    'evt-arijit-mumbai', 'VIP Front Row', 1500000, 'INR'),
    ('tier-gold',   'evt-arijit-mumbai', 'Gold',           750000, 'INR'),
    ('tier-silver', 'evt-arijit-mumbai', 'Silver',         350000, 'INR'),
    ('tier-cp-ga',  'evt-coldplay-mumbai', 'General Admission', 900000, 'INR'),
    ('tier-mi-std', 'evt-mi-vs-rcb', 'Standard',           250000, 'INR')
ON CONFLICT (id) DO NOTHING;
