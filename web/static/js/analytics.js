// Product usage is collected from server-confirmed, aggregate receipts only.
// Private URLs, identifiers, names, card values and estimates are never sent.
(() => {
  if (window.pokerAnalytics) return;
  const measurementId = document.currentScript?.dataset.measurementId || '';
  if (location.hostname !== 'pokerplanning.net' || !/^G-[A-Z0-9]+$/.test(measurementId)) return;
  const roomPage = location.pathname.startsWith('/room/');
  let referrer = '';
  try {
    const source = new URL(document.referrer);
    if (source.protocol === 'https:' || source.protocol === 'http:') referrer = source.origin + '/';
  } catch {}
  const page = {
    page_location: 'https://pokerplanning.net' + (roomPage ? '/room/' : '/'),
    page_title: roomPage ? 'Planning Poker room' : 'Online Planning Poker',
    page_referrer: referrer,
    content_group: roomPage ? 'room' : 'landing',
  };
  const enums = {
    participant_role: new Set(['voter', 'spectator']),
    member_type: new Set(['visitor', 'host', 'participant', 'system']),
    deck_type: new Set(['modified-fibonacci', 'fibonacci', 't-shirt', 'custom']),
    auto_reveal: new Set(['enabled', 'disabled']),
    permission_mode: new Set(['open', 'host_only', 'mixed']),
    reveal_method: new Set(['manual', 'automatic']),
    vote_action: new Set(['initial', 'change']),
    round_source: new Set(['creation', 'next_round']),
    share_method: new Set(['copy_link']),
  };
  const counts = new Set(['participant_count', 'connected_count', 'voter_count', 'spectator_count', 'vote_count', 'round_number']);
  const serverEvents = new Set(['room_created', 'room_joined', 'round_started', 'round_revealed', 'round_reset', 'vote_cast', 'room_settings_changed']);
  const events = new Set([...serverEvents, 'room_connected', 'room_share', 'share_open', 'connection_lost', 'connection_restored']);
  const pendingKey = 'pp-analytics-receipts';
  let context = {}, connected = false, welcomed = false, leaving = false;
  window.addEventListener('pagehide', () => { leaving = true; });
  window.addEventListener('pageshow', () => { leaving = false; });
  const safe = parameters => {
    const result = {};
    for (const [key, value] of Object.entries(parameters || {})) {
      if (Object.hasOwn(enums, key) && enums[key].has(value)) result[key] = value;
      if (counts.has(key) && Number.isInteger(value) && value >= 0 && value <= 100000) result[key] = value;
    }
    return result;
  };
  let debug = false;
  try { debug = sessionStorage.getItem('site-analytics-debug') === '1'; } catch {}
  window.dataLayer = window.dataLayer || [];
  window.gtag = function () { window.dataLayer.push(arguments); };
  window.gtag('js', new Date());
  // Set globally too, so automatically generated engagement events inherit the
  // same anonymous page identity, including after a private room is renamed.
  window.gtag('set', page);
  window.gtag('config', measurementId, {...page, allow_google_signals: false,
    allow_ad_personalization_signals: false, ...(debug ? {debug_mode: true} : {})});
  const tag = document.createElement('script');
  tag.async = true;
  tag.src = 'https://www.googletagmanager.com/gtag/js?id=' + measurementId;
  document.head.append(tag);

  function send(name, parameters) {
    if (!events.has(name)) return;
    window.gtag('event', name, {...safe(parameters), ...page, value: 1, ...(debug ? {debug_mode: true} : {})});
  }
  window.pokerAnalytics = {track: (name, parameters = {}) => send(name, {...context, ...parameters})};
  function receipts(values) {
    return Array.isArray(values) ? values.filter(e => e && serverEvents.has(e.name))
      .slice(0, 3).map(e => ({name: e.name, parameters: safe(e.parameters)})) : [];
  }
  // HX-Trigger precedes HX-Redirect. Carry its safe receipt to the destination
  // page instead of relying on a tag request to finish during navigation.
  document.addEventListener('pp:usage', event => {
    const pending = receipts(event.detail?.events);
    try { sessionStorage.setItem(pendingKey, JSON.stringify(pending)); }
    catch { pending.forEach(e => send(e.name, e.parameters)); }
  });
  try {
    const pending = sessionStorage.getItem(pendingKey);
    sessionStorage.removeItem(pendingKey);
    if (pending) receipts(JSON.parse(pending)).forEach(e => send(e.name, e.parameters));
  } catch {}
  document.addEventListener('htmx:wsAfterMessage', event => {
    let message;
    try { message = JSON.parse(event.detail.message); } catch { return; }
    const payload = message.payload || {};
    if (message.type === 'usage_event') {
      receipts([payload]).forEach(e => send(e.name, e.parameters));
    } else if (message.type === 'room_state') {
      const participants = Array.isArray(payload.participants) ? payload.participants : [];
      const self = participants.find(p => p.ID === payload.currentParticipantId);
      context = safe({participant_role: self?.Role,
        member_type: self ? (payload.isCreator ? 'host' : 'participant') : 'visitor',
        participant_count: participants.length, connected_count: participants.filter(p => p.Connected).length,
        voter_count: participants.filter(p => p.Role === 'voter').length,
        spectator_count: participants.filter(p => p.Role === 'spectator').length,
        round_number: payload.roundNumber});
      if (!welcomed && self) { send('room_connected', context); welcomed = true; }
    }
  });
  document.addEventListener('htmx:wsOpen', () => {
    if (welcomed && !connected) send('connection_restored', context);
    connected = true;
  });
  document.addEventListener('htmx:wsClose', () => {
    if (connected && !leaving) send('connection_lost', context);
    connected = false;
  });
})();
