// Development seed for catalog's MongoDB content collection.
//
// The point of these three documents is that their `body` shapes have almost
// nothing in common. That is the justification for Mongo sitting alongside
// Postgres: modelling this relationally means a wide sparse table or an EAV
// schema, and modelling it in protobuf means a message that is mostly unset.
//
// Run: docker exec -i tf-mongo mongosh --quiet < content_seed.js

db = db.getSiblingDB('ticketflow');

// EventKind: 1=CONCERT 2=SPORTS 3=THEATRE 4=CONFERENCE

db.event_content.replaceOne(
  { _id: 'evt-arijit-mumbai' },
  {
    _id: 'evt-arijit-mumbai',
    kind: 1,
    body: {
      summary: 'An evening of playback classics and unreleased material.',
      doors_open: '18:30',
      support_acts: ['Shreya Ghoshal', 'Local Train'],
      setlist_preview: ['Tum Hi Ho', 'Channa Mereya', 'Kesariya'],
      age_restriction: '5+',
      merchandise: { available: true, stalls: 4 },
      accessibility: { wheelchair_bays: 24, hearing_loop: true }
    },
    updated_at: new Date()
  },
  { upsert: true }
);

db.event_content.replaceOne(
  { _id: 'evt-mi-vs-rcb' },
  {
    _id: 'evt-mi-vs-rcb',
    kind: 2,
    // Nothing here overlaps with the concert document above.
    body: {
      competition: 'Indian Premier League',
      match_number: 42,
      home: { name: 'Mumbai Indians', captain: 'Hardik Pandya', form: ['W', 'W', 'L', 'W', 'L'] },
      away: { name: 'Royal Challengers Bengaluru', captain: 'Rajat Patidar', form: ['L', 'W', 'W', 'W', 'W'] },
      pitch_report: 'Red soil, expected to favour stroke play under lights.',
      broadcast: { tv: ['Star Sports 1'], streaming: ['JioCinema'] },
      standings_snapshot: [
        { team: 'RCB', played: 9, points: 12 },
        { team: 'MI', played: 9, points: 10 }
      ]
    },
    updated_at: new Date()
  },
  { upsert: true }
);

db.event_content.replaceOne(
  { _id: 'evt-coldplay-mumbai' },
  {
    _id: 'evt-coldplay-mumbai',
    kind: 1,
    body: {
      summary: 'Music of the Spheres World Tour.',
      doors_open: '17:00',
      support_acts: ['Elyanna'],
      production_notes: { led_wristbands: true, confetti: true, runtime_minutes: 135 },
      sustainability: { kinetic_dancefloor: true, tree_per_ticket: true }
    },
    updated_at: new Date()
  },
  { upsert: true }
);

print('seeded ' + db.event_content.countDocuments() + ' content documents');
